package snmp

import (
	"encoding/hex"
	"log"
	"net"
)

// ServerHooks defines a structure for handling server hook events
type ServerHooks struct {

	// StartListening is called when the server is about to start listening for messages.
	StartListening func(addr net.Addr)

	// StopListening is called when the server has stopped listening.
	StopListening func(addr net.Addr, err error)

	// Error is called after an error condition has been detected.
	Error func(config *serverConfig, err error)

	// WriteComplete is called after a packet has been written
	WriteComplete func(config *serverConfig, addr net.Addr, output []byte, err error)

	// ReadComplete is called after a read has completed
	ReadComplete func(config *serverConfig, addr net.Addr, input []byte, err error)
}

// DefaultServerHooks provides a default logging hook to report server errors.
var DefaultServerHooks = &ServerHooks{
	Error: func(config *serverConfig, err error) {
		log.Printf("Error target:%s err:%v\n", config.address, err)
	},
	WriteComplete: func(config *serverConfig, addr net.Addr, output []byte, err error) {
		if err != nil {
			log.Printf("WriteComplete target:%s err:%v\n", addr, err)
		}
	},
	ReadComplete: func(config *serverConfig, addr net.Addr, input []byte, err error) {
		if err != nil {
			log.Printf("ReadComplete source:%s err:%v\n", addr, err)
		}
	},
}

// DiagnosticServerHooks provides a set of default diagnostic server hooks
var DiagnosticServerHooks = &ServerHooks{
	StartListening: func(addr net.Addr) {
		log.Printf("StartListening address:%s\n", addr)
	},
	StopListening: func(addr net.Addr, err error) {
		log.Printf("StopListening address:%s err:%v\n", addr, err)
	},
	Error: func(config *serverConfig, err error) {
		log.Printf("Error err:%v\n", err)
	},
	WriteComplete: func(config *serverConfig, addr net.Addr, output []byte, err error) {
		log.Printf("WriteComplete target:%s err:%v data:%s\n", addr, err, hex.EncodeToString(output))
	},
	ReadComplete: func(config *serverConfig, addr net.Addr, input []byte, err error) {
		log.Printf("ReadComplete source:%s err:%v data:%s\n", addr, err, hex.EncodeToString(input))
	},
}

// NoOpServerHooks provides set of server hooks that do nothing.
var NoOpServerHooks = &ServerHooks{
	StartListening: func(addr net.Addr) {},
	StopListening:  func(addr net.Addr, err error) {},
	Error:          func(config *serverConfig, err error) {},
	WriteComplete:  func(config *serverConfig, addr net.Addr, output []byte, err error) {},
	ReadComplete:   func(config *serverConfig, addr net.Addr, input []byte, err error) {},
}
