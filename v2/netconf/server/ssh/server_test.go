//nolint:dupl
package ssh

import (
	"context"
	"fmt"
	"testing"

	"github.com/damianoneill/net/v2/netconf/client"

	xssh "golang.org/x/crypto/ssh"

	assert "github.com/stretchr/testify/require"
)

// Defines credentials used for test sessions.
const (
	TestUserName = "testUser"
	TestPassword = "testPassword"
)

type sHandler struct{}

func (s *sHandler) Handle(ch xssh.Channel) {
	buffer := make([]byte, 5)
	_, _ = ch.Read(buffer)
	_, _ = ch.Write([]byte(">" + string(buffer) + "<"))
}

func handlerFactory() HandlerFactory {
	return func(svrconn *xssh.ServerConn) Handler {
		return &sHandler{}
	}
}

func TestServer(t *testing.T) {
	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSSHTrace(context.Background(), DefaultLoggingHooks)
	server, err := NewServer(ctx, "localhost", 0, sshcfg, handlerFactory())
	assert.NotNil(t, server)
	assert.NoError(t, err)
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            TestUserName,
		Auth:            []xssh.AuthMethod{xssh.Password(TestPassword)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ctx = context.Background()
	target := fmt.Sprintf("localhost:%d", server.Port())
	tr, err := client.NewSSHTransport(ctx, client.NewDialer(target, sshConfig), target)
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	_, _ = tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	_, _ = tr.Read(buffer)
	assert.Equal(t, ">hello<", string(buffer))
}

func TestServerListenFailure(t *testing.T) {
	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSSHTrace(context.Background(), DefaultLoggingHooks)
	server, err := NewServer(ctx, "9.9.9.9", 9999, sshcfg, handlerFactory())
	assert.Nil(t, server)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "assign requested address")
}

func TestServerConnectionFailure(t *testing.T) {
	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSSHTrace(context.Background(), DefaultLoggingHooks)
	server, err := NewServer(ctx, "localhost", 0, sshcfg, handlerFactory())
	assert.NotNil(t, server)
	assert.NoError(t, err)
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            TestUserName,
		Auth:            []xssh.AuthMethod{xssh.Password("WrongPassword")},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ctx = context.Background()
	target := fmt.Sprintf("localhost:%d", server.Port())
	_, err = client.NewSSHTransport(ctx, client.NewDialer(target, sshConfig), target)
	assert.Error(t, err, "Not expecting new transport to succeed")
	assert.Contains(t, err.Error(), "authenticate")
}

func TestServerDiagnosticTraceHooks(t *testing.T) {
	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSSHTrace(context.Background(), DiagnosticLoggingHooks)
	server, err := NewServer(ctx, "localhost", 0, sshcfg, handlerFactory())
	assert.NotNil(t, server)
	assert.NoError(t, err)
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            TestUserName,
		Auth:            []xssh.AuthMethod{xssh.Password(TestPassword)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ctx = context.Background()
	target := fmt.Sprintf("localhost:%d", server.Port())
	tr, err := client.NewSSHTransport(ctx, client.NewDialer(target, sshConfig), target)
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	_, _ = tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	_, _ = tr.Read(buffer)
	assert.Equal(t, ">hello<", string(buffer))
}

func TestServerNoOpTraceHooks(t *testing.T) {
	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := context.Background()
	server, err := NewServer(ctx, "localhost", 0, sshcfg, handlerFactory())
	assert.NotNil(t, server)
	assert.NoError(t, err)
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            TestUserName,
		Auth:            []xssh.AuthMethod{xssh.Password(TestPassword)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ctx = context.Background()
	target := fmt.Sprintf("localhost:%d", server.Port())
	tr, err := client.NewSSHTransport(ctx, client.NewDialer(target, sshConfig), target)
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	_, _ = tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	_, _ = tr.Read(buffer)
	assert.Equal(t, ">hello<", string(buffer))
}
