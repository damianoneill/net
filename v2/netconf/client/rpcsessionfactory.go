package client

import (
	"context"
	"time"

	"github.com/imdario/mergo"
	"golang.org/x/crypto/ssh"
)

// Defines a factory method for instantiating netconf rpc sessions.

// NewRPCSession connects to the  target using the ssh configuration, and establishes
// a netconf session with default configuration.
func NewRPCSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string) (s Session, err error) {
	return NewRPCSessionWithConfig(ctx, sshcfg, target, DefaultConfig)
}

// NewRPCSessionFromSSHClient establishes a netconf session over the given ssh Client with default configuration.
func NewRPCSessionFromSSHClient(ctx context.Context, client *ssh.Client) (s Session, err error) {
	return NewRPCSessionFromSSHClientWithConfig(ctx, client, DefaultConfig)
}

// NewRPCSessionWithConfig connects to the  target using the ssh configuration, and establishes
// a netconf session with the client configuration.
func NewRPCSessionWithConfig(ctx context.Context, sshcfg *ssh.ClientConfig, target string, cfg *Config) (s Session, err error) {
	// Use supplied config, but apply any defaults to unspecified values.
	resolvedConfig := *cfg
	_ = mergo.Merge(&resolvedConfig, DefaultConfig)

	var t Transport
	if t, err = createTransport(ctx, sshcfg, target); err != nil {
		return
	}

	if s, err = NewSession(ctx, t, &resolvedConfig); err != nil {
		_ = t.Close()
	}
	return
}

// NewRPCSessionFromSSHClientWithConfig establishes a netconf session over the given ssh Client with the client configuration.
func NewRPCSessionFromSSHClientWithConfig(ctx context.Context, client *ssh.Client, cfg *Config) (s Session, err error) {
	// Use supplied config, but apply any defaults to unspecified values.
	resolvedConfig := *cfg
	_ = mergo.Merge(&resolvedConfig, DefaultConfig)

	var t Transport
	if t, err = createTransportFromSSHClient(ctx, client); err != nil {
		return
	}

	if s, err = NewSession(ctx, t, &resolvedConfig); err != nil {
		_ = t.Close()
	}
	return
}

func createTransport(ctx context.Context, clientConfig *ssh.ClientConfig, target string) (t Transport, err error) {
	return NewSSHTransport(ctx, NewDialer(target, clientConfig), target)
}

func NewDialer(target string, clientConfig *ssh.ClientConfig) *RealDialer { //nolint: golint
	return &RealDialer{target: target, config: clientConfig}
}

type RealDialer struct {
	target string
	config *ssh.ClientConfig
}

func (rd *RealDialer) Dial(ctx context.Context) (cli *ssh.Client, err error) {
	tracer := ContextClientTrace(ctx)

	tracer.DialStart(rd.config, rd.target)
	defer func(begin time.Time) {
		tracer.DialDone(rd.config, rd.target, err, time.Since(begin))
	}(time.Now())

	return ssh.Dial("tcp", rd.target, rd.config)
}

func (rd *RealDialer) Close(cli *ssh.Client) (err error) {
	if cli != nil {
		err = cli.Close()
	}
	return err
}

func createTransportFromSSHClient(ctx context.Context, client *ssh.Client) (t Transport, err error) {
	return NewSSHTransport(ctx, newNoOpDialer(client), client.RemoteAddr().String())
}

func newNoOpDialer(client *ssh.Client) *noOpDialer {
	return &noOpDialer{client: client}
}

type noOpDialer struct {
	client *ssh.Client
}

func (nd *noOpDialer) Dial(ctx context.Context) (cli *ssh.Client, err error) {
	return nd.client, nil
}

func (nd *noOpDialer) Close(_ *ssh.Client) error {
	// Don't want to close a pre-existing connection.
	return nil
}
