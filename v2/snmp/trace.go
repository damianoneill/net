package snmp

import (
	"log"
)

// ManagerTrace defines a structure for handling trace events
type ManagerTrace struct {
	// ConnectStart is called before establishing a network connection to an agent.
	ConnectStart func(config *managerConfig)

	// ConnectDone is called when the network connection attempt completes, with err indicating
	// whether it was successful.
	ConnectDone func(config *managerConfig, err error)

	// Error is called after an error condition has been detected.
	Error func(location string, config *managerConfig, err error)

	// TODO Define other hooks
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &ManagerTrace{
	Error: func(location string, config *managerConfig, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", location, config.address, err)
	},
}

// DiagnosticLoggingHooks provides a set of default diagnostic hooks
var DiagnosticLoggingHooks = &ManagerTrace{
	ConnectStart: func(config *managerConfig) {
		log.Printf("ConnectStart target:%s\n", config.address)
	},
	ConnectDone: func(config *managerConfig, err error) {
		log.Printf("ConnectDone target:%s err:%v\n", config.address, err)
	},
	Error: func(location string, config *managerConfig, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", location, config.address, err)
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &ManagerTrace{
	ConnectStart: func(config *managerConfig) {},
	ConnectDone:  func(config *managerConfig, err error) {},
	Error:        func(location string, config *managerConfig, err error) {},
}
