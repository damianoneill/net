package netconf

import (
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

var (
	nameRPC = xml.Name{Space: netconfNS, Local: "rpc"}
)

type RPCRequest struct {
	XMLName xml.Name
	Body    string `xml:",innerxml"`
}

// RPCRequestMessage defines the request sent to the server.
type RPCRequestMessage struct {
	XMLName   xml.Name   //`xml:"rpc"`
	MessageID string     `xml:"message-id,attr"`
	Request   RPCRequest `xml:",any"`
}

type ReplyData struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",innerxml"`
}

type RPCReplyMessage struct {
	XMLName   xml.Name   `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	Errors    []RPCError `xml:"rpc-error,omitempty"`
	Data      ReplyData  `xml:"data"`
	Ok        bool       `xml:",omitempty"`
	RawReply  string     `xml:"-"`
	MessageID string     `xml:"message-id,attr"`
}

type NotifyMessage struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:netconf:notification:1.0 notification"`
	EventTime string   `xml:"eventTime"`
	Data      string   `xml:",innerxml"`
}

type RequestHandler func(h *netconfSessionHandler, req *RPCRequestMessage)

var DefaultRequestHandler = func(h *netconfSessionHandler, req *RPCRequestMessage) {
	data := ReplyData{Data: req.Request.Body}
	reply := &RPCReplyMessage{Data: data, MessageID: req.MessageID}
	err := h.enc.encode(reply)
	assert.NoError(h.t, err, "Failed to encode response")
}

var CloseRequestHandler = func(h *netconfSessionHandler, req *RPCRequestMessage) {
	h.ch.Close()
}

var IgnoreRequestHandler = func(h *netconfSessionHandler, req *RPCRequestMessage) {

}

type netconfSessionHandler struct {
	t   assert.TestingT
	ch  ssh.Channel
	enc *encoder
	dec *decoder

	hellochan   chan bool
	clientHello *HelloMessage
	sid         int

	startwg *sync.WaitGroup

	reqHandlers []RequestHandler

	reqCount int
}

func newHandler(t assert.TestingT, sid int) *netconfSessionHandler {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &netconfSessionHandler{t: t, sid: sid, hellochan: make(chan bool), startwg: wg}
}

func (h *netconfSessionHandler) Handle(t assert.TestingT, ch ssh.Channel) {
	h.ch = ch
	h.dec = newDecoder(injectDiagReader(ch))
	h.enc = newEncoder(ch)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go h.handleIncomingMessages(wg)
	h.exchangeHelloMessages()
	h.startwg.Done()

	wg.Wait()

}

func (h *netconfSessionHandler) waitStart() {
	h.startwg.Wait()
}

type diagReader struct {
	r io.Reader
}

func (dr *diagReader) Read(p []byte) (int, error) {
	fmt.Printf("Server ReadStart %d\n", len(p))
	c, err := dr.r.Read(p)
	fmt.Printf("Server ReadDone %d %v %s\n", c, err, string(p[:c]))
	return c, err
}

func injectDiagReader(r io.Reader) io.Reader {
	return &diagReader{r: r}
}

func (h *netconfSessionHandler) withRequestHandler(rh RequestHandler) *netconfSessionHandler {
	h.reqHandlers = append(h.reqHandlers, rh)
	return h
}

func (h *netconfSessionHandler) sendNotification(n string) *netconfSessionHandler {
	nm := &NotifyMessage{EventTime: time.Now().String(), Data: n}
	err := h.enc.encode(nm)
	assert.NoError(h.t, err, "Failed to send server notification")
	return h
}

func (h *netconfSessionHandler) close() {
	h.ch.Close()
}

func (h *netconfSessionHandler) exchangeHelloMessages() (err error) {

	err = h.enc.encode(&HelloMessage{Capabilities: DefaultCapabilities, SessionID: h.sid})
	assert.NoError(h.t, err, "Failed to send server hello")

	// Wait for the input handler to send the client hello.
	select {
	case <-h.hellochan:
	case <-time.After(time.Duration(5) * time.Second):
	}

	assert.NotNil(h.t, h.clientHello, "Failed to get client hello")

	// if peerSupportsChunkedFraming(h.clientHello) {

	// 	// Update the codec to use chunked framing from now.
	// 	enableChunkedFraming(h.dec, h.enc)
	// 	fmt.Println("Server enabled chunks")
	// }

	return
}

func (h *netconfSessionHandler) handleIncomingMessages(wg *sync.WaitGroup) {
	fmt.Printf("Server - handleIncoming\n")
	defer wg.Done()

	// Loop, looking for a start element type of hello, rpc-reply.
	for {
		token, err := h.dec.Token()
		fmt.Printf("Server - decode err %v\n", err)
		if err != nil {
			break
		}
		h.handleToken(token)
	}
}

func (h *netconfSessionHandler) handleToken(token xml.Token) {
	switch token := token.(type) {
	case xml.StartElement:
		switch token.Name {
		case nameHello: // <hello>
			h.handleHello(token)

		case nameRPC: // <rpc>
			h.handleRPC(token)

		default:
		}
	}
	return
}

func (h *netconfSessionHandler) handleHello(token xml.StartElement) {
	// Decode the hello element and send it down the channel to trigger the rest of the session setup.

	h.decodeElement(&h.clientHello, &token)

	if peerSupportsChunkedFraming(h.clientHello) {

		// Update the codec to use chunked framing from now.
		enableChunkedFraming(h.dec, h.enc)
		fmt.Println("Server enabled chunks")
	}

	h.hellochan <- true
}

func (h *netconfSessionHandler) handleRPC(token xml.StartElement) {
	request := &RPCRequestMessage{}
	h.decodeElement(&request, &token)

	h.reqCount++
	reqh := h.nextReqHandler()
	reqh(h, request)
}

func (h *netconfSessionHandler) decodeElement(v interface{}, start *xml.StartElement) {
	err := h.dec.DecodeElement(v, start)
	assert.NoError(h.t, err, "DecodeElement failed")
}

func (h *netconfSessionHandler) nextReqHandler() (reqh RequestHandler) {
	l := len(h.reqHandlers)
	if l == 0 {
		reqh = DefaultRequestHandler
	} else {
		h.reqHandlers, reqh = h.reqHandlers[1:], h.reqHandlers[0]
	}
	return
}
