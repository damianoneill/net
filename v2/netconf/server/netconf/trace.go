package netconf

import (
	"context"
	"log"

	"github.com/damianoneill/net/v2/netconf/server/ssh"

	"github.com/imdario/mergo"
)

// unique type to prevent assignment.
type netconfEventContextKey struct{}

// ContextNetconfTrace returns the Trace associated with the
// provided context. If none, it returns nil.
func ContextNetconfTrace(ctx context.Context) *Trace {
	trace, _ := ctx.Value(netconfEventContextKey{}).(*Trace)
	if trace == nil {
		trace = NoOpLoggingHooks
	} else {
		_ = mergo.Merge(trace, NoOpLoggingHooks)
	}
	return trace
}

// WithTrace returns a new context based on the provided parent
// ctx. Requests made with the returned context will use
// the provided trace hooks
func WithTrace(ctx context.Context, trace *Trace) context.Context {
	ctx = context.WithValue(ctx, netconfEventContextKey{}, trace)
	return ctx
}

// Trace defines a structure for handling trace events
type Trace struct {
	*ssh.Trace
	StartSession func(s *SessionHandler)
	EndSession   func(s *SessionHandler, e error)
	ClientHello  func(s *SessionHandler)
	Encoded      func(s *SessionHandler, e error)
	Decoded      func(s *SessionHandler, e error)
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &Trace{
	ClientHello: func(s *SessionHandler) {
		if s.ClientHello == nil {
			log.Printf("ClientHello id:%d message:%v\n", s.sid, s.ClientHello)
		}
	},
	EndSession: func(s *SessionHandler, e error) {
		if e != nil {
			log.Printf("EndSession id:%d error:%v\n", s.sid, e)
		}
	},
	Encoded: func(s *SessionHandler, e error) {
		if e != nil {
			log.Printf("Encoded id:%d error:%v\n", s.sid, e)
		}
	},
	Decoded: func(s *SessionHandler, e error) {
		if e != nil {
			log.Printf("Decoded id:%d error:%v\n", s.sid, e)
		}
	},
}

// DiagnosticLoggingHooks provides a set of default diagnostic hooks
var DiagnosticLoggingHooks = &Trace{
	ClientHello: func(s *SessionHandler) {
		log.Printf("ClientHello id:%d message:%v\n", s.sid, s.ClientHello)
	},
	StartSession: func(s *SessionHandler) {
		log.Printf("StartSession id:%d remote:%s\n", s.sid, s.svrcon.RemoteAddr())
	},
	EndSession: func(s *SessionHandler, e error) {
		log.Printf("EndSession id:%d error:%v\n", s.sid, e)
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &Trace{
	StartSession: func(s *SessionHandler) {},
	ClientHello:  func(s *SessionHandler) {},
	EndSession:   func(s *SessionHandler, e error) {},
	Encoded:      func(s *SessionHandler, e error) {},
	Decoded:      func(s *SessionHandler, e error) {},
}
