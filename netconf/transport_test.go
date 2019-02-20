package netconf

import (
	"bufio"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestSuccessfulConnection(t *testing.T) {

	ts := testutil.NewSSHServer(t, "testUser", "testPassword")
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := context.Background()
	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()
}

func TestFailingConnection(t *testing.T) {

	ts := testutil.NewSSHServer(t, "testUser", "testPassword")
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("wrongPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := context.Background()
	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
	assert.Error(t, err, "Not expecting new transport to succeed")
	assert.Nil(t, tr, "Transport should not be defined")
}

func TestWriteRead(t *testing.T) {

	ts := testutil.NewSSHServer(t, "testUser", "testPassword")
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	ctx := context.Background()
	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	rdr := bufio.NewReader(tr)
	tr.Write([]byte("Message\n"))
	response, _ := rdr.ReadString('\n')
	assert.Equal(t, "GOT:Message\n", response, "Failed to get expected response")
}

func TestTrace(t *testing.T) {

	ts := testutil.NewSSHServer(t, "testUser", "testPassword")
	defer ts.Close()

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	traces := []string{}
	trace := &ClientTrace{
		ConnectStart: func(clientConfig *ssh.ClientConfig, target string) {
			traces = append(traces, fmt.Sprintf("ConnectStart %s", target))
		},
		ConnectDone: func(clientConfig *ssh.ClientConfig, target string, err error, d time.Duration) {
			traces = append(traces, fmt.Sprintf("ConnectDone %s error:%v", target, err))
			assert.True(t, d > 0, "Duration should be defined")
		},
		ConnectionClosed: func(target string, err error) {
			traces = append(traces, fmt.Sprintf("ConnectionClosed target:%s error:%v", target, err))
		},
		ReadStart: func(p []byte) {
			traces = append(traces, "ReadStart called")

		},
		ReadDone: func(p []byte, c int, err error, d time.Duration) {
			traces = append(traces, fmt.Sprintf("ReadDone %s %d %v", string(p[:c]), c, err))
			assert.True(t, d > 0, "Duration should be defined")
		},
		WriteStart: func(p []byte) {
			traces = append(traces, fmt.Sprintf("WriteStart %s", p))
		},
		WriteDone: func(p []byte, c int, err error, d time.Duration) {
			traces = append(traces, fmt.Sprintf("WriteDone %s %d %v", string(p[:c]), c, err))
			assert.True(t, d > 0, "Duration should be defined")
		},
	}

	ctx := WithClientTrace(context.Background(), trace)
	tr, _ := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")

	tr.Write([]byte("Message\n"))
	bufio.NewReader(tr).ReadString('\n')

	tr.Close()

	assert.Equal(t, fmt.Sprintf("ConnectStart localhost:%d", ts.Port()), traces[0])
	assert.Equal(t, fmt.Sprintf("ConnectDone localhost:%d error:<nil>", ts.Port()), traces[1])
	assert.Equal(t, "WriteStart Message\n", traces[2])
	assert.Equal(t, "WriteDone Message\n 8 <nil>", traces[3])
	assert.Equal(t, "ReadStart called", traces[4])
	assert.Equal(t, "ReadDone GOT:Message\n 12 <nil>", traces[5])
	assert.Contains(t, traces[6], fmt.Sprintf("ConnectionClosed target:localhost:"), traces[6])
}
