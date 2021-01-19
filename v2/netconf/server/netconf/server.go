package netconf

import (
	"context"
	"encoding/xml"
	"sync"
	"sync/atomic"
	"time"

	"github.com/damianoneill/net/v2/netconf/common"
	"github.com/damianoneill/net/v2/netconf/common/codec"

	"github.com/damianoneill/net/v2/netconf/server/ssh"

	xssh "golang.org/x/crypto/ssh"
)

// Server represents a Netconf Server.
// It encapsulates a transport connection to an SSH server, and session handlers that will
// be invoked to handle netconf messages.
type Server struct {
	*ssh.Server
	sf              SessionFactory
	sessionHandlers map[uint64]*SessionHandler
	nextSid         uint64
	trace           *Trace
}

// SessionCallback defines the caller supplied callback functions.
type SessionCallback interface {
	// Capabilities is called to retrieve the capabilities that should be advertised to the client.
	// If the callback returns nil, the default set of capabilities is used.
	Capabilities() []string
	// HandleRequest is called to handle an RPC request.
	HandleRequest(req *RpcRequestMessage) *RpcReplyMessage
}

type SessionFactory func(*SessionHandler) SessionCallback

// SessionHandler represents the server side of an active netconf SSH session.
type SessionHandler struct {

	// server references the Netconf server that launched the session.
	server *Server

	// svrcon is the underlying ssh server connection.
	svrcon *xssh.ServerConn

	// ch is the underlying transport channel.
	ch xssh.Channel

	// The codecs used to handle client i/o
	enc *codec.Encoder
	dec *codec.Decoder

	// Serialises access to encoder (avoiding contention between sending notifications and request responses).
	encLock sync.Mutex

	// The capabilities advertised to the client.
	capabilities []string
	// The session id to be reported to the client.
	sid uint64

	// Channel used to signal successful receipt of client capabilities.
	hellochan chan bool

	// The HelloMessage sent by the connecting client.
	ClientHello *common.HelloMessage

	// Caller supplied callbacks
	cb SessionCallback
}

// RpcRequestMessage and rpcRequest represent an RPC request from a client, where the element type of the
// request body is unknown.
type RpcRequestMessage struct {
	XMLName   xml.Name
	MessageID string     `xml:"message-id,attr"`
	Request   RPCRequest `xml:",any"`
	Body      string     `xml:",innerxml"`
}

// RPCRequest describes an RPC request.
type RPCRequest struct {
	XMLName xml.Name
	Body    string `xml:",innerxml"`
}

// RpcReplyMessage  and ReplyData represent an rpc-reply message that will be sent to a client session, where the
// element type of the reply body (i.e. the content of the data element)
// is unknown.
type RpcReplyMessage struct {
	XMLName   xml.Name          `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-reply"`
	Errors    []common.RPCError `xml:"rpc-error,omitempty"`
	Data      ReplyData         `xml:"data"`
	Ok        bool              `xml:",omitempty"`
	RawReply  string            `xml:"-"`
	MessageID string            `xml:"message-id,attr"`
}
type ReplyData struct {
	XMLName xml.Name `xml:"data"`
	Data    string   `xml:",innerxml"`
}

// RequestHandler is a function type that will be invoked by the session handler to handle an RPC
// request.
type RequestHandler func(h *SessionHandler, req *RpcRequestMessage)

// NewServer creates a new Server that will accept Netconf localhost connections on an ephemeral port (available
// via Port()), with credentials defined by the sshcfg configuration.
func NewServer(ctx context.Context, address string, port int, sshcfg *xssh.ServerConfig, sf SessionFactory) (ncs *Server, err error) {

	trace := ContextNetconfTrace(ctx)
	if trace.Trace != nil && ssh.ContextSshTrace(ctx) == nil {
		ctx = ssh.WithSshTrace(ctx, trace.Trace)
	}

	ncs = &Server{sessionHandlers: make(map[uint64]*SessionHandler), sf: sf, trace: trace}

	ncs.Server, err = ssh.NewServer(ctx, address, port, sshcfg, ncs.handlerFactory())
	if err != nil {
		return nil, err
	}
	return
}

