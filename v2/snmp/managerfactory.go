package snmp

import (
	"context"
	"net"
	"time"

	"github.com/imdario/mergo"
)

// Defines a factory method for instantiating SNMP Managers.
type ManagerFactory interface {
	// NewManager instantiates an SNMP manager for managing the target device.
	NewManager(ctx context.Context, target string, opts ...ManagerOption) (Manager, error)
}

// Delivers a new manager factory.
func NewFactory() ManagerFactory {
	return &factoryImpl{}
}

type factoryImpl struct{}

func (f *factoryImpl) NewManager(ctx context.Context, target string, opts ...ManagerOption) (Manager, error) {

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
	return &managerImpl{config: &config, conn: conn}, nil
}

// ManagerOption implements options for configuring manager behaviour.
type ManagerOption func(*managerConfig)

// Timeout defines the timeout for receiving a response to a request
// Default value is 1s.
func Timeout(timeout time.Duration) ManagerOption {
	return func(c *managerConfig) {
		c.timeout = timeout
	}
}

// Retries defines the number of times an unsuccessful request will be retried.
// Default value is 0
func Retries(value int) ManagerOption {
	return func(c *managerConfig) {
		c.retries = value
	}
}

// Network defines the transport network.
// Default value is udp
func Network(value string) ManagerOption {
	return func(c *managerConfig) {
		c.network = value
	}
}

// Version defines the SNMP version to use.
// Default value is SNMPV2C
func Version(value SNMPVersion) ManagerOption {
	return func(c *managerConfig) {
		c.version = value
	}
}

// Commmunity defines the community string to be used.
// Default value is public.
func Community(value string) ManagerOption {
	return func(c *managerConfig) {
		c.community = value
	}
}

// LoggingHooks defines a set of logging hooks to be used by the manager.
// Default value is DefaultLoggingHooks.
func LoggingHooks(trace *ManagerTrace) ManagerOption {
	return func(c *managerConfig) {
		c.trace = trace
	}
}

// SNMP Versions.
type SNMPVersion int

const (
	SNMPV1 SNMPVersion = iota
	SNMPV2C
	SNMPV3
)

// Deliver a new network connection to the address defined in the configuration.
func newConnection(ctx context.Context, m *managerConfig) (conn net.Conn, err error) {
	m.trace.ConnectStart(m)
	defer m.trace.ConnectDone(m, err)
	return net.Dial(m.network, m.address)
}

// Defines properties controlling manager behaviour.
type managerConfig struct {
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
	trace *ManagerTrace
	// TODO Define additional configuration properties as required.
}

var defaultConfig = managerConfig{
	network:   "udp",
	address:   "",
	community: "public",
	version:   SNMPV2C,
	timeout:   time.Second,
	retries:   0,
	trace:     DefaultLoggingHooks,
}
