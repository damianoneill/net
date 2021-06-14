package netconf

import (
	"context"
	"fmt"
	"testing"

	"github.com/damianoneill/net/v2/netconf/ops"

	"github.com/damianoneill/net/v2/netconf/common"
	"github.com/damianoneill/net/v2/netconf/server/ssh"
	xssh "golang.org/x/crypto/ssh"

	assert "github.com/stretchr/testify/require"
)

// Defines credentials used for test sessions.
const (
	TestUserName = "testUser"
	TestPassword = "testPassword"
)

var sessionFactory = func(sh *SessionHandler) SessionCallback {
	fmt.Println("Session", sh.sid, sh.svrcon.Conn.RemoteAddr())
	return &callback{}
}

type callback struct{}

func (cb *callback) Capabilities() []string {
	return common.DefaultCapabilities
}

func (cb *callback) HandleRequest(req *RpcRequestMessage) *RpcReplyMessage {
	data := ReplyData{Data: responseFor(req)}

	errors := []common.RPCError{}
	return &RpcReplyMessage{
		Data: data, MessageID: req.MessageID,
		Errors: errors,
	}
}

func responseFor(req *RpcRequestMessage) string {
	switch req.Request.XMLName.Local {
	case "get":
		return `<top><sub attr="avalue"><child1>cvalue</child1><child2/></sub></top>`
	case "get-config":
		return `<top><sub attr="cfgval1"><child1>cfgval2</child1></sub></top>`
	// case "edit-config":
	//	etc...
	default:
		return req.Request.Body
	}
}

func TestServer(t *testing.T) {
	sshcfg, err := ssh.PasswordConfig(TestUserName, TestPassword)
	assert.NoError(t, err)

	ctx := WithTrace(context.Background(), DiagnosticLoggingHooks)
	ctx = ssh.WithSshTrace(ctx, ssh.DiagnosticLoggingHooks)
	server, err := NewServer(ctx, "localhost", 0, sshcfg, sessionFactory)
	assert.NotNil(t, server)
	assert.NoError(t, err)
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            TestUserName,
		Auth:            []xssh.AuthMethod{xssh.Password(TestPassword)},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ncs, err := ops.NewSession(context.Background(), sshConfig, fmt.Sprintf("%s:%d", "localhost", server.Port()))
	assert.NoError(t, err, "Not expecting new session to fail")
	defer ncs.Close()

	var result string
	err = ncs.GetSubtree("/", &result)
	assert.NoError(t, err, "Not expecting get to fail")
	assert.NotEmpty(t, result, "Reply should be non-nil")
	assert.Equal(t, `<top><sub attr="avalue"><child1>cvalue</child1><child2/></sub></top>`, result)

	err = ncs.GetConfigSubtree("/", ops.CandidateCfg, &result)
	assert.NoError(t, err, "Not expecting get-config to fail")
	assert.NotEmpty(t, result, "Reply should be non-nil")
	assert.Equal(t, `<top><sub attr="cfgval1"><child1>cfgval2</child1></sub></top>`, result)
}
