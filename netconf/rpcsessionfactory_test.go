package netconf

import (
	"fmt"
	"testing"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Simple real NE access tests

func TestTransportFailure(t *testing.T) {

	s, err := NewRPCSession(&ssh.ClientConfig{}, "localhost:0")
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

	s, err := NewRPCSessionWithConfig(sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &ClientConfig{setupTimeoutSecs: 1})
	assert.Error(t, err, "Expecting new session to fail")
	assert.Nil(t, s, "Session should be nil")
}

// Simple real NE access test

// func TestRealNewSession(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "WRuser",
// 		Auth:            []ssh.AuthMethod{ssh.Password("WRuser123")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	s, err := NewRpcSession(sshConfig, fmt.Sprintf("172.26.138.57:%d", 830))
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, s, "Session should be non-nil")

// 	defer s.Close()

// 	reply, err := s.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 	assert.NoError(t, err, "Not expecting exec to fail")
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// }
