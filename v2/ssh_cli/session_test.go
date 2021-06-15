package cli

import (
	"context"
	"fmt"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestSessionSendDefault(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(), fmt.Sprintf("localhost:%d", ts.Port()))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	resp, err := session.Send("Command")
	assert.NoError(t, err)
	assert.Equal(t, "GOT:Command\n", resp)
}

func TestSessionSendAndWait(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(), fmt.Sprintf("localhost:%d", ts.Port()))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	resp, err := session.Send("Command")
	assert.NoError(t, err)
	assert.Equal(t, "GOT:Command\n", resp)

	resp, err = session.Send("enable", WaitFor("Password: $"))
	assert.NoError(t, err)
	assert.Empty(t, resp)

	resp, err = session.Send("EPASS", ResetPrompt())
	assert.NoError(t, err)
	assert.Empty(t, resp)

	resp, err = session.Send("Command2")
	assert.NoError(t, err)
	assert.Equal(t, "GOT:Command2\n", resp)

	resp, err = session.Send("enable", WaitFor("BadRegex)"))
	assert.Contains(t, err.Error(), "invalid WaitFor value")
	assert.Empty(t, resp)
}

func TestSessionSendOptions(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(), fmt.Sprintf("localhost:%d", ts.Port()))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	resp, err := session.Send("enable", WaitFor("Password: "))
	assert.NoError(t, err)
	assert.Empty(t, resp)

	resp, err = session.Send("EPASS", ResetPrompt())
	assert.NoError(t, err)
	assert.Empty(t, resp)

	_, err = session.Send("Command ", NoNewline(), NoWait())
	assert.NoError(t, err)
	resp, err = session.Send("Param")
	assert.NoError(t, err)
	assert.Equal(t, "GOT:Command Param\n", resp)
}

func TestSessionWithNoPrompt(t *testing.T) {
	_, ts := dummyServerWithPrompt(t, "Special> ")
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(),
		fmt.Sprintf("localhost:%d", ts.Port()),
		WithPrompt(""))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	// Lose the initial prompt.
	_, err = session.Send("", WaitFor("Special> "))
	assert.NoError(t, err)

	resp, err := session.Send("command")
	assert.Contains(t, err.Error(), "need to specify WaitFor")
	assert.Empty(t, resp)

	resp, err = session.Send("command", WaitFor("Special> "))
	assert.NoError(t, err)
	assert.Equal(t, "GOT:command\n", resp)
}

func TestSessionWithPrompt(t *testing.T) {
	_, ts := dummyServer(t)
	defer ts.Close()

	factory := NewSessionFactory(nil)

	session, err := factory.NewSession(context.Background(), validSSHConfig(),
		fmt.Sprintf("localhost:%d", ts.Port()),
		WithPrompt("A.C> $"))
	assert.NoError(t, err)
	assert.NotNil(t, session, "Session should not be nil")
	defer session.Close()

	resp, err := session.Send("command")
	assert.NoError(t, err)
	assert.Equal(t, "GOT:command\n", resp)
}
