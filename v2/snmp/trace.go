package snmp

import (
	"encoding/hex"
	"log"
	"time"
)

// SessionTrace defines a structure for handling trace events
type SessionTrace struct {
	// ConnectStart is called before establishing a network connection to an agent.
	ConnectStart func(config *SessionConfig)

	// ConnectDone is called when the network connection attempt completes, with err indicating
	// whether it was successful.
	ConnectDone func(config *SessionConfig, err error, d time.Duration)

	// Error is called after an error condition has been detected.
	Error func(location string, config *SessionConfig, err error)

	// WriteDone is called after a packet has been written
	WriteDone func(config *SessionConfig, output []byte, err error, d time.Duration)

	// ReadDone is called after a read has completed
	ReadDone func(config *SessionConfig, input []byte, err error, d time.Duration)

	// TODO Define other hooks
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &SessionTrace{
	Error: func(location string, config *SessionConfig, err error) {
		log.Printf("SNMP-Error context:%s target:%s err:%v\n", location, config.address, err)
	},
}

// MetricLoggingHooks provides a set of hooks that log metrics.
var MetricLoggingHooks = &SessionTrace{
	ConnectDone: func(config *SessionConfig, err error, d time.Duration) {
		log.Printf("SNMP-ConnectDone target:%s err:%v took:%dms\n", config.address, err, d.Milliseconds())
	},
	Error: DefaultLoggingHooks.Error,
	WriteDone: func(config *SessionConfig, output []byte, err error, d time.Duration) {
		log.Printf("SNMP-WriteDone target:%s err:%v took:%dms\n", config.address, err, d.Milliseconds())
	},
	ReadDone: func(config *SessionConfig, input []byte, err error, d time.Duration) {
		log.Printf("SNMP-ReadDone target:%s err:%v took:%dms\n", config.address, err, d.Milliseconds())
	},
}

// DiagnosticLoggingHooks provides a set of hooks that log all events with all data.
var DiagnosticLoggingHooks = &SessionTrace{
	ConnectStart: func(config *SessionConfig) {
		log.Printf("SNMP-ConnectStart target:%s\n", config.address)
	},
	ConnectDone: MetricLoggingHooks.ConnectDone,
	Error:       DefaultLoggingHooks.Error,
	WriteDone: func(config *SessionConfig, output []byte, err error, d time.Duration) {
		log.Printf("SNMP-WriteDone target:%s err:%v took:%dms data:%s\n", config.address, err, d.Milliseconds(), hex.EncodeToString(output))
	},
	ReadDone: func(config *SessionConfig, input []byte, err error, d time.Duration) {
		log.Printf("SNMP-ReadDone target:%s err:%v took:%dms data:%s\n", config.address, err, d.Milliseconds(), hex.EncodeToString(input))
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &SessionTrace{
	ConnectStart: func(config *SessionConfig) {},
	ConnectDone:  func(config *SessionConfig, err error, d time.Duration) {},
	Error:        func(location string, config *SessionConfig, err error) {},
	WriteDone:    func(config *SessionConfig, output []byte, err error, d time.Duration) {},
	ReadDone:     func(config *SessionConfig, input []byte, err error, d time.Duration) {},
}
