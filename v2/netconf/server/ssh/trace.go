package ssh

import (
	"context"
	"log"
	"net"

	"github.com/imdario/mergo"
)

// unique type to prevent assignment.
type sshEventContextKey struct{}

// ContextSshTrace returns the Trace associated with the
// provided context. If none, it returns nil.
func ContextSshTrace(ctx context.Context) *Trace {
	trace, _ := ctx.Value(sshEventContextKey{}).(*Trace)
	if trace == nil {
		trace = NoOpLoggingHooks
	} else {
		_ = mergo.Merge(trace, NoOpLoggingHooks) // nolint: gosec, errcheck
	}
	return trace
}

// WithSshTrace returns a new context based on the provided parent
// ctx. Requests made with the returned context will use
// the provided trace hooks
func WithSshTrace(ctx context.Context, trace *Trace) context.Context {
	ctx = context.WithValue(ctx, sshEventContextKey{}, trace)
	return ctx
}

// Trace defines a structure for handling trace events
type Trace struct {

	// Listened is called when when an Listen() call completes, with err indicating
	// whether it was successful.
	Listened func(adddress string, err error)

	// StartAccepting is called when starting to accept connections.
	StartAccepting func()

	// Accepted is called when an Accept() call completes, with err indicating
	// whether it was successful.
	Accepted func(conn net.Conn, err error)

	// NewServerConn is called when a NewServerConn() call completes, with err indicating
	// whether it was successful.
	NewServerConn func(conn net.Conn, err error)

	// SshChannelAccept is called when a ssh channel Accept() call completes, with err indicating
	// whether it was successful.
	SshChannelAccept func(conn net.Conn, err error)

	// SubsystemRequestReply is called when a subsystem request Reply call completes, with err indicating
	// whether it was successful.
	SubsystemRequestReply func(err error)
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &Trace{
	Listened: func(address string, e error) {
		if e != nil {
			log.Printf("Listen address:%s status:%v\n", address, e)
		}
	},
	StartAccepting: func() {
		log.Printf("Start Accepting\n")
	},
	Accepted: func(conn net.Conn, e error) {
		if e != nil {
			log.Printf("Accept status:%v\n", e)
		}
	},
	NewServerConn: func(conn net.Conn, e error) {
		if e != nil {
			log.Printf("NewServerConn status:%v\n", e)
		}
	},
	SshChannelAccept: func(conn net.Conn, e error) {
		if e != nil {
			log.Printf("SshChannelAccept status:%v\n", e)
		}
	},
	SubsystemRequestReply: func(e error) {
		if e != nil {
			log.Printf("SubsystemRequestReply status:%v\n", e)
		}
	},
}

// DiagnosticLoggingHooks provides a set of default diagnostic hooks
var DiagnosticLoggingHooks = &Trace{
	Listened: func(address string, e error) {
		log.Printf("Listen address:%s status:%v\n", address, e)
	},
	StartAccepting: func() {
		log.Printf("Start Accepting\n")
	},
	Accepted: func(conn net.Conn, e error) {
		log.Printf("Accept conn:%v status:%v\n", conn, e)
	},
	NewServerConn: func(conn net.Conn, e error) {
		log.Printf("NewServerConn conn:%v status:%v\n", conn, e)
	},
	SshChannelAccept: func(conn net.Conn, e error) {
		log.Printf("NewServerConn conn:%v status:%v\n", conn, e)
	},
	SubsystemRequestReply: func(e error) {
		log.Printf("SubsystemRequestReply status:%v\n", e)
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &Trace{
	Listened:              func(address string, e error) {},
	StartAccepting:        func() {},
	Accepted:              func(conn net.Conn, ze error) {},
	NewServerConn:         func(conn net.Conn, ze error) {},
	SshChannelAccept:      func(conn net.Conn, ze error) {},
	SubsystemRequestReply: func(ze error) {},
}
