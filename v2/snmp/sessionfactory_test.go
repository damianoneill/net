package snmp

import (
	"context"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestNewManagerSuccess(t *testing.T) {
	f := NewFactory()
	m, err := f.NewSession(context.Background(), "localhost:161")
	assert.NoError(t, err)
	assert.NotNil(t, m, "Session should not be nil")
}

func TestNewManagerOptions(t *testing.T) {
	f := NewFactory()
	m, err := f.NewSession(context.Background(), "localhost:161",
		Network("udp"),
		Timeout(time.Second),
		Retries(5),
		Version(SNMPV2C),
		Community("public"),
		LoggingHooks(DiagnosticLoggingHooks),
	)
	assert.NoError(t, err)
	assert.NotNil(t, m, "Session should not be nil")
	impl := m.(*sessionImpl)
	assert.Equal(t, "localhost:161", impl.config.address)
	assert.Equal(t, "udp", impl.config.network)
	assert.Equal(t, time.Second, impl.config.timeout)
	assert.Equal(t, 5, impl.config.retries)
	assert.Equal(t, SNMPV2C, impl.config.version)
	assert.Equal(t, "public", impl.config.community)
}

func TestConnectionFailure(t *testing.T) {
	f := NewFactory()
	m, err := f.NewSession(context.Background(), "nosuchhost:161")
	assert.Error(t, err, "Expecting new session to fail - invalid port")
	assert.Nil(t, m, "Session should be nil")
}
