package snmp

import (
	"encoding/hex"
	"log"
)

// SessionTrace defines a structure for handling trace events
type SessionTrace struct {
	// ConnectStart is called before establishing a network connection to an agent.
	ConnectStart func(config *sessionConfig)

	// ConnectDone is called when the network connection attempt completes, with err indicating
	// whether it was successful.
	ConnectDone func(config *sessionConfig, err error)

	// Error is called after an error condition has been detected.
	Error func(location string, config *sessionConfig, err error)

	// WriteComplete is called after a packet has been written
	WriteComplete func(config *sessionConfig, output []byte, err error)

	// ReadComplete is called after a read has completed
	ReadComplete func(config *sessionConfig, input []byte, err error)

	// TODO Define other hooks
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &SessionTrace{
	Error: func(location string, config *sessionConfig, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", location, config.address, err)
	},
}

// DiagnosticLoggingHooks provides a set of default diagnostic hooks
var DiagnosticLoggingHooks = &SessionTrace{
	ConnectStart: func(config *sessionConfig) {
		log.Printf("ConnectStart target:%s\n", config.address)
	},
	ConnectDone: func(config *sessionConfig, err error) {
		log.Printf("ConnectDone target:%s err:%v\n", config.address, err)
	},
	Error: func(location string, config *sessionConfig, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", location, config.address, err)
	},
	WriteComplete: func(config *sessionConfig, output []byte, err error) {
		log.Printf("WriteComplete target:%s err:%v data:%s\n", config.address, err, hex.EncodeToString(output))
	},
	ReadComplete: func(config *sessionConfig, input []byte, err error) {
		log.Printf("ReadComplete target:%s err:%v data:%s\n", config.address, err, hex.EncodeToString(input))
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &SessionTrace{
	ConnectStart:  func(config *sessionConfig) {},
	ConnectDone:   func(config *sessionConfig, err error) {},
	Error:         func(location string, config *sessionConfig, err error) {},
	WriteComplete: func(config *sessionConfig, output []byte, err error) {},
	ReadComplete:  func(config *sessionConfig, input []byte, err error) {},
}
