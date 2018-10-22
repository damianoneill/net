package netconf

import (
	"fmt"
	"runtime"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
)

// Defines credentials used for test sessions.
const (
	TestUserName = "testUser"
	TestPassword = "testPassword"
)

// TestNCServer represents a Netconf Server that can be used for 'on-board' testing.
// It encapsulates a transport connection to an SSH server, and a netconf session handler that will
// be invoked to handle netconf messages.
type TestNCServer struct {
	*testutil.SSHServer
	*netconfSessionHandler
}

// NewTestNetconfServer creates a new TestNCServer that will accept Netconf localhost connections on an ephemeral port (available
// via Port(), with credentials defined by TestUserName and TestPassword.
// tctx will be used for handling failures; if the supplied value is nil, a default test context will be used.
// The behaviour of the Netconf session handler can be conifgured using the WithCapabilities and
// WithRequestHandler methods.
func NewTestNetconfServer(tctx assert.TestingT) *TestNCServer {

	ncs := &TestNCServer{}

	ncs.netconfSessionHandler = newSessionHandler(ncs, 4)

	if tctx == nil {
		// Default test context to built-in implementation.
		tctx = ncs
	}
	ncs.SSHServer = testutil.NewSSHServerHandler(tctx, TestUserName, TestPassword, ncs.netconfSessionHandler)

	return ncs
}

// WithRequestHandler adds a request handler to the netconf session.
func (ncs *TestNCServer) WithRequestHandler(rh RequestHandler) *TestNCServer {
	ncs.netconfSessionHandler.reqHandlers = append(ncs.netconfSessionHandler.reqHandlers, rh)
	return ncs
}

// WithCapabilities define the capabilities that the server will advertise when a netconf client connects.
func (ncs *TestNCServer) WithCapabilities(caps []string) *TestNCServer {
	ncs.netconfSessionHandler.capabilities = caps
	return ncs
}

// Close closes any active transport to the test server and prevents subsequent connections.
func (ncs *TestNCServer) Close() {
	if ncs.netconfSessionHandler.ch != nil {
		ncs.netconfSessionHandler.ch.Close() // nolint: gosec, errcheck
		ncs.netconfSessionHandler.ch = nil
	}
	ncs.SSHServer.Close()
}

// Errorf provides testing.T compatibility if a test context is not provided when the test server is
// created.
func (ncs *TestNCServer) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// FailNow provides testing.T compatibility if a test context is not provided when the test server is
// created.
func (ncs *TestNCServer) FailNow() {
	runtime.Goexit()
}
