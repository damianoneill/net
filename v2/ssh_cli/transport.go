package cli

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/imdario/mergo"

	"golang.org/x/crypto/ssh"
)

type SSHTransport interface {
	io.WriteCloser
	io.Reader
}

type TransportConfig struct{}

var DefaultTransportConfig = TransportConfig{}

type transportImpl struct {
	cfg     *TransportConfig
	client  *ssh.Client
	session *ssh.Session
	io.Reader
	io.WriteCloser
}

func NewSSHTransport(ctx context.Context, sshcfg *ssh.ClientConfig, cfg *TransportConfig, target string) (SSHTransport, error) {
	// Use supplied config, but apply any defaults to unspecified values.
	resolvedConfig := *cfg
	_ = mergo.Merge(&resolvedConfig, DefaultTransportConfig)

	var err error
	t := &transportImpl{cfg: &resolvedConfig}
	t.client, err = ssh.Dial("tcp", target, sshcfg)
	if err != nil {
		return nil, errors.Wrap(err, "new Clisession failed")
	}

	t.session, err = t.client.NewSession()
	if err != nil {
		t.Close()
		return nil, errors.Wrap(err, "new ssh session failed")
	}

	t.Reader, _ = t.session.StdoutPipe()
	// TODO Handle stderr
	// ereader, _ := session.StderrPipe()
	t.WriteCloser, _ = t.session.StdinPipe()

	terminalMode := ssh.TerminalModes{
		ssh.ECHO: 0,
		// ssh.TTY_OP_ISPEED: 28800,
		// ssh.TTY_OP_OSPEED: 28800,
	}
	err = t.session.RequestPty("dumb", 80, 80, terminalMode)
	if err != nil {
		_ = t.Close()
		return nil, errors.Wrap(err, "request pty failed")
	}

	if err = t.session.Shell(); err != nil {
		_ = t.Close()
		return nil, errors.Wrap(err, "login shell failed")
	}

	return t, nil
}

func (t *transportImpl) Close() error {
	if t.WriteCloser != nil {
		_ = t.WriteCloser.Close()
	}
	if t.session != nil {
		_ = t.session.Close()
	}
	if t.client != nil {
		_ = t.client.Close()
	}
	return nil
}
