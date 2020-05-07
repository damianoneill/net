package snmp

import (
	"context"
	"net"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestNewServerSuccess(t *testing.T) {
	f := NewServerFactory()

	handler := &dummyHandler{}
	s, err := f.NewServer(context.Background(), handler)
	assert.NoError(t, err)
	assert.NotNil(t, s, "Server should not be nil")
	impl := s.(*serverImpl)
	assert.Equal(t, "", impl.config.address)
	assert.Equal(t, 162, impl.config.port)
	assert.Same(t, handler, impl.handler)
}

func TestNewServerOptions(t *testing.T) {
	f := NewServerFactory()
	s, err := f.NewServer(context.Background(), nil,
		ServerNetwork("udp"),
		Address("127.0.0.1"),
		Port(0),
		Hooks(NoOpServerHooks),
	)
	assert.NoError(t, err)
	assert.NotNil(t, s, "Server should not be nil")
	impl := s.(*serverImpl)
	assert.Equal(t, "udp", impl.config.network)
	assert.Equal(t, "127.0.0.1", impl.config.address)
	assert.Equal(t, 0, impl.config.port)
}

func TestListenFailure(t *testing.T) {
	f := NewServerFactory()
	s, err := f.NewServer(context.Background(), nil, Port(1000000000))
	assert.Error(t, err, "Expecting new server to fail - invalid port")
	assert.Nil(t, s, "Server should be nil")
}

type dummyHandler struct{}

func (h *dummyHandler) NewMessage(*PDU, bool, net.Addr) {
}
