package netconf

import (
	"encoding/xml"
)

// Defines structs representing netconf messages and notifications.

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

type Notification struct {
	XMLName   xml.Name
	EventTime string
	Event     string `xml:",innerxml"`
}
type NotificationMessage struct {
	XMLName   xml.Name     //`xml:"notification"`
	EventTime string       `xml:"eventTime"`
	Event     Notification `xml:",any"`
}
