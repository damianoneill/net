package netconf

import (
	"context"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

// Defines a factory method for instantiating netconf rpc sessions.

var (
	defaultLogger = log.New(os.Stderr, "logger:", log.Lshortfile)
)

// NewRPCSession connects to the  target using the ssh configuration, and establishes
// a netconf session with default configuration.
func NewRPCSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string) (s Session, err error) {

	return NewRPCSessionWithConfig(ctx, sshcfg, target, defaultConfig)
}

// NewRPCSessionWithConfig connects to the  target using the ssh configuration, and establishes
// a netconf session with the client configuration.
func NewRPCSessionWithConfig(ctx context.Context, sshcfg *ssh.ClientConfig, target string, cfg *ClientConfig) (s Session, err error) {

	var t Transport
	if t, err = createTransport(ctx, sshcfg, target); err != nil {
		return
	}

	if s, err = NewSession(t, defaultLogger, defaultLogger, cfg); err != nil {
		t.Close() // nolint: gosec,errcheck
	}
	return
}

func createTransport(ctx context.Context, clientConfig *ssh.ClientConfig, target string) (t Transport, err error) {
	return NewSSHTransport(ctx, clientConfig, target, "netconf")
}
