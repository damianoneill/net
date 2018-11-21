package netconf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestTransportFailure(t *testing.T) {

	s, err := NewRPCSession(context.Background(), &ssh.ClientConfig{}, "localhost:0")
	assert.Error(t, err, "Expecting new session to fail")
	assert.Nil(t, s, "Session should be nil")
}

func TestSessionSetupFailure(t *testing.T) {

	ts := testutil.NewSSHServer(t, TestUserName, TestPassword)
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := WithClientTrace(context.Background(), DefaultLoggingHooks)
	s, err := NewRPCSessionWithConfig(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &ClientConfig{setupTimeoutSecs: 1})
	assert.Error(t, err, "Expecting new session to fail - no hello from server")
	assert.Nil(t, s, "Session should be nil")
}

func TestSessionSetupSuccess(t *testing.T) {

	ts := NewTestNetconfServer(t)

	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	s, err := NewRPCSessionWithConfig(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), &ClientConfig{setupTimeoutSecs: 1})
	assert.NoError(t, err, "Expecting new session to succeed")
	assert.NotNil(t, s, "Session should not be nil")
}

func TestSessionWithHooks(t *testing.T) {

	logged := exerciseSession(t, NoOpLoggingHooks)
	assert.Equal(t, "", logged, "Nothing should be logged")

	logged = exerciseSession(t, DefaultLoggingHooks)
	assert.NotEqual(t, "", logged, "Something should be logged")
	assert.Contains(t, logged, "Error context", "Error should be logged")
	assert.NotContains(t, logged, "ConnectStart", "ConnectStart should not be logged")
	assert.NotContains(t, logged, "ReadDone", "ReadDone should not be logged")

	logged = exerciseSession(t, DiagnosticLoggingHooks)
	assert.NotEqual(t, "", logged, "Something should be logged")
	assert.Contains(t, logged, "Error context", "Error should be logged")
	assert.Contains(t, logged, "ReadDone", "ReadDone should be logged")
}

func exerciseSession(t *testing.T, hooks *ClientTrace) string {

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	log.SetOutput(w)

	ts := NewTestNetconfServer(t).WithRequestHandler(EchoRequestHandler).WithRequestHandler(EchoRequestHandler).WithRequestHandler(EchoRequestHandler).WithRequestHandler(CloseRequestHandler)
	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := context.Background()
	if hooks != nil {
		ctx = WithClientTrace(ctx, hooks)
	}
	s, _ := NewRPCSession(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	sh := ts.SessionHandler(s.ID())

	reply, _ := s.Execute(Request("<get/>"))
	assert.NotNil(t, reply, "Execute failed unexpectedly")

	rch := make(chan *RPCReply)
	s.ExecuteAsync(Request("<get/>"), rch)
	reply = <-rch
	assert.NotNil(t, reply, "ExecuteAsync failed unexpectedly")

	nch := make(chan *Notification)
	reply, _ = s.Subscribe("<create-subscription/>", nch)
	assert.NotNil(t, reply, "Subscribe failed unexpectedly")

	time.AfterFunc(time.Duration(100)*time.Millisecond, func() { sh.SendNotification("<eventA/>") })

	nmsg := <-nch
	assert.NotNil(t, nmsg, "Failed to receive notification")

	sh.SendNotification("<eventB/>") // Should be dropped

	ts.WithRequestHandler(CloseRequestHandler) // Force error on next request
	reply, _ = s.Execute(Request("<get/>"))
	assert.Nil(t, reply, "Execute succeeded unexpectedly")

	s.Close()

	w.Flush()
	return b.String()
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
