package testserver_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/damianoneill/net/v2/netconf/client"
	"github.com/damianoneill/net/v2/netconf/common"
	"github.com/damianoneill/net/v2/netconf/testserver"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

const req = `<get>
   <filter type="subtree">
       <physical-ports xmlns="http://www.lumentum.com/lumentum-ote-port" xmlns:loteeth="http://www.lumentum.com/lumentum-ote-port-ethernet">
       </physical-ports>
   </filter>
</get>`

func TestMultipleTestServersWithoutChunkedEncoding(t *testing.T) {

	var svrCount = 10
	var reqCount = 100

	ts := createServersWithoutChunkedEncoding(t, svrCount)
	defer func() {
		for i := 0; i < len(ts); i++ {
			ts[i].Close()
		}
	}()

	ss := createSessions(t, ts)

	wg := &sync.WaitGroup{}
	for i := 0; i < len(ss); i++ {
		wg.Add(1)
		go exSession(t, ss[i], wg, reqCount)
	}

	wg.Wait()

	for i := 0; i < len(ts); i++ {
		assert.Equal(t, reqCount, ts[i].LastHandler().ReqCount())
	}
}

func TestMultipleTestServersWithChunkedEncoding(t *testing.T) {

	var svrCount = 10
	var reqCount = 100

	ts := createServersWithChunkedEncoding(t, svrCount)
	defer func() {
		for i := 0; i < len(ts); i++ {
			ts[i].Close()
		}
	}()

	ss := createSessions(t, ts)

	wg := &sync.WaitGroup{}
	for i := 0; i < len(ss); i++ {
		wg.Add(1)
		go exSession(t, ss[i], wg, reqCount)
	}

	wg.Wait()

	for i := 0; i < len(ts); i++ {
		assert.Equal(t, reqCount, ts[i].LastHandler().ReqCount())
	}
}

func TestMultipleSessions(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)

	ncs := newNCClientSession(t, ts)
	assert.Nil(t, ts.SessionHandler(ncs.ID()).LastReq(), "No requests should have been executed")

	reply, err := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")

	ncs.Close()

	ncs = newNCClientSession(t, ts)
	defer ncs.Close()

	reply, err = ncs.Execute(common.Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")

}

func exSession(t *testing.T, s client.Session, wg *sync.WaitGroup, reqCount int) {
	defer wg.Done()
	defer s.Close()
	for e := 0; e < reqCount; e++ {

		reply, _ := s.Execute(common.Request(req))

		assert.NotNil(t, reply, "Execute failed unexpectedly")

	}
}

func createServersWithoutChunkedEncoding(t *testing.T, count int) []*testserver.TestNCServer {
	ts := make([]*testserver.TestNCServer, count)
	for i := 0; i < count; i++ {
		ts[i] = testserver.NewTestNetconfServer(t).WithCapabilities([]string{
			common.CapBase10,
		})
	}
	return ts
}

func createServersWithChunkedEncoding(t *testing.T, count int) []*testserver.TestNCServer {
	ts := make([]*testserver.TestNCServer, count)
	for i := 0; i < count; i++ {
		ts[i] = testserver.NewTestNetconfServer(t).WithCapabilities([]string{
			common.CapBase10,
			common.CapBase11,
		})
	}
	return ts
}

func createSessions(t *testing.T, ts []*testserver.TestNCServer) []client.Session {
	ss := make([]client.Session, len(ts))
	for i := 0; i < len(ts); i++ {
		s, err := client.NewRPCSession(context.Background(), sshConfig(), fmt.Sprintf("localhost:%d", ts[i].Port()))
		assert.NoError(t, err, "Expecting new session to succeed")
		ss[i] = s
	}
	return ss
}

func sshConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func newNCClientSession(t assert.TestingT, ts *testserver.TestNCServer) client.Session {
	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := client.NewRPCSession(context.Background(), sshConfig, serverAddress)
	assert.NoError(t, err, "Failed to create session")
	return s
}
