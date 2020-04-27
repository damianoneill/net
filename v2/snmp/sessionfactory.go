package snmp

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/imdario/mergo"
)

// Defines a factory method for instantiating SNMP Sessions.
type SessionFactory interface {
	// NewSession instantiates an SNMP session for managing the target device.
	NewSession(ctx context.Context, target string, opts ...SessionOption) (Session, error)
}

// Delivers a new session factory.
func NewFactory() SessionFactory {
	return &factoryImpl{}
}

type factoryImpl struct{}

func (f *factoryImpl) NewSession(ctx context.Context, target string, opts ...SessionOption) (Session, error) {

	config := defaultConfig
	config.address = target
	for _, opt := range opts {
		opt(&config)
	}

	mergo.Merge(config.trace, NoOpLoggingHooks) // nolint: gosec, errcheck

	conn, err := newConnection(ctx, &config)
	if err != nil {
		config.trace.Error("Network Connection", &config, err)
		return nil, err
	}

	return &sessionImpl{config: &config, conn: conn, nextRequestID: rand.Int31()}, nil
}

// SessionOption implements options for configuring session behaviour.
type SessionOption func(*sessionConfig)

// Timeout defines the timeout for receiving a response to a request
// Default value is 1s.
func Timeout(timeout time.Duration) SessionOption {
	return func(c *sessionConfig) {
		c.timeout = timeout
	}
}

// Retries defines the number of times an unsuccessful request will be retried.
// Default value is 0
func Retries(value int) SessionOption {
	return func(c *sessionConfig) {
		c.retries = value
	}
}

// Network defines the transport network.
// Default value is udp
func Network(value string) SessionOption {
	return func(c *sessionConfig) {
		c.network = value
	}
}

// Version defines the SNMP version to use.
// Default value is SNMPV2C
func Version(value SNMPVersion) SessionOption {
	return func(c *sessionConfig) {
		c.version = value
	}
}

// Commmunity defines the community string to be used.
// Default value is public.
func Community(value string) SessionOption {
	return func(c *sessionConfig) {
		c.community = value
	}
}

// LoggingHooks defines a set of logging hooks to be used by the session.
// Default value is DefaultLoggingHooks.
func LoggingHooks(trace *SessionTrace) SessionOption {
	return func(c *sessionConfig) {
		c.trace = trace
	}
}

// SNMP Versions.
type SNMPVersion int

const (
	SNMPV1  SNMPVersion = 0
	SNMPV2C SNMPVersion = 1
	SNMPV3  SNMPVersion = 3
)

// Deliver a new network connection to the address defined in the configuration.
func newConnection(ctx context.Context, m *sessionConfig) (conn net.Conn, err error) {
	m.trace.ConnectStart(m)
	defer m.trace.ConnectDone(m, err)
	return net.Dial(m.network, m.address)
}

// Defines properties controlling session behaviour.
type sessionConfig struct {
	// Connection network, typically udp.
	network string
	// Network address/hostname with port, for example: 10.48.24.234:161
	address string
	// SNMP version
	version SNMPVersion
	// community string for v2c.
	community string
	// Timeout for receiving a response
	timeout time.Duration
	// Defines the number of times an unsuccessful request will be retried.
	retries int
	// Trace hooks
	trace *SessionTrace
	// TODO Define additional configuration properties as required.
}

var defaultConfig = sessionConfig{
	network:   "udp",
	address:   "",
	community: "public",
	version:   SNMPV2C,
	timeout:   time.Second * 5,
	retries:   3,
	trace:     DefaultLoggingHooks,
}
