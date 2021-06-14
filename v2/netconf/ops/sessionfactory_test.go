package ops

import (
	"context"
	"fmt"
	"testing"

	"github.com/damianoneill/net/v2/netconf/client"

	"github.com/damianoneill/net/v2/netconf/testserver"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestTransportFailure(t *testing.T) {
	s, err := NewSession(context.Background(), &ssh.ClientConfig{}, "localhost:0")
	assert.Error(t, err, "Expecting new session to fail")
	assert.Nil(t, s, "OpSession should be nil")
}

func TestSessionSetupFailure(t *testing.T) {
	ts := testserver.NewSSHServer(t, testserver.TestUserName, testserver.TestPassword)
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := client.WithClientTrace(context.Background(), client.DefaultLoggingHooks)
	s, err := NewSessionWithConfig(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &client.Config{SetupTimeoutSecs: 1})
	assert.Error(t, err, "Expecting new session to fail - no hello from server")
	assert.Nil(t, s, "OpSession should be nil")
}

func TestSessionSetupSuccess(t *testing.T) {
	ts := testserver.NewTestNetconfServer(t)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	s, err := NewSessionWithConfig(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &client.Config{SetupTimeoutSecs: 1})
	assert.NoError(t, err, "Expecting new session to succeed")
	assert.NotNil(t, s, "OpSession should not be nil")
}

// Simple real NE access test

// func TestRealNewSession(t *testing.T) {

// 	sshConfig := &ssh.Config{
// 		User:            "XXxxxx",
// 		Auth:            []ssh.AuthMethod{ssh.Password("XXxxxxxxx")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	s, err := NewRPCSession(WithClientTrace(context.Background(), DefaultLoggingHooks), sshConfig, fmt.Sprintf("172.26.138.57:%d", 830))
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, s, "OpSession should be non-nil")

// 	defer s.Close()

// 	reply, err := s.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 	assert.NoError(t, err, "Not expecting exec to fail")
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// }
