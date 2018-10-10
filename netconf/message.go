package netconf

import (
	"encoding/xml"
	"fmt"

	"github.com/satori/go.uuid"

	"io"
	"log"
	"sync"

	"github.com/damianoneill/net/netconf/rfc6242"
)

// The Operations layer defines a set of base protocol operations
// invoked as RPC methods with XML-encoded parameters.

// Request represents the body of a Netconf RPC request.
type Request string

// HelloMessage defines the message sent/received during session negotiation.
type HelloMessage struct {
	XMLName      xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 hello"`
	Capabilities []string `xml:"capabilities>capability"`
	SessionID    int      `xml:"session-id,omitempty"`
}

type RPCMessage struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc"`
	MessageID string   `xml:"message-id,attr"`
	Methods   []byte   `xml:",innerxml"`
}

type RPCReply struct {
	XMLName   xml.Name   `xml:"rpc-reply"`
	Errors    []RPCError `xml:"rpc-error,omitempty"`
	Data      string     `xml:",innerxml"`
	Ok        bool       `xml:",omitempty"`
	RawReply  string     `xml:"-"`
	MessageID string     `xml:"message-id,attr"`
}

// RPCError defines an error reply to a RPC request
type RPCError struct {
	Type     string `xml:"error-type"`
	Tag      string `xml:"error-tag"`
	Severity string `xml:"error-severity"`
	Path     string `xml:"error-path"`
	Message  string `xml:"error-message"`
	Info     string `xml:",innerxml"`
}

// Session represents a Netconf Session
type Session interface {
	Execute(req Request) (*RPCReply, error)
	ExecuteAsync(req Request, rchan chan *RPCReply) (err error)
	Close()
}

type sesImpl struct {
	t      Transport
	dec    *decoder
	enc    *encoder
	nclog  *log.Logger
	evtlog *log.Logger

	pool []chan *RPCReply

	responseq []chan *RPCReply
	hello     *HelloMessage
	reqLock   sync.Mutex
	pchLock   sync.Mutex
	rchLock   sync.Mutex
}

type decoder struct {
	*xml.Decoder
	ncDecoder *rfc6242.Decoder
}

type encoder struct {
	xmlEncoder *xml.Encoder
	ncEncoder  *rfc6242.Encoder
}

func (e *encoder) encode(msg interface{}) error {

	err := e.xmlEncoder.Encode(msg)
	if err != nil {
		return err
	}
	err = e.ncEncoder.EndOfMessage()
	if err != nil {
		return err
	}
	return nil
}

// DefaultCapabilities sets the default capabilities of the client library
var DefaultCapabilities = []string{
	"urn:ietf:params:netconf:base:1.0",
}

var (
	netconfNS       = "urn:ietf:params:xml:ns:netconf:base:1.0"
	netconfNotifyNS = "urn:ietf:params:xml:ns:netconf:notification:1.0"
	nameHello       = xml.Name{Space: netconfNS, Local: "hello"}
	nameRPCReply    = xml.Name{Space: netconfNS, Local: "rpc-reply"}
	notification    = xml.Name{Space: netconfNotifyNS, Local: "notification"}
	// :base:1.1 protocol capability
	capBase11 = "urn:ietf:params:netconf:base:1.1"
)

// NewSession creates a new Netconf session, using the supplied Transport.
func NewSession(t Transport, evtlog *log.Logger, nclog *log.Logger) (Session, error) {

	dec := newDecoder(t)
	enc := newEncoder(t)

	sess := &sesImpl{t: t, dec: dec, enc: enc, evtlog: evtlog, nclog: nclog}

	hch := make(chan *HelloMessage)
	go sess.handleInput(hch)

	sess.hello = <-hch
	helloresp := &HelloMessage{Capabilities: DefaultCapabilities}
	for _, capability := range sess.hello.Capabilities {
		if capability == capBase11 {
			// rfc6242.SetChunkedFraming(sess.dec.ncDecoder, sess.enc.ncEncoder)
			// helloresp.Capabilities = []string{capBase11}
			// fmt.Println("upgraded to :base:1.1 chunked-message framing")
			break
		}
	}

	err := sess.enc.encode(helloresp)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

func (si *sesImpl) Execute(req Request) (*RPCReply, error) {

	rchan := si.allocChan()
	defer si.relChan(rchan)

	err := si.ExecuteAsync(req, rchan)
	if err != nil {
		return nil, err
	}
	reply := <-rchan
	return reply, nil
}

func (si *sesImpl) ExecuteAsync(req Request, rchan chan *RPCReply) (err error) {
	si.reqLock.Lock()
	defer si.reqLock.Unlock()
	msg := &RPCMessage{MessageID: uuid.NewV4().String(), Methods: []byte(string(req))}

	si.pushRespChan(rchan)

	return si.enc.encode(msg)

}

func (si *sesImpl) Close() {
	err := si.t.Close()
	if err != nil {
		si.evtlog.Printf("Session close failed %v\n", err)
	}
}

func (si *sesImpl) handleInput(hch chan<- *HelloMessage) {

	defer close(hch)
	for {
		token, err := si.dec.Token()
		if err != nil {
			if err != io.EOF {
				si.evtlog.Printf("Token() error: %v\n", err)
			}
			break
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name {
			case nameHello: // <hello>
				hello := HelloMessage{}
				if err := si.dec.DecodeElement(&hello, &token); err != nil {
					si.evtlog.Printf("DecodeElement() error: %v\n", err)
					return
				}
				hch <- &hello
			case nameRPCReply: // <rpc-reply>
				reply := RPCReply{}
				if err := si.dec.DecodeElement(&reply, &token); err != nil {
					si.evtlog.Printf("DecodeElement() error: %v\n", err)
					return
				}

				respch := si.popRespChan()
				go func(ch chan *RPCReply, r *RPCReply) {
					ch <- r
				}(respch, &reply)

			case notification: // <notification>
				fmt.Println("saw <notification>")
			}
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
	si.responseq, ch = si.responseq[1:], si.responseq[0]
	return
}

func newDecoder(t Transport) *decoder {
	ncDecoder := rfc6242.NewDecoder(t)
	return &decoder{Decoder: xml.NewDecoder(ncDecoder), ncDecoder: ncDecoder}
}

func newEncoder(t Transport) *encoder {
	ncEncoder := rfc6242.NewEncoder(t)
	return &encoder{xmlEncoder: xml.NewEncoder(ncEncoder), ncEncoder: ncEncoder}
}
