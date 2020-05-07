package snmp

import (
	"context"
	"net"

	"github.com/imdario/mergo"
)

// SewrverFactory defines an interface for instantiating SNMP Trap/Inform servers.
type ServerFactory interface {
	// NewServer instantiates an SNMP Trap/Inform server.
	NewServer(ctx context.Context, handler Handler, opts ...ServerOption) (Server, error)
}

// Delivers a new server factory.
func NewServerFactory() ServerFactory {
	return &serverFactoryImpl{}
}

type serverFactoryImpl struct{}

func (f *serverFactoryImpl) NewServer(ctx context.Context, handler Handler, opts ...ServerOption) (Server, error) {

	config := defaultServerConfig
	for _, opt := range opts {
		opt(&config)
	}

	config.resolveServerHooks()

	addr := &net.UDPAddr{Port: config.port, IP: net.ParseIP(config.address)}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	impl := &serverImpl{config: &config, conn: conn, handler: handler}
	impl.handleMessages()

	return impl, err
}

// ServerOption implements options for configuring server behaviour.
type ServerOption func(*serverConfig)

// ServerNetwork defines the transport network.
// Default value is udp
func ServerNetwork(value string) ServerOption {
	return func(c *serverConfig) {
		c.network = value
	}
}

// Address defines the address on which to listen.
// Default value is ""
func Address(value string) ServerOption {
	return func(c *serverConfig) {
		c.address = value
	}
}

// Port defines the port on which to listen.
// Default value is 162.
func Port(value int) ServerOption {
	return func(c *serverConfig) {
		c.port = value
	}
}

// Hooks defines a set of hooks to be invoked by the server.
// Default value is DefaultServerHooks.
func Hooks(trace *ServerHooks) ServerOption {
	return func(c *serverConfig) {
		c.trace = trace
	}
}

// Defines properties controlling server behaviour.
type serverConfig struct {
	// Connection network, typically udp.
	network string
	// Network address, for example: 10.48.24.234. Empty string means all interfaces.
	address string
	// Port number on which to listen, for example 162.
	port int
	// Trace hooks
	trace *ServerHooks
}

var defaultServerConfig = serverConfig{
	network: "udp",
	address: "",
	port:    162,
	trace:   DefaultServerHooks,
}

func (c *serverConfig) resolveServerHooks() {
	mergo.Merge(c.trace, NoOpServerHooks) // nolint: gosec, errcheck
}
