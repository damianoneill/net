package netconf

import (
	"crypto/rand"
	"encoding/xml"
	"fmt"

	"io"
	"log"

	"github.com/damianoneill/net/netconf/rfc6242"
)

// The Operations layer defines a set of base protocol operations
// invoked as RPC methods with XML-encoded parameters.

// Request represents the body of a Netconf RPC request.
type Request string

// Response represents the response to a Netconf RPC request.
type Response struct {
}

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

// ResponseHandler defines a callback function that will be invoked to handle a response to
// an asynchronous request.
type ResponseHandler func(res Response)

// Session represents a Netconf Session
type Session interface {
	Execute(req Request) (*Response, error)
	ExecuteAsync(req Request) (resp <-chan *Response, err error)
}

type sesImpl struct {
	t      Transport
	dec    *decoder
	enc    *encoder
	nclog  *log.Logger
	evtlog *log.Logger

	pool []chan *Response

	responseq []chan *Response
	hello     *HelloMessage
}

type decoder struct {
	*xml.Decoder
	ncDecoder *rfc6242.Decoder
}

type encoder struct {
	*xml.Encoder
	ncEncoder *rfc6242.Encoder
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

	err := sess.enc.Encode(helloresp)
	if err != nil {
		return nil, err
	}
	err = sess.enc.ncEncoder.EndOfMessage()
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (si *sesImpl) Execute(req Request) (*Response, error) {
	msg := &RPCMessage{MessageID: msgID(), Methods: []byte(string(req))}
	err := si.enc.Encode(msg)
	if err != nil {
		return nil, err
	}
	err = si.enc.ncEncoder.EndOfMessage()
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (si *sesImpl) ExecuteAsync(req Request) (resp <-chan *Response, err error) {
	return nil, nil
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
			fmt.Printf("Start token name:%v\n", token.Name)
			switch token.Name {
			case nameHello: // <hello>
				hello := HelloMessage{}
				if err := si.dec.DecodeElement(&hello, &token); err != nil {
					si.evtlog.Printf("DecodeElement() error: %v\n", err)
					return
				}
				hch <- &hello
			case nameRPCReply: // <rpc-reply>
				fmt.Println("saw <rpc-reply>")
				reply := RPCReply{}
				if err := si.dec.DecodeElement(&reply, &token); err != nil {
					si.evtlog.Printf("DecodeElement() error: %v\n", err)
					return
				}
				fmt.Printf("Reply:%v\n", reply)
			case notification: // <notification>
				fmt.Println("saw <notification>")
			}
		}
	}
}

func newDecoder(t Transport) *decoder {
	ncDecoder := rfc6242.NewDecoder(t)
	return &decoder{Decoder: xml.NewDecoder(ncDecoder), ncDecoder: ncDecoder}
}

func newEncoder(t Transport) *encoder {
	ncEncoder := rfc6242.NewEncoder(t)
	return &encoder{Encoder: xml.NewEncoder(ncEncoder), ncEncoder: ncEncoder}
}

var msgID = uuid

// uuid generates a "good enough" uuid without adding external dependencies
func uuid() string {
	b := make([]byte, 16)
	io.ReadFull(rand.Reader, b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
