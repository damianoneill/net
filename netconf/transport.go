package netconf

import (
	"context"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// The Secure Transport layer provides a communication path between
// the client and server.  NETCONF can be layered over any
// transport protocol that provides a set of basic requirements.

// Transport interface defines what characteristics make up a NETCONF transport
// layer object.
type Transport interface {
	io.ReadWriteCloser
}

type tImpl struct {
	reader      io.Reader
	writeCloser io.WriteCloser
	sshSession  *ssh.Session
	sshClient   *ssh.Client
	trace       *ClientTrace
	target 		string
}

// NewSSHTransport creates a new SSH transport, connecting to the target with the supplied client configuration
// and requesting the specified subsystem.
// nolint : gosec
func NewSSHTransport(ctx context.Context, clientConfig *ssh.ClientConfig, target, subsystem string) (rt Transport, err error) {

	impl := tImpl{target: target}
	impl.trace = ContextClientTrace(ctx)

	impl.trace.ConnectStart(clientConfig, target)

	defer func(begin time.Time) {
		impl.trace.ConnectDone(clientConfig, target, err, time.Since(begin))
	}(time.Now())

	defer func() {
		// nolint: gosec, errcheck
		if err != nil {
			if impl.sshClient != nil {
				impl.sshClient.Close()
			}
			if impl.sshSession != nil {
				impl.sshSession.Close()
			}
		}
	}()

	impl.sshClient, err = ssh.Dial("tcp", target, clientConfig)
	if err != nil {
		return
	}

	if impl.sshSession, err = impl.sshClient.NewSession(); err != nil {
		return
	}

	if err = impl.sshSession.RequestSubsystem(subsystem); err != nil {
		return
	}

	if impl.reader, err = impl.sshSession.StdoutPipe(); err != nil {
		return
	}

	if impl.writeCloser, err = impl.sshSession.StdinPipe(); err != nil {
		return
	}

	impl.injectTraceReader()
	impl.injectTraceWriter()

	rt = &impl
	return
}

func (t *tImpl) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

func (t *tImpl) Write(p []byte) (n int, err error) {
	return t.writeCloser.Write(p)
}

// Close closes all session resources in the following order:
//
//  1. stdin pipe
//  2. SSH session
//  3. SSH client
//
// Errors are returned with priority matching the same order.
func (t *tImpl) Close() (err error) {

	defer t.trace.ConnectionClosed(t.target, err)

	var (
		writeCloseErr      error
		sshSessionCloseErr error
	)

	if t.writeCloser != nil {
		writeCloseErr = t.writeCloser.Close()
	}

	if t.sshSession != nil {
		sshSessionCloseErr = t.sshSession.Close()
	}

	if t.sshClient != nil {
		err = t.sshClient.Close()
	}

	if err == nil {
		err = writeCloseErr
	}

	if err == nil {
		err = sshSessionCloseErr
	}

	return err
}

type traceReader struct {
	r     io.Reader
	trace *ClientTrace
}

func (t *tImpl) injectTraceReader() {
	t.reader = &traceReader{r: t.reader, trace: t.trace}
}

func (tr *traceReader) Read(p []byte) (c int, err error) {

	tr.trace.ReadStart(p)
	defer func(begin time.Time) {
		tr.trace.ReadDone(p, c, err, time.Since(begin))
	}(time.Now())

	c, err = tr.r.Read(p)

	return
}

type traceWriter struct {
	w     io.WriteCloser
	trace *ClientTrace
}

func (t *tImpl) injectTraceWriter() {
	t.writeCloser = &traceWriter{w: t.writeCloser, trace: t.trace}
}

func (tw *traceWriter) Write(p []byte) (c int, err error) {
	tw.trace.WriteStart(p)
	defer func(begin time.Time) {
		tw.trace.WriteDone(p, c, err, time.Since(begin))
	}(time.Now())

	c, err = tw.w.Write(p)

	return
}

func (tw *traceWriter) Close() (err error) {
	return tw.w.Close()
}
