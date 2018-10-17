package netconf

import (
	"context"
	"fmt"
	"testing"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Simple real NE access tests

func TestTransportFailure(t *testing.T) {

	s, err := NewRPCSession(context.Background(), &ssh.ClientConfig{}, "localhost:0")
	assert.Error(t, err, "Expecting new session to fail")
	assert.Nil(t, s, "Session should be nil")
}

func TestSessionSetupFailure(t *testing.T) {

	ts := testutil.NewSSHServer(t, "testUser", "testPassword")
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	s, err := NewRPCSessionWithConfig(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &ClientConfig{setupTimeoutSecs: 1})
	assert.Error(t, err, "Expecting new session to fail - no hello from server")
	assert.Nil(t, s, "Session should be nil")
}

func TestSessionSetupSuccess(t *testing.T) {

	handler := newHandler(t, 4)
	ts := testutil.NewSSHServerHandler(t, "testUser", "testPassword", handler)
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := WithClientTrace(context.Background(), DiagnosticLoggingHooks)
	s, err := NewRPCSessionWithConfig(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &ClientConfig{setupTimeoutSecs: 1})
	assert.NoError(t, err, "Expecting new session to succeed")
	assert.NotNil(t, s, "Session should not be nil")
}

// Simple real NE access test

// func TestRealNewSession(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "XXxxxx",
// 		Auth:            []ssh.AuthMethod{ssh.Password("XXxxxxxxx")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	s, err := NewRPCSession(WithClientTrace(context.Background(), DefaultLoggingHooks), sshConfig, fmt.Sprintf("172.26.138.57:%d", 830))
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, s, "Session should be non-nil")

// 	defer s.Close()

// 	reply, err := s.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 	assert.NoError(t, err, "Not expecting exec to fail")
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// }
