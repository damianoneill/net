package netconf

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// The Message layer defines a set of base protocol operations
// invoked as RPC methods with XML-encoded parameters.

// Session represents a Netconf Session
type Session interface {
	// Execute executes an RPC request on the server and returns the reply.
	Execute(req Request) (*RPCReply, error)

	// ExecuteAsync submits an RPC request for execution on the server, arranging for the
	// reply to be sent to the supplied channel.
	ExecuteAsync(req Request, rchan chan *RPCReply) (err error)

	// Subscribe issues an RPC request and returns the reply. If successful, notifications will
	// be sent to the supplied channel.
	Subscribe(req Request, nchan chan *Notification) (reply *RPCReply, err error)

	// Close closes the session and releases any associated resources.
	// The channel will be automatically closed if the underlying network connection is closed, for
	// example if the remote server discoonects.
	// When the session is closed, any outstanding execute requests and reads from a notification
	// channel will return nil.
	Close()

	// ID delivers the server-allocated id of the session.
	ID() uint64

	// Capabilities delivers the server-supplied capabilities.
	ServerCapabilities() []string
}

type sesImpl struct {
	cfg   *ClientConfig
	t     Transport
	dec   *decoder
	enc   *encoder
	trace *ClientTrace

	pool []chan *RPCReply

	hellochan chan bool
	responseq []chan *RPCReply
	subchan   chan *Notification

	hello   *HelloMessage
	reqLock sync.Mutex
	pchLock sync.Mutex
	rchLock sync.Mutex

	notificationDropCount uint64

	target string
}

// DefaultCapabilities sets the default capabilities of the client library
var DefaultCapabilities = []string{
	CapBase10,
	CapBase11,
}

var (
	nameHello    = xml.Name{Space: netconfNS, Local: "hello"}
	nameRPCReply = xml.Name{Space: netconfNS, Local: "rpc-reply"}
	notification = xml.Name{Space: netconfNotifyNS, Local: "notification"}
)

const (
	netconfNS       = "urn:ietf:params:xml:ns:netconf:base:1.0"
	netconfNotifyNS = "urn:ietf:params:xml:ns:netconf:notification:1.0"

	// CapBase10 defines capability value identifying 1.0 support
	CapBase10 = "urn:ietf:params:netconf:base:1.0"
	// CapBase11 defines capability value identifying 1.1 support
	CapBase11 = "urn:ietf:params:netconf:base:1.1"
)

// NewSession creates a new Netconf session, using the supplied Transport.
func NewSession(ctx context.Context, t Transport, cfg *ClientConfig) (Session, error) {
	si := &sesImpl{
		cfg:    cfg,
		t:      t,
		target: t.(*tImpl).target,
		dec:    newDecoder(t),
		enc:    newEncoder(t),
		trace:  ContextClientTrace(ctx),

		hellochan: make(chan bool),
	}

	// Send hello
	err := si.enc.encode(&HelloMessage{Capabilities: DefaultCapabilities})
	if err != nil {
		si.trace.Error("Failed to encode hello", si.target, err)
		si.Close()
		return nil, err
	}

	// Launch goroutine to handle incoming messages from the server.
	go si.handleIncomingMessages()

	err = si.waitForServerHello()
	if err != nil {
		si.trace.Error("Failed to receive hello", si.target, err)
		si.Close()
		return nil, err
	}
	return si, nil
}

func (si *sesImpl) Execute(req Request) (reply *RPCReply, err error) {
	si.trace.ExecuteStart(req, false)

	defer func(begin time.Time) {
		si.trace.ExecuteDone(req, false, reply, err, time.Since(begin))
	}(time.Now())

	// Allocate a response channel
	rchan := si.allocChan()
	defer si.relChan(rchan)

	// Submit the request
	err = si.execute(req, rchan)
	if err != nil {
		return nil, err
	}

	// Wait for the response.
	reply = <-rchan

	err = mapError(reply)
	return reply, err
}

func (si *sesImpl) ExecuteAsync(req Request, rchan chan *RPCReply) (err error) {
	si.trace.ExecuteStart(req, true)
	defer func(begin time.Time) {
		si.trace.ExecuteDone(req, true, nil, err, time.Since(begin))
	}(time.Now())

	return si.execute(req, rchan)
}

func (si *sesImpl) execute(req Request, rchan chan *RPCReply) (err error) {
	// Build the request to be submitted.
	msg := &RPCMessage{MessageID: uuid.New().String(), Methods: []byte(string(req))}

	// Lock the request channel, so the request and response channel set up is atomic.
	si.reqLock.Lock()
	defer si.reqLock.Unlock()

	// Add the response channel to the response queue, but take it off if the request was not
	// submitted successfully.
	si.pushRespChan(rchan)
	if err = si.enc.encode(msg); err != nil {
		si.popRespChan()
	}
	return
}

func (si *sesImpl) Subscribe(req Request, nchan chan *Notification) (reply *RPCReply, err error) {
	// Store the notification channel for the session.
	si.subchan = nchan
	return si.Execute(req)
}

func (si *sesImpl) Close() {
	err := si.t.Close()
	if err != nil {
		si.trace.Error("Session close failed", si.target, err)
	}
}

func (si *sesImpl) ID() uint64 {
	return si.hello.SessionID
}

func (si *sesImpl) ServerCapabilities() []string {
	return si.hello.Capabilities
}

