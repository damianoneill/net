package netconf

import (
	"fmt"
	"log"
	"os"
	"testing"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNewSession(t *testing.T) {

	sshConfig := &ssh.ClientConfig{
		User:            "WRuser",
		Auth:            []ssh.AuthMethod{ssh.Password("WRuser123")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
	assert.NoError(t, err, "Not expecting new transport to fail")
	defer tr.Close()

	l := log.New(os.Stderr, "logger:", log.Lshortfile)
	ncs, err := NewSession(tr, l, l)
	assert.NoError(t, err, "Not expecting new session to fail")
	assert.NotNil(t, ncs, "Session should be non-nil")

	reply, err := ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	// ncs.Execute(Request(`<get-config><source><running/></source><filter type="subtree"><top xmlns="http://example.com/schema/1.2/config"><users/></top></filter></get-config>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")

	reply, err = ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
}
