package netconf

import (
	"context"
	"log"
	"time"

	"github.com/imdario/mergo"
	"golang.org/x/crypto/ssh"
)

// unique type to prevent assignment.
type clientEventContextKey struct{}

// ContextClientTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextClientTrace(ctx context.Context) *ClientTrace {
	trace, _ := ctx.Value(clientEventContextKey{}).(*ClientTrace)
	if trace == nil {
		trace = NoOpLoggingHooks
	} else {
		mergo.Merge(trace, NoOpLoggingHooks) // nolint: gosec, errcheck
	}
	return trace
}

// WithClientTrace returns a new context based on the provided parent
// ctx. Netconf client requests made with the returned context will use
// the provided trace hooks
func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context {

	// old := ContextClientTrace(ctx)
	// trace.compose(old)

	ctx = context.WithValue(ctx, clientEventContextKey{}, trace)
	return ctx
}

// ClientTrace defines a structure for handling trace events
type ClientTrace struct {
	// ConnectStart is called when starting to connect to a remote server.
	ConnectStart func(clientConfig *ssh.ClientConfig, target string)

	// ConnectDone is called when the connection attempt compiletes, with err indicating
	// whether it was successful.
	ConnectDone func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration)

	// ConnectionClosed is called after a transport connection has been closed, with
	// err indicating any error condition.
	ConnectionClosed func(target string, err error)

	// ReadStart is called before a read from the underlying transport.
	ReadStart func(buf []byte)

	// ReadDone is called after a read from the underlying transport.
	ReadDone func(buf []byte, c int, err error, d time.Duration)

	// WriteStart is called before a write to the underlying transport.
	WriteStart func(buf []byte)

	// WriteDone is called after a write to the underlying transport.
	WriteDone func(buf []byte, c int, err error, d time.Duration)

	// Error is called after an error condition has been detected.
	Error func(context, target string, err error)

	// NotificationReceived is called when a notification has been received.
	NotificationReceived func(m *Notification)

	// NotificationDropped is called when a notification is dropped because the reader is not ready.
	NotificationDropped func(m *Notification)

	// ExecuteStart is called before the execution of an rpc request.
	ExecuteStart func(req Request, async bool)

	// ExecuteDone is called after the execution of an rpc request.
	ExecuteDone func(req Request, async bool, res *RPCReply, err error, d time.Duration)
}

// DefaultLoggingHooks provides a default logging hook to report errors.
var DefaultLoggingHooks = &ClientTrace{
	Error: func(context, target string, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", context, target, err)
	},
}

// DiagnosticLoggingHooks provides a set of default diagnostic hooks
var DiagnosticLoggingHooks = &ClientTrace{
	ConnectStart: func(clientConfig *ssh.ClientConfig, target string) {
		log.Printf("ConnectStart target:%s config:%v\n", target, clientConfig)
	},
	ConnectDone: func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration) {
		log.Printf("ConnectDone target:%s config:%v err:%v took:%dns\n", target, clientConfig, err, d)
	},
	ConnectionClosed: func(target string, err error) {
		log.Printf("ConnectionClosed target:%s err:%v\n", target, err)
	},
	ReadStart: func(p []byte) {
		log.Printf("ReadStart capacity:%d\n", len(p))
	},
	ReadDone: func(p []byte, c int, err error, d time.Duration) {
		log.Printf("ReadDone len:%d err:%v took:%dns\n", c, err, d)
	},
	WriteStart: func(p []byte) {
		log.Printf("WriteStart len:%d\n", len(p))
	},
	WriteDone: func(p []byte, c int, err error, d time.Duration) {
		log.Printf("WriteDone len:%d err:%v took:%dns\n", c, err, d)
	},

	Error: func(context, target string, err error) {
		log.Printf("Error context:%s target:%s err:%v\n", context, target, err)
	},
	NotificationReceived: func(n *Notification) {
		log.Printf("NotificationReceived %s\n", n.XMLName.Local)
	},
	NotificationDropped: func(n *Notification) {
		log.Printf("NotificationDropped %s\n", n.XMLName.Local)
	},
	ExecuteStart: func(req Request, async bool) {
		log.Printf("ExecuteStart async:%v req:%s\n", async, req)
	},
	ExecuteDone: func(req Request, async bool, res *RPCReply, err error, d time.Duration) {
		log.Printf("ExecuteDone async:%v req:%s err:%v took:%dns\n", async, req, err, d)
	},
}

// NoOpLoggingHooks provides set of hooks that do nothing.
var NoOpLoggingHooks = &ClientTrace{
	ConnectStart:     func(clientConfig *ssh.ClientConfig, target string) {},
	ConnectDone:      func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration) {},
	ConnectionClosed: func(target string, err error) {},

	ReadStart: func(p []byte) {},
	ReadDone:  func(p []byte, c int, err error, d time.Duration) {},

	WriteStart: func(p []byte) {},
	WriteDone:  func(p []byte, c int, err error, d time.Duration) {},

	Error:                func(context, target string, err error) {},
	NotificationReceived: func(n *Notification) {},
	NotificationDropped:  func(n *Notification) {},
	ExecuteStart:         func(req Request, async bool) {},
	ExecuteDone:          func(req Request, async bool, res *RPCReply, err error, d time.Duration) {},
}
