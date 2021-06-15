//nolint: goconst,gosec
package netconf

import (
	"context"
	"fmt"

	"github.com/damianoneill/net/v2/netconf/common"

	"github.com/damianoneill/net/v2/netconf/ops"
	"github.com/damianoneill/net/v2/netconf/server/ssh"
	xssh "golang.org/x/crypto/ssh"
)

type exampleServer struct{}

func (es *exampleServer) Capabilities() []string {
	return common.DefaultCapabilities
}

func (es *exampleServer) HandleRequest(req *RPCRequestMessage) *RPCReplyMessage {
	switch req.Request.XMLName.Local {
	case "get":
		return &RPCReplyMessage{Data: ReplyData{Data: `<top><sub attr="avalue"><child1>cvalue</child1></sub></top>`}, MessageID: req.MessageID}
	case "get-config":
		return &RPCReplyMessage{Errors: []common.RPCError{
			{Severity: "error", Message: "oops"},
		}, MessageID: req.MessageID}
	}
	return nil
}

func ExampleNewServer() {
	sshcfg, _ := ssh.PasswordConfig("UserA", "PassA")
	server, _ := NewServer(context.Background(), "localhost", 0, sshcfg,
		func(sh *SessionHandler) SessionCallback {
			return &exampleServer{}
		})
	defer server.Close()

	//----------------------------

	sshConfig := &xssh.ClientConfig{
		User:            "UserA",
		Auth:            []xssh.AuthMethod{xssh.Password("PassA")},
		HostKeyCallback: xssh.InsecureIgnoreHostKey(),
	}

	ncs, _ := ops.NewSession(context.Background(), sshConfig, fmt.Sprintf("%s:%d", "localhost", server.Port()))
	defer ncs.Close()

	var result string
	_ = ncs.GetSubtree("/", &result)
	fmt.Println("Get:", result)

	err := ncs.GetConfigSubtree("/", ops.CandidateCfg, &result)
	fmt.Println("Get-Config:", err)

	// Output: Get: <top><sub attr="avalue"><child1>cvalue</child1></sub></top>
	// Get-Config: netconf rpc [error] 'oops'
}
