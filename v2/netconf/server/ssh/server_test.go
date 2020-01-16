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
	ch.Read(buffer)
	ch.Write([]byte(">" + string(buffer) + "<"))
}

func handlerFactory() HandlerFactory {
	return func(svrconn *xssh.ServerConn) Handler {
		return &sHandler{}
	}
}

func TestServer(t *testing.T) {

	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSshTrace(context.Background(), DefaultLoggingHooks)
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
	tr, err := client.NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", server.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	tr.Read(buffer)
	assert.Equal(t, ">hello<", string(buffer))
}

func TestServerListenFailure(t *testing.T) {

	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSshTrace(context.Background(), DefaultLoggingHooks)
	server, err := NewServer(ctx, "9.9.9.9", 9999, sshcfg, handlerFactory())
	assert.Nil(t, server)
	assert.Error(t, err)
}

func TestServerConnectionFailure(t *testing.T) {

	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSshTrace(context.Background(), DefaultLoggingHooks)
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
	_, err = client.NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", server.Port()), "netconf")
	assert.Error(t, err, "Not expecting new transport to succeed")
	assert.Contains(t, err.Error(), "authenticate")
}

func TestServerDiagnosticTraceHooks(t *testing.T) {

	sshcfg, err := PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithSshTrace(context.Background(), DiagnosticLoggingHooks)
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
	tr, err := client.NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", server.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	tr.Read(buffer)
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
	tr, err := client.NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", server.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	tr.Write([]byte("hello"))
	buffer := make([]byte, 7)
	tr.Read(buffer)
	assert.Equal(t, ">hello<", string(buffer))
}
