package netconf

import (
	"io"

	"golang.org/x/crypto/ssh"
)

// The Secure Transport layer provides a communication path between
// the client and server.  NETCONF can be layered over any
// transport protocol that provides a set of basic requirements.

// Transport interface defines what characterisitics make up a NETCONF transport
// layer object.
type Transport interface {
	io.ReadWriteCloser
}

type tImpl struct {
	reader      io.Reader
	writeCloser io.WriteCloser
	sshSession  *ssh.Session
	sshClient   *ssh.Client
}

// NewSSHTransport creates a ...
func NewSSHTransport(clientConfig *ssh.ClientConfig, target, subsystem string) (rt Transport, err error) {

	var impl tImpl

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
func (t *tImpl) Close() error {

	var (
		writeCloseErr      error
		sshSessionCloseErr error
		sshClientCloseErr  error
	)

	if t.writeCloser != nil {
		writeCloseErr = t.writeCloser.Close()
	}

	if t.sshSession != nil {
		sshSessionCloseErr = t.sshSession.Close()
	}

	if t.sshClient != nil {
		sshClientCloseErr = t.sshClient.Close()
	}

	if writeCloseErr != nil {
		return writeCloseErr
	}

	if sshSessionCloseErr != nil {
		return sshSessionCloseErr
	}

	return sshClientCloseErr
}
