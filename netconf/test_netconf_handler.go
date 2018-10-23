package netconf

import (
	"encoding/xml"
	"sync"
	"time"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

var (
	nameRPC = xml.Name{Space: netconfNS, Local: "rpc"}
)

// netconfSessionHandler represents the server side of an active netconf SSH session.
type netconfSessionHandler struct {
	// t is the testing context used for handling unexpected errors.
	t assert.TestingT

	// ch is the underlying transport connection.
	ch ssh.Channel

	// The codecs used to handle client i/o
	enc *encoder
	dec *decoder

	// Serialises access to encoder (avoiding contention between sending notifications and request responses).
	encLock sync.Mutex

	// The capabilities advertised to the client.
	capabilities []string
	// The session id to be reported to the client.
	sid int

	// Channel used to signal successful receipt of client capabilities.
	hellochan chan bool

	// The HelloMessage sent by the connecting client.
	ClientHello *HelloMessage

	// startwg will be signalled when the session is started (specifically after client
	// capabilities have been received).
	startwg *sync.WaitGroup

	// The queue of handlers used to process incoming client requests.
	// If the queue is empty, a request is processed by the EchoRequestHandler
	reqHandlers []RequestHandler

	// Counts the number of requests that have been received by the session.
	ReqCount int
}

// rpcRequestMessage and rpcRequest represent an RPC request from a client, where the element type of the
// request body is unknown.
type rpcRequestMessage struct {
	XMLName   xml.Name   //`xml:"rpc"`
	MessageID string     `xml:"message-id,attr"`
	Request   rpcRequest `xml:",any"`
}
type rpcRequest struct {
	XMLName xml.Name
	Body    string `xml:",innerxml"`
}

// RPCReplyMessage  and replyData represent an rpc-reply message that will be sent to a client session, where the
// element type of the reply body (i.e. the content of the data element) is unknown.
type RPCReplyMessage struct {
	XMLName   xml.Name   `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	Errors    []RPCError `xml:"rpc-error,omitempty"`
	Data      replyData  `xml:"data"`
	Ok        bool       `xml:",omitempty"`
	RawReply  string     `xml:"-"`
	MessageID string     `xml:"message-id,attr"`
}
type replyData struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",innerxml"`
}

// NotifyMessage defines the contents of a notification message that will be sent to a client session, where the
// element type of the notification event is unknown.
type NotifyMessage struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:netconf:notification:1.0 notification"`
	EventTime string   `xml:"eventTime"`
	Data      string   `xml:",innerxml"`
}

// RequestHandler is a function type that will be invoked by the session handler to handle an RPC
// request.
type RequestHandler func(h *netconfSessionHandler, req *rpcRequestMessage)

// EchoRequestHandler responds to a request with a reply containing a data element holding
// the body of the request.
var EchoRequestHandler = func(h *netconfSessionHandler, req *rpcRequestMessage) {
	data := replyData{Data: req.Request.Body}
	reply := &RPCReplyMessage{Data: data, MessageID: req.MessageID}
	err := h.encode(reply)
	assert.NoError(h.t, err, "Failed to encode response")
}

// FailingRequestHandler replies to a request with an error.
var FailingRequestHandler = func(h *netconfSessionHandler, req *rpcRequestMessage) {
	reply := &RPCReplyMessage{
		MessageID: req.MessageID,
		Errors: []RPCError{
			RPCError{Severity: "error", Message: "oops"}},
	}
	err := h.encode(reply)
	assert.NoError(h.t, err, "Failed to encode response")
}

// CloseRequestHandler closes the transport channel on request receipt.
var CloseRequestHandler = func(h *netconfSessionHandler, req *rpcRequestMessage) {
	h.ch.Close() // nolint: errcheck, gosec
}

// IgnoreRequestHandler does in nothing on receipt of a request.
var IgnoreRequestHandler = func(h *netconfSessionHandler, req *rpcRequestMessage) {}

func newSessionHandler(t assert.TestingT, sid int) *netconfSessionHandler { // nolint: deadcode
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &netconfSessionHandler{t: t,
		sid:          sid,
		hellochan:    make(chan bool),
		startwg:      wg,
		capabilities: DefaultCapabilities}
}

// Handle establishes a Netconf server session on a newly-connected SSH channel.
func (h *netconfSessionHandler) Handle(t assert.TestingT, ch ssh.Channel) {
	h.ch = ch
	h.dec = newDecoder(ch)
	h.enc = newEncoder(ch)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Send server hello to client.
	err := h.encode(&HelloMessage{Capabilities: h.capabilities, SessionID: h.sid})
	assert.NoError(h.t, err, "Failed to send server hello")

	go h.handleIncomingMessages(wg)

	h.waitForClientHello()

	// Signal server has completed setup
	h.startwg.Done()

	// Wait for message handling routine to finish.
	wg.Wait()
}

func (h *netconfSessionHandler) WaitStart() {
	h.startwg.Wait()
}

// SendNotification sends a notification message with the supplied body to the client.
func (h *netconfSessionHandler) SendNotification(body string) *netconfSessionHandler {
	nm := &NotifyMessage{EventTime: time.Now().String(), Data: body}
	err := h.encode(nm)
	assert.NoError(h.t, err, "Failed to send server notification")
	return h
}

// Close initiates session tear-down by closing the underlying transport channel.
func (h *netconfSessionHandler) Close() {
	h.ch.Close() // nolint: errcheck, gosec
}

func (h *netconfSessionHandler) waitForClientHello() {

	// Wait for the input handler to send the client hello.
	select {
	case <-h.hellochan:
	case <-time.After(time.Duration(5) * time.Second):
	}

	assert.NotNil(h.t, h.ClientHello, "Failed to get client hello")
}

func (h *netconfSessionHandler) handleIncomingMessages(wg *sync.WaitGroup) {

	defer wg.Done()

	// Loop, looking for a start element type of hello, rpc-reply.
	for {
		token, err := h.dec.Token()
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
	default:
	}
}

func (h *netconfSessionHandler) handleHello(token xml.StartElement) {
	// Decode the hello element and send it down the channel to trigger the rest of the session setup.

	h.decodeElement(&h.ClientHello, &token)

	if peerSupportsChunkedFraming(h.ClientHello.Capabilities) && peerSupportsChunkedFraming(h.capabilities) {

		// Update the codec to use chunked framing from now.
		enableChunkedFraming(h.dec, h.enc)
	}

	h.hellochan <- true
}

func (h *netconfSessionHandler) handleRPC(token xml.StartElement) {
	request := &rpcRequestMessage{}
	h.decodeElement(&request, &token)

	h.ReqCount++
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
		reqh = EchoRequestHandler
	} else {
		h.reqHandlers, reqh = h.reqHandlers[1:], h.reqHandlers[0]
	}
	return
}

func (h *netconfSessionHandler) encode(m interface{}) error {
	h.encLock.Lock()
	defer h.encLock.Unlock()

	return h.enc.encode(m)
}