func (si *sesImpl) waitForServerHello() (err error) {
	helloOk := false
	select {
	case helloOk = <-si.hellochan:
	case <-time.After(time.Duration(si.cfg.setupTimeoutSecs) * time.Second):
	}

	if !helloOk {
		err = errors.New("failed to get hello from server")
	}
	return
}

func peerSupportsChunkedFraming(caps []string) bool {
	for _, capability := range caps {
		if capability == CapBase11 {
			return true
		}
	}
	return false
}

func (si *sesImpl) handleIncomingMessages() {
	// When this goroutine finishes, make sure anytbody waiting for an async response or notification
	// gets informed.
	defer si.closeChannels()

	// Loop, looking for a start element type of hello, rpc-reply or notification.
	for {
		token, err := si.dec.Token()
		if err != nil {
			break
		}

		if err = si.handleToken(token); err != nil {
			return
		}
	}
}

func (si *sesImpl) handleToken(token xml.Token) (err error) {
	switch token := token.(type) {
	case xml.StartElement:
		switch token.Name {
		case nameHello: // <hello>
			err = si.handleHello(token)

		case nameRPCReply: // <rpc-reply>
			err = si.handleRPCReply(token)

		case notification: // <notification>
			err = si.handleNotification(token)

		default:
		}
	default:
	}
	return
}

func (si *sesImpl) handleHello(token xml.StartElement) (err error) {
	// Decode the hello element and send it down the channel to trigger the rest of the session setup.

	if err = si.decodeElement(&si.hello, &token); err != nil {
		si.hellochan <- false
		return
	}

	if peerSupportsChunkedFraming(si.hello.Capabilities) {
		// Update the codec to use chunked framing from now.
		enableChunkedFraming(si.dec, si.enc)
	}

	si.hellochan <- true
	si.trace.HelloDone(si.hello)
	return
}

func (si *sesImpl) handleRPCReply(token xml.StartElement) (err error) {
	reply := RPCReply{}
	if err = si.decodeElement(&reply, &token); err != nil {
		return
	}

	// Pop the channel off the head of the queue and send the reply to it.
	respch := si.popRespChan()
	go func(ch chan *RPCReply, r *RPCReply) {
		ch <- r
	}(respch, &reply)
	return
}

func (si *sesImpl) handleNotification(token xml.StartElement) (err error) {
	result := &NotificationMessage{}
	if err = si.decodeElement(&result, &token); err != nil {
		return
	}

	// Send notification to subscription channel, if it's defined and not full.
	if si.subchan != nil {
		notification := buildNotification(result)

		si.trace.NotificationReceived(notification)

		select {
		case si.subchan <- notification:
		default:
			atomic.AddUint64(&si.notificationDropCount, 1)
			si.trace.NotificationDropped(notification)
		}
	}
	return
}

func buildNotification(nmsg *NotificationMessage) *Notification {
	event := fmt.Sprintf(`<%s xmlns="%s">%s</%s>`,
		nmsg.Event.XMLName.Local, nmsg.Event.XMLName.Space, nmsg.Event.Event, nmsg.Event.XMLName.Local)
	notification := &Notification{XMLName: nmsg.Event.XMLName, EventTime: nmsg.EventTime, Event: event}
	return notification
}

func (si *sesImpl) decodeElement(v interface{}, start *xml.StartElement) (err error) {
	if err = si.dec.DecodeElement(v, start); err != nil {
		si.trace.Error(fmt.Sprintf("DecodeElement token:%s", start.Name.Local), si.target, err)
	}
	return
}

func (si *sesImpl) closeChannels() {
	close(si.hellochan)
	if si.subchan != nil {
		close(si.subchan)
	}
	si.closeAllResponseChannels()
}

func (si *sesImpl) closeAllResponseChannels() {
	for {
		if ch := si.popRespChan(); ch != nil {
			close(ch)
		} else {
			return
		}
	}
}

func (si *sesImpl) allocChan() (ch chan *RPCReply) {
	si.pchLock.Lock()
	defer si.pchLock.Unlock()

	l := len(si.pool)
	if l == 0 {
		return make(chan *RPCReply)
	}

	si.pool, ch = si.pool[:l-1], si.pool[l-1]
	return
}

func (si *sesImpl) relChan(ch chan *RPCReply) {
	si.pchLock.Lock()
	defer si.pchLock.Unlock()
	si.pool = append(si.pool, ch)
}

func (si *sesImpl) pushRespChan(ch chan *RPCReply) {
	si.rchLock.Lock()
	defer si.rchLock.Unlock()
	si.responseq = append(si.responseq, ch)
}

func (si *sesImpl) popRespChan() (ch chan *RPCReply) {
	si.rchLock.Lock()
	defer si.rchLock.Unlock()
	if len(si.responseq) > 0 {
		si.responseq, ch = si.responseq[1:], si.responseq[0]
	}
	return
}

// Map an RPC reply to an error, if the reply is either null or contains any RPC error.
func mapError(r *RPCReply) (err error) {
	if r == nil {
		err = io.ErrUnexpectedEOF
	} else if r.Errors != nil {
		for i := 0; i < len(r.Errors); i++ {
			rpcErr := r.Errors[i]
			if rpcErr.Severity == "error" {
				err = &rpcErr
				break
			}
		}
	}
	return
}
