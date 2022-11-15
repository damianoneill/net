package common

import (
	"encoding/xml"
	"fmt"
)

// Defines structs representing netconf messages and notifications.

// Request represents the body of a Netconf RPC request.
type Request interface{}

// HelloMessage defines the message sent/received during session negotiation.
type HelloMessage struct {
	XMLName      xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 hello"`
	Capabilities []string `xml:"capabilities>capability"`
	SessionID    uint64   `xml:"session-id,omitempty"`
}

// RPCMessage defines an rpc request message
type RPCMessage struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc"`
	MessageID string   `xml:"message-id,attr"`
	*Union
}

// RPCReply defines the an rpc request message
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

// Error generates a string representation of the RPC error
func (re *RPCError) Error() string {
	return fmt.Sprintf("netconf rpc [%s] '%s'", re.Severity, re.Message)
}

// Notification defines a specific notification event.
type Notification struct {
	XMLName   xml.Name
	EventTime string
	Event     string `xml:",innerxml"`
}

// NotificationMessage defines the notification message sent from the server.
type NotificationMessage struct {
	XMLName   xml.Name     // `xml:"notification"`
	EventTime string       `xml:"eventTime"`
	Event     Notification `xml:",any"`
}

type Union struct {
	ValueStr interface{}
	ValueXML string `xml:",innerxml"`
}

func GetUnion(s interface{}) *Union {
	switch request := s.(type) {
	case string:
		return &Union{ValueXML: request}
	default:
		return &Union{ValueStr: request}
	}
}

// DefaultCapabilities sets the default capabilities of the client library
var DefaultCapabilities = []string{
	CapBase10,
	CapBase11,
	CapXpath,
}

// NoChunkedCodecCapabilities sets omits chunked codec capability.
var NoChunkedCodecCapabilities = []string{
	CapBase10,
	CapXpath,
}

// Define xml names for different netconf messages.
var (
	NameHello        = xml.Name{Space: NetconfNS, Local: "hello"}
	NameRPC          = xml.Name{Space: NetconfNS, Local: "rpc"}
	NameRPCReply     = xml.Name{Space: NetconfNS, Local: "rpc-reply"}
	NameNotification = xml.Name{Space: NetconfNotifyNS, Local: "notification"}
)

// Define netconf URNs.
const (
	NetconfNS       = "urn:ietf:params:xml:ns:netconf:base:1.0"
	NetconfNotifyNS = "urn:ietf:params:xml:ns:netconf:notification:1.0"
	CapBase10       = "urn:ietf:params:netconf:base:1.0"
	CapBase11       = "urn:ietf:params:netconf:base:1.1"
	CapXpath        = "urn:ietf:params:netconf:capability:xpath:1.0"
)

// PeerSupportsChunkedFraming returns true if capability list indicates support for chunked framing.
func PeerSupportsChunkedFraming(caps []string) bool {
	for _, capability := range caps {
		if capability == CapBase11 {
			return true
		}
	}
	return false
}
