package cli

import (
	"bufio"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/damianoneill/net/v2/netconf/testserver"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestSessionWithFailingShell(t *testing.T) {
	_, ts := dummyServerWithFailingShell(t)
	defer ts.Close()

	sshConfig := validSSHConfig()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig,
		fmt.Sprintf("localhost:%d", ts.Port()), WithCommands("Init1", "Init2"))
	assert.Contains(t, err.Error(), "EOF")
	assert.Nil(t, session, "Session should be nil")
}

func TestSessionSetupWithInitCommands(t *testing.T) {
	dummySh, ts := dummyServer(t)
	defer ts.Close()

	sshConfig := validSSHConfig()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig,
		fmt.Sprintf("localhost:%d", ts.Port()), WithCommands("Init1", "Init2"))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	assert.Equal(t, "Init1\n", dummySh.lines[0])
	assert.Equal(t, "Init2\n", dummySh.lines[1])
}

func TestSessionSetupWithFailingInitCommands(t *testing.T) {
	dummySh, ts := dummyServer(t)
	defer ts.Close()

	sshConfig := validSSHConfig()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig,
		fmt.Sprintf("localhost:%d", ts.Port()), WithCommands("Init1", "close", "Init2"))
	assert.Contains(t, err.Error(), "EOF")
	assert.Nil(t, session, "Session should be nil")

	assert.Len(t, dummySh.lines, 2)
	assert.Equal(t, "Init1\n", dummySh.lines[0])
	assert.Equal(t, "close\n", dummySh.lines[1])
}

func TestSessionSetupWithTimeout(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	sshConfig := validSSHConfig()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig,
		fmt.Sprintf("localhost:%d", ts.Port()), WithTimeout(time.Millisecond*250))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()
	assert.Equal(t, time.Millisecond*250, session.(*SessionImpl).cfg.readTimeout)
}

func TestSessionSetupInvalidOptions(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	sshConfig := validSSHConfig()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), sshConfig,
		fmt.Sprintf("localhost:%d", ts.Port()), WithPrompt("BadRegex("))
	assert.Contains(t, err.Error(), "invalid prompt pattern")
	assert.Nil(t, session)
}

type dummyShell struct {
	// Prompt that should be emitted.
	prompt string
	// Record of commands received.
	lines []string
	// Signals that shell should close immediately.
	fail bool
}

const defaultPrompt = "ABC> "

// Simple Handler implementation that echoes lines.
func (e *dummyShell) Handle(t assert.TestingT, ch ssh.Channel) {
	if e.fail {
		_ = ch.Close()
		return
	}
	chReader := bufio.NewReader(ch)
	chWriter := bufio.NewWriter(ch)
	prompt := e.prompt
	if prompt == "" {
		prompt = defaultPrompt
	}
	_, _ = chWriter.WriteString(prompt)
	chWriter.Flush()
	for {
		input, err := chReader.ReadString('\n')
		if err != nil {
			return
		}
		e.lines = append(e.lines, input)

		switch input {
		case "enable\n":
			_, _ = chWriter.WriteString("\nPassword: ")
			_ = chWriter.Flush()
			prompt = "ABC# "
		case "close\n":
			_ = ch.Close()
			return
		default:
			_, err = chWriter.WriteString(fmt.Sprintf("GOT:%s\n", input))
			assert.NoError(t, err, "Write failed")
			_, _ = chWriter.WriteString(prompt)
			err = chWriter.Flush()
			assert.NoError(t, err, "Flush failed")
		}
	}
}

func dummyServer(t *testing.T) (*dummyShell, *testserver.SSHServer) {
	return dummyServerWithPrompt(t, "")
}

func dummyServerWithPrompt(t *testing.T, prompt string) (*dummyShell, *testserver.SSHServer) {
	dummySh := &dummyShell{prompt: prompt}
	ts := testserver.NewSSHServerHandler(t, testserver.TestUserName, testserver.TestPassword,
		func(t assert.TestingT) testserver.SSHHandler {
			return dummySh
		},
		testserver.RequestTypes([]string{"pty-req", "shell"}))
	return dummySh, ts
}

func dummyServerWithFailingShell(t *testing.T) (*dummyShell, *testserver.SSHServer) {
	dummySh := &dummyShell{fail: true}
	ts := testserver.NewSSHServerHandler(t, testserver.TestUserName, testserver.TestPassword,
		func(t assert.TestingT) testserver.SSHHandler {
			return dummySh
		},
		testserver.RequestTypes([]string{"pty-req", "shell"}))
	return dummySh, ts
}

//nolint: gocritic
//Simple real NE access tests
//
//func TestRealNewSession(tport *testing.T) {
//
//	factory := NewSessionFactory(nil)
//
//	sshConfig := &ssh.ClientConfig{
//		User:            "cisco",
//		Auth:            []ssh.AuthMethod{ssh.Password("cisco")},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//		Config: ssh.Config{
//			KeyExchanges: []string{"diffie-hellman-group1-sha1", "diffie-hellman-group14-sha1"},
//			Ciphers:      []string{"aes128-cbc", "aes128-ctr"},
//		},
//	}
//
//	//devs := []string{"172.26.138.72"}
//	devs := []string{"172.26.138.73", "172.26.138.52", "172.26.138.72", "10.48.36.115"}
//	for _, d := range devs {
//		session, err := factory.NewSession(context.Background(), sshConfig, d+":22", WithCommands([]string{"terminal length 0"}))
//		assert.NoError(tport, err, "failed to create session")
//		assert.NotNil(tport, session)
//		defer session.Close()
//
//		b, err := session.Send("show running-config")
//		assert.NoError(tport, err, "failed to send command")
//
//		fmt.Println("running config >" + b + "<")
//	}
//}
//
//func TestRealNewSession2(tport *testing.T) {
//
//	factory := NewSessionFactory(nil)
//
//	sshConfig := &ssh.ClientConfig{
//		User:            "root",
//		Auth:            []ssh.AuthMethod{ssh.Password("Be1fast")},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	}
//
//	session, err := factory.NewSession(context.Background(), sshConfig, "atomic83:22")
//	assert.NoError(tport, err, "failed to create session")
//	assert.NotNil(tport, session)
//	defer session.Close()
//
//	b, err := session.Send("ls")
//	assert.NoError(tport, err, "failed to send command")
//
//	fmt.Println("ls >" + b + "<")
//}
//
//func TestRealNewSession3(tport *testing.T) {
//
//	factory := NewSessionFactory(nil)
//
//	sshConfig := &ssh.ClientConfig{
//		User:            "cisco",
//		Auth:            []ssh.AuthMethod{ssh.Password("cisco")},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//		Config: ssh.Config{
//			KeyExchanges: []string{"diffie-hellman-group1-sha1"},
//			Ciphers:      []string{"aes128-cbc"},
//		},
//	}
//
//	devs := []string{"172.26.138.50"}
//	for _, d := range devs {
//		session, err := factory.NewSession(context.Background(), sshConfig, d+":22", WithCommands([]string{"terminal length 0"}))
//		assert.NoError(tport, err, "failed to create session")
//		assert.NotNil(tport, session)
//		defer session.Close()
//		_, err = session.Send("enable", WaitFor("Password: "))
//		assert.NoError(tport, err)
//		_, err = session.Send("Be1fast", ResetPrompt())
//		b, err := session.Send("show running-config")
//		assert.NoError(tport, err, "failed to send command")
//
//		fmt.Println("running config >" + b + "<")
//	}
//}
