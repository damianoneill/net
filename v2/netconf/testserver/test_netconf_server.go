package testserver

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"github.com/damianoneill/net/v2/netconf/common"

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
	*SSHServer
	sessionHandlers map[uint64]*SessionHandler
	reqHandlers     []RequestHandler
	caps            []string
	nextSid         uint64
	tctx            assert.TestingT
}

// NewTestNetconfServer creates a new TestNCServer that will accept Netconf localhost connections on an ephemeral port (available
// via Port(), with credentials defined by TestUserName and TestPassword.
// tctx will be used for handling failures; if the supplied value is nil, a default test context will be used.
// The behaviour of the Netconf session handler can be conifgured using the WithCapabilities and
// WithRequestHandler methods.
func NewTestNetconfServer(tctx assert.TestingT) *TestNCServer {
	ncs := &TestNCServer{sessionHandlers: make(map[uint64]*SessionHandler), caps: common.DefaultCapabilities}

	if tctx == nil {
		// Default test context to built-in implementation.
		tctx = ncs
	}
	ncs.tctx = tctx

	ncs.SSHServer = NewSSHServerHandler(tctx, TestUserName, TestPassword, ncs.newFactory())

	return ncs
}

func (ncs *TestNCServer) newFactory() HandlerFactory {
	return func(t assert.TestingT) SSHHandler {
		sid := atomic.AddUint64(&ncs.nextSid, 1)
		sess := newSessionHandler(ncs, sid)
		ncs.sessionHandlers[sid] = sess
		sess.capabilities = ncs.caps
		sess.reqHandlers = ncs.reqHandlers
		return sess
	}
}

// LastHandler delivers the most recently instantiated session handler.
func (ncs *TestNCServer) LastHandler() *SessionHandler {
	return ncs.sessionHandlers[ncs.nextSid]
}

// WithRequestHandler adds a request handler to the netconf session.
func (ncs *TestNCServer) WithRequestHandler(rh RequestHandler) *TestNCServer {
	ncs.reqHandlers = append(ncs.reqHandlers, rh)
	return ncs
}

// WithCapabilities define the capabilities that the server will advertise when a netconf client connects.
func (ncs *TestNCServer) WithCapabilities(caps []string) *TestNCServer {
	ncs.caps = caps
	return ncs
}

// Close closes any active transport to the test server and prevents subsequent connections.
func (ncs *TestNCServer) Close() {
	for k, v := range ncs.sessionHandlers {
		if v.ch != nil {
			v.Close()
			ncs.sessionHandlers[k] = nil
		}
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

// SessionHandler delivers the netconf session handler associated with the specified session id.
func (ncs *TestNCServer) SessionHandler(id uint64) *SessionHandler {
	sh, ok := ncs.sessionHandlers[id]
	if !ok {
		ncs.tctx.Errorf("Failed to get handler for session %d", id)
		ncs.tctx.FailNow()
	}
	return sh
}
