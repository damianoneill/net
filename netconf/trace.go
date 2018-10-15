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

type ClientTrace struct {
	// ConnectStart is called when before starting to connect to a remote server.
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

var DefaultLoggingHooks = &ClientTrace{
	ConnectStart: func(clientConfig *ssh.ClientConfig, target string) {
		log.Printf("ConnectStart target:%s config:%v\n", target, clientConfig)
	},
	ConnectDone: func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration) {
		log.Printf("ConnectDone target:%s config:%v err:%v took:%d\n", target, clientConfig, err, d)
	},
	ConnectionClosed: func(err error) {
		log.Printf("ConnectionClosed err:%v\n", err)
	},
	ReadStart: func(p []byte) {
		log.Printf("ReadStart capacity:%d\n", len(p))
	},
	ReadDone: func(p []byte, c int, err error, d time.Duration) {
		log.Printf("ReadDone len:%d err:%v took:%d\n", c, err, d)
	},
	WriteStart: func(p []byte) {
		log.Printf("WriteStart len:%d\n", len(p))
	},
	WriteDone: func(p []byte, c int, err error, d time.Duration) {
		log.Printf("WriteDone len:%d err:%v took:%d\n", c, err, d)
	},
}
