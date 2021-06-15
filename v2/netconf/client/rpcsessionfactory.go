package client

import (
	"context"

	"github.com/imdario/mergo"
	"golang.org/x/crypto/ssh"
)

// Defines a factory method for instantiating netconf rpc sessions.

// NewRPCSession connects to the  target using the ssh configuration, and establishes
// a netconf session with default configuration.
func NewRPCSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string) (s Session, err error) {
	return NewRPCSessionWithConfig(ctx, sshcfg, target, DefaultConfig)
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

func createTransport(ctx context.Context, clientConfig *ssh.ClientConfig, target string) (t Transport, err error) {
	return NewSSHTransport(ctx, clientConfig, target, "netconf")
}
