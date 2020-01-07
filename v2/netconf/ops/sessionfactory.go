package ops

import (
	"context"

	"github.com/damianoneill/net/v2/netconf/client"

	"golang.org/x/crypto/ssh"
)

// Defines a factory method for instantiating netconf sessions.

// NewRPCSession connects to the  target using the ssh configuration, and establishes
// a netconf session with default configuration.
func NewSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string) (s OpSession, err error) {

	return NewSessionWithConfig(ctx, sshcfg, target, client.DefaultConfig)
}

// NewRPCSessionWithConfig connects to the  target using the ssh configuration, and establishes
// a netconf session with the client configuration.
func NewSessionWithConfig(ctx context.Context, sshcfg *ssh.ClientConfig, target string, cfg *client.Config) (s OpSession, err error) {

	var cs client.Session
	if cs, err = client.NewRPCSessionWithConfig(ctx, sshcfg, target, cfg); err != nil {
		return
	}

	s = &sImpl{Session: cs}
	return
}

func createTransport(ctx context.Context, clientConfig *ssh.ClientConfig, target string) (t client.Transport, err error) {
	return client.NewSSHTransport(ctx, clientConfig, target, "netconf")
}
