package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/damianoneill/net/v2/netconf/testserver"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestAuthenticationFailure(t *testing.T) {
	ts := testserver.NewSSHServer(t, testserver.TestUserName, testserver.TestPassword)
	defer ts.Close()

	sshConfig := sshConfigWithPassword("WrongPassword")

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))

	assert.Error(t, err, "Expecting new session to fail - invalid password")
	assert.Nil(t, session, "Session should be nil")
}

func TestRequestPtyFailure(t *testing.T) {
	ts := testserver.NewSSHServerHandler(t, testserver.TestUserName, testserver.TestPassword,
		func(t assert.TestingT) testserver.SSHHandler {
			return &dummyShell{}
		},
		testserver.RequestTypes([]string{}))
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(), fmt.Sprintf("localhost:%d", ts.Port()))
	assert.Contains(t, err.Error(), "request pty failed")
	assert.Nil(t, session)
}

func TestShellFailure(t *testing.T) {
	ts := testserver.NewSSHServerHandler(t, testserver.TestUserName, testserver.TestPassword,
		func(t assert.TestingT) testserver.SSHHandler {
			return &dummyShell{}
		},
		testserver.RequestTypes([]string{"pty-req"}))
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(), fmt.Sprintf("localhost:%d", ts.Port()))
	assert.Contains(t, err.Error(), "login shell failed")
	assert.Nil(t, session)
}

func validSSHConfig() *ssh.ClientConfig {
	return sshConfigWithPassword(testserver.TestPassword)
}

func sshConfigWithPassword(pass string) *ssh.ClientConfig {
	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint: gosec
	}
	return sshConfig
}
