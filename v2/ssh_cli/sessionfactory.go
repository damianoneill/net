package cli

import (
	"context"
	"time"

	"golang.org/x/crypto/ssh"
)

type SessionFactory interface {
	NewSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string, opts ...SessionOption) (s Session, err error)
}

// SessionOption implements options for configuring session behaviour.
type SessionOption func(*SessionConfig)

// WithCommand defines initialisation commands to be executed after a session has been established.
func WithCommands(cmds ...string) SessionOption {
	return func(c *SessionConfig) {
		c.initCmds = cmds
	}
}

// WithPrompt overrides the automatic prompt detection that a new client session applies to determine the cli prompt
// that is used to detect the end of a server response.
// A non-empty pattern value defines a regular expression that will be used to detect the cli prompt.
// An empty pattern indicates the the WaitFor option when calling Send.
func WithPrompt(pattern string) SessionOption {
	return func(c *SessionConfig) {
		c.autoDetect = false
		c.pattern = pattern
	}
}

// WithTimeout defines the length of time to wait without receiving any input that is used to determine
// that the server has completed a response.
// Typically, only used when auto-detecting the cli prompt.
func WithTimeout(timeout time.Duration) SessionOption {
	return func(c *SessionConfig) {
		c.readTimeout = timeout
	}
}

// SessionConfig defines properties controlling session behaviour.
type SessionConfig struct {
	// Any commands that should be executed after establishing a new session.
	initCmds []string
	// If true, the session will auto-detect the cli prompt at session startup.
	autoDetect bool
	// If not empty, defines a regular expression that should be used to identify the cli prompt.
	// If pattern is empty and autoDetect is false, all calls to the Send() method should specfiy the WaitFor option.
	pattern string
	// See WithTimeout above.
	readTimeout time.Duration
}

var DefaultConfig = SessionConfig{
	autoDetect:  true,
	readTimeout: time.Second * 1,
}

type FactoryImpl struct {
	cfg *SessionConfig
}

func (f FactoryImpl) NewSession(ctx context.Context, sshcfg *ssh.ClientConfig, target string,
	opts ...SessionOption) (s Session, err error) {
	config := *f.cfg
	for _, opt := range opts {
		opt(&config)
	}

	t, err := NewSSHTransport(ctx, sshcfg, &TransportConfig{}, target)
	if err != nil {
		return nil, err
	}

	return NewCliSession(ctx, t, &config)
}

func NewSessionFactory(cfg *SessionConfig) SessionFactory {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	return &FactoryImpl{cfg: cfg}
}
