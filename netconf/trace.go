package netconf

import (
	"context"
	"log"
	"reflect"
	"time"

	"golang.org/x/crypto/ssh"
)

// unique type to prevent assignment.
type clientEventContextKey struct{}

// ContextClientTrace returns the ClientTrace associated with the
// provided context. If none, it returns nil.
func ContextClientTrace(ctx context.Context) *ClientTrace {
	trace, _ := ctx.Value(clientEventContextKey{}).(*ClientTrace)
	return trace
}

// WithClientTrace returns a new context based on the provided parent
// ctx. Netconf client requests made with the returned context will use
// the provided trace hooks, in addition to any previous hooks
// registered with ctx. Any hooks defined in the provided trace will
// be called first.
func WithClientTrace(ctx context.Context, trace *ClientTrace) context.Context {
	if trace == nil {
		panic("nil trace")
	}
	old := ContextClientTrace(ctx)
	trace.compose(old)

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
	ConnectionClosed func(err error)

	// ReadStart is called before a read from the underlying transport.
	ReadStart func(buf []byte)

	// ReadDone is called after a read from the underlying transport.
	ReadDone func(buf []byte, c int, err error, d time.Duration)

	// WriteStart is called before a write to the underlying transport.
	WriteStart func(buf []byte)

	// WriteDone is called after a write to the underlying transport.
	WriteDone func(buf []byte, c int, err error, d time.Duration)

	// Error is called after an error condition has been detected.
	Error func(context string, err error)

	// NotificationReceived is called when a notification has been received.
	NotificationReceived func(m *Notification)

	// NotificationDropped is called when a notification is dropped because the reader is not ready.
	NotificationDropped func(m *Notification)

	// ExecuteStart is called before the execution of an rpc request.
	ExecuteStart func(req Request, async bool)

	// ExecuteDone is called after the execution of an rpc request.
	ExecuteDone func(req Request, async bool, res *RPCReply, err error, d time.Duration)
}

// compose modifies t such that it respects the previously-registered hooks in old,
// subject to the composition policy requested in t.Compose.
func (t *ClientTrace) compose(old *ClientTrace) {
	if old == nil {
		return
	}
	tv := reflect.ValueOf(t).Elem()
	ov := reflect.ValueOf(old).Elem()
	structType := tv.Type()
	for i := 0; i < structType.NumField(); i++ {
		tf := tv.Field(i)
		hookType := tf.Type()
		if hookType.Kind() != reflect.Func {
			continue
		}
		of := ov.Field(i)
		if of.IsNil() {
			continue
		}
		if tf.IsNil() {
			tf.Set(of)
			continue
		}

		// Make a copy of tf for tf to call. (Otherwise it
		// creates a recursive call cycle and stack overflows)
		tfCopy := reflect.ValueOf(tf.Interface())

		// We need to call both tf and of in some order.
		newFunc := reflect.MakeFunc(hookType, func(args []reflect.Value) []reflect.Value {
			tfCopy.Call(args)
			return of.Call(args)
		})
		tv.Field(i).Set(newFunc)
	}
}

// DefaultLoggingHooks provides a default logging hook
var DefaultLoggingHooks = &ClientTrace{
	Error: func(context string, err error) {
		log.Printf("Error context:%s err:%v\n", context, err)
	},
}

// DiagnosticLoggingHooks provides a default diagnostic hook
var DiagnosticLoggingHooks = &ClientTrace{
	ConnectStart: func(clientConfig *ssh.ClientConfig, target string) {
		log.Printf("ConnectStart target:%s config:%v\n", target, clientConfig)
	},
	ConnectDone: func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration) {
		log.Printf("ConnectDone target:%s config:%v err:%v took:%dns\n", target, clientConfig, err, d)
	},
	ConnectionClosed: func(err error) {
		log.Printf("ConnectionClosed err:%v\n", err)
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

	Error: func(context string, err error) {
		log.Printf("Error context:%s err:%v\n", context, err)
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
