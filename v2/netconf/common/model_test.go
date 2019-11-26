package common

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)

type testStr struct {
	Field string
}

func TestRPCErrorString(t *testing.T) {

	err := &RPCError{
		Severity: "Severity",
		Message:  "Message",
	}

	assert.Equal(t, "netconf rpc [Severity] 'Message'", err.Error())
}


func TestPeerSupportsChunkedFraming(t *testing.T) {
	assert.False(t, PeerSupportsChunkedFraming([]string {NetconfNS, NetconfNotifyNS, CapBase10}))
	assert.True(t, PeerSupportsChunkedFraming([]string {NetconfNS, NetconfNotifyNS, CapBase11}))
}