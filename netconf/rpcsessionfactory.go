package netconf

import (
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
func NewRPCSession(sshcfg *ssh.ClientConfig, target string) (s Session, err error) {

	return NewRPCSessionWithConfig(sshcfg, target, defaultConfig)
}

// NewRPCSessionWithConfig connects to the  target using the ssh configuration, and establishes
// a netconf session with the client configuration.
func NewRPCSessionWithConfig(sshcfg *ssh.ClientConfig, target string, cfg *ClientConfig) (s Session, err error) {

	var t Transport
	if t, err = createTransport(sshcfg, target); err != nil {
		return
	}

	if s, err = NewSession(t, defaultLogger, defaultLogger, cfg); err != nil {
		t.Close() // nolint: gosec,errcheck
	}
	return
}

func createTransport(clientConfig *ssh.ClientConfig, target string) (t Transport, err error) {
	return NewSSHTransport(clientConfig, target, "netconf")
}
