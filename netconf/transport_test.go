package netconf

import (
	"bufio"
	"fmt"
	"testing"

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

	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
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

	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
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

	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	rdr := bufio.NewReader(tr)
	tr.Write([]byte("Message\n"))
	response, _ := rdr.ReadString('\n')
	assert.Equal(t, "GOT:Message\n", response, "Failed to get expected response")
}