func (ncs *Server) handlerFactory() ssh.HandlerFactory {
	return func(svrconn *xssh.ServerConn) ssh.Handler {
		sid := atomic.AddUint64(&ncs.nextSid, 1)
		sess := ncs.newSessionHandler(svrconn, sid)
		ncs.sessionHandlers[sid] = sess
		return sess
	}
}

// Close closes any active transport to the test server and prevents subsequent connections.
func (ncs *Server) Close() {
	for k, v := range ncs.sessionHandlers {
		if v.ch != nil {
			v.Close() // nolint: gosec, errcheck
			ncs.sessionHandlers[k] = nil
		}
	}
	ncs.Server.Close()
}

func (ncs *Server) newSessionHandler(svrcon *xssh.ServerConn, sid uint64) *SessionHandler { // nolint: deadcode
	sh := &SessionHandler{
		server:       ncs,
		svrcon:       svrcon,
		sid:          sid,
		hellochan:    make(chan bool),
		capabilities: common.DefaultCapabilities,
	}

	ncs.trace.StartSession(sh)

	sh.cb = ncs.sf(sh)
	caps := sh.cb.Capabilities()
	if caps != nil {
		sh.capabilities = caps
	}
	return sh
}

// Handle establishes a Netconf server session on a newly-connected SSH channel.
func (h *SessionHandler) Handle(ch xssh.Channel) {
	h.ch = ch
	h.dec = codec.NewDecoder(ch)
	h.enc = codec.NewEncoder(ch)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Send server hello to client.
	err := h.encode(&common.HelloMessage{Capabilities: h.capabilities, SessionID: h.sid})
	if err == nil {

		go h.handleIncomingMessages(wg)
		ok := h.waitForClientHello()
		if ok {
			// Wait for message handling routine to finish.
			wg.Wait()
		}
	}
	h.server.trace.EndSession(h, err)
}

// Close initiates session tear-down by closing the underlying transport channel.
func (h *SessionHandler) Close() {
	_ = h.ch.Close() // nolint: errcheck, gosec
}

func (h *SessionHandler) waitForClientHello() bool {

	// Wait for the input handler to send the client hello.
	select {
	case <-h.hellochan:
	case <-time.After(time.Duration(5) * time.Second):
	}

	h.server.trace.ClientHello(h)
	return h.ClientHello != nil
}

func (h *SessionHandler) handleIncomingMessages(wg *sync.WaitGroup) {

	defer wg.Done()

	// Loop, looking for a start element type of hello, rpc.
	for {
		token, err := h.dec.Token()
		if err != nil {
			break
		}
		h.handleToken(token)
	}
}

func (h *SessionHandler) handleToken(token xml.Token) {
	switch token := token.(type) {
	case xml.StartElement:
		switch token.Name.Local {
		case common.NameHello.Local: // <hello>
			h.handleHello(token)

		case common.NameRPC.Local: // <rpc>
			h.handleRPC(token)
		}
	}
}

func (h *SessionHandler) handleHello(token xml.StartElement) {
	// Decode the hello element and send it down the channel to trigger the rest of the session setup.

	err := h.decodeElement(&h.ClientHello, &token)
	if err == nil {
		if common.PeerSupportsChunkedFraming(h.ClientHello.Capabilities) && common.PeerSupportsChunkedFraming(h.capabilities) {

			// Update the codec to use chunked framing from now.
			codec.EnableChunkedFraming(h.dec, h.enc)
		}
	}

	h.hellochan <- true
}

func (h *SessionHandler) handleRPC(token xml.StartElement) {
	request := &RpcRequestMessage{}
	err := h.decodeElement(&request, &token)
	if err != nil {
		return
	}

	reply := h.cb.HandleRequest(request)
	if reply != nil {
		_ = h.encode(reply)
	}
}

func (h *SessionHandler) decodeElement(v interface{}, start *xml.StartElement) error {
	err := h.dec.DecodeElement(v, start)
	h.server.trace.Decoded(h, err)
	return err
}

func (h *SessionHandler) encode(m interface{}) error {
	h.encLock.Lock()
	defer h.encLock.Unlock()
	err := h.enc.Encode(m)
	h.server.trace.Encoded(h, err)
	return err
}
