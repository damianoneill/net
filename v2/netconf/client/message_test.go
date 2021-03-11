package client

import (
	"context"
	"encoding/xml"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/damianoneill/net/v2/netconf/common"
	"github.com/damianoneill/net/v2/netconf/testserver"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNewSessionWithChunkedEncoding(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	sh := ts.SessionHandler(ncs.ID())

	assert.NotNil(t, ncs, "Session should be non-nil")
	assert.Equal(t, uint64(1), ncs.ID(), "Session id not defined correctly")
	assert.Contains(t, ncs.ServerCapabilities(), common.CapBase10, "Failed to retrieve expected capabilities")

	sh.WaitStart()
	assert.NotNil(t, sh.ClientHello, "Should have sent hello")
	assert.Equal(t, sh.ClientHello.Capabilities, common.DefaultCapabilities, "Did not send expected server capabilities")

	ncs.Close()
}

func TestExecute(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	defer ncs.Close()

	sh := ts.SessionHandler(ncs.ID())
	assert.Nil(t, sh.LastReq(), "No requests should have been executed")

	reply, err := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
	assert.Equal(t, 1, sh.ReqCount(), "Expected request count to be 1")
	assert.Equal(t, "get", sh.LastReq().XMLName.Local, "Expected GET request")
	assert.Equal(t, "<response/>", sh.LastReq().Body, "Expected request body")
}

func TestExecuteWithStruct(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	defer ncs.Close()

	sh := ts.SessionHandler(ncs.ID())
	assert.Nil(t, sh.LastReq(), "No requests should have been executed")

	type req struct {
		XMLName xml.Name `xml:"get"`
		Body    string   `xml:"body"`
	}

	reply, err := ncs.Execute(common.Request(&req{}))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><body></body></data>`, reply.Data, "Reply should contain response data")
	assert.Equal(t, 1, sh.ReqCount(), "Expected request count to be 1")
	assert.Equal(t, "get", sh.LastReq().XMLName.Local, "Expected GET request")
	assert.Equal(t, "<body></body>", sh.LastReq().Body, "Expected request body")
}

func TestExecuteWithFailingRequest(t *testing.T) {

	ncs := newNCClientSession(t, testserver.NewTestNetconfServer(t).WithRequestHandler(testserver.FailingRequestHandler))
	defer ncs.Close()

	reply, err := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.Error(t, err, "Expecting exec to fail")
	assert.Equal(t, "netconf rpc [error] 'oops'", err.Error(), "Expected error")
	assert.NotNil(t, reply, "Reply should be non-nil")
}

func TestExecuteFailure(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	defer ncs.Close()

	// Close the transport - to force error when we try to use it.
	ts.Close()
	time.Sleep(time.Millisecond * time.Duration(250))

	reply, err := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.Error(t, err, "Expecting exec to fail")
	assert.Equal(t, "EOF", err.Error(), "Expected EOF error")
	assert.Nil(t, reply, "Reply should be nil")
}

func TestNewSessionWithEndOfMessageEncoding(t *testing.T) {

	ncs := newNCClientSession(t, testserver.NewTestNetconfServer(t).WithCapabilities([]string{common.CapBase10}))

	assert.False(t, common.PeerSupportsChunkedFraming(ncs.(*sesImpl).hello.Capabilities), "Server not expected to support chunked framing")

	reply, _ := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")

	ncs.Close()
}

func TestNewSessionWithNoChunkedCodec(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSessionWithConfig(t, ts, &Config{DisableChunkedCodec: true})
	defer ncs.Close()

	sh := ts.SessionHandler(ncs.ID())
	assert.Nil(t, sh.LastReq(), "No requests should have been executed")

	reply, err := ncs.Execute(common.Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
	assert.Equal(t, 1, sh.ReqCount(), "Expected request count to be 1")
	assert.Equal(t, "get", sh.LastReq().XMLName.Local, "Expected GET request")
	assert.Equal(t, "<response/>", sh.LastReq().Body, "Expected request body")
}

func TestExecuteAsync(t *testing.T) {

	ncs := newNCClientSession(t, testserver.NewTestNetconfServer(t))
	defer ncs.Close()

	rch1 := make(chan *common.RPCReply)
	rch2 := make(chan *common.RPCReply)
	rch3 := make(chan *common.RPCReply)
	_ = ncs.ExecuteAsync(common.Request(`<get><test1/></get>`), rch1)
	_ = ncs.ExecuteAsync(common.Request(`<get><test2/></get>`), rch2)
	_ = ncs.ExecuteAsync(common.Request(`<get><test3/></get>`), rch3)

	reply := <-rch3
	assert.NotNil(t, reply, "Reply should not be nil")
	assert.Equal(t, `<data><test3/></data>`, reply.Data, "Reply should contain response data")
	reply = <-rch2
	assert.NotNil(t, reply, "Reply should not be nil")
	assert.Equal(t, `<data><test2/></data>`, reply.Data, "Reply should contain response data")
	reply = <-rch1
	assert.NotNil(t, reply, "Reply should not be nil")
	assert.Equal(t, `<data><test1/></data>`, reply.Data, "Reply should contain response data")
}

func TestExecuteAsyncUnfulfilled(t *testing.T) {

	ncs := newNCClientSession(t, testserver.NewTestNetconfServer(t).WithRequestHandler(testserver.CloseRequestHandler))
	defer ncs.Close()

	rch1 := make(chan *common.RPCReply)
	_ = ncs.ExecuteAsync(common.Request(`<get><test1/></get>`), rch1)

	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestExecuteAsyncInterrupted(t *testing.T) {

	ncs := newNCClientSession(t, testserver.NewTestNetconfServer(t).WithRequestHandler(testserver.IgnoreRequestHandler))
	defer ncs.Close()

	rch1 := make(chan *common.RPCReply)
	_ = ncs.ExecuteAsync(common.Request(`<get><test1/></get>`), rch1)

	time.AfterFunc(time.Second*time.Duration(2), func() { ncs.Close() })
	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestSubscribe(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	sh := ts.SessionHandler(ncs.ID())

	nch := make(chan *common.Notification)

	var wg sync.WaitGroup
	wg.Add(1)
	// Capture notification that we expect.
	var result *common.Notification
	go func() {
		result = <-nch
		wg.Done()
	}()

	reply, _ := ncs.Subscribe(common.Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nch)
	assert.NotNil(t, reply, "create-subscription failed")
	assert.NotNil(t, reply.Data, "create-subscription failed")

	sh.SendNotification(notificationEvent())

	// Wait for notification.
	wg.Wait()
	assert.NotNil(t, result, "Expected notification")
	assert.Equal(t, "netconf-session-start", result.XMLName.Local, "Unexpected event type")
	assert.Equal(t, "urn:ietf:params:xml:ns:yang:ietf-netconf-notifications", result.XMLName.Space, "Unexpected event NS")
	assert.NotNil(t, result.EventTime, "Unexpected nil event time")
	assert.Equal(t, notificationEvent(), result.Event, "Unexpected event XML")

	// Get server to send notifications, wait a while for them to arrive and confirm they've been dropped.
	sh.SendNotification(notificationEvent())
	sh.SendNotification(notificationEvent())
	time.Sleep(time.Millisecond * time.Duration(500))
	assert.Equal(t, uint64(2), atomic.LoadUint64(&(ncs.(*sesImpl).notificationDropCount)), "Expected notification to have been dropped")

	ts.Close()
	result = <-nch
	assert.Nil(t, result, "No more notifications expected")
}

func TestConcurrentExecute(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)

	var wg sync.WaitGroup
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			request := fmt.Sprintf(`<get><Id_%d/></get>`, id)
			replybody := fmt.Sprintf(`<data><Id_%d/></data>`, id)
			for i := 0; i < 100; i++ {
				reply, err := ncs.Execute(common.Request(request))
				assert.NoError(t, err, "Not expecting exec to fail")
				assert.Equal(t, replybody, reply.Data, "Reply should contain response data")
			}

		}(r)
	}
	wg.Wait()
	sh := ts.SessionHandler(ncs.ID())
	assert.Equal(t, 1000, sh.ReqCount(), "Unexpected request count")
}

func TestConcurrentExecuteAsync(t *testing.T) {

	ts := testserver.NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)

	var wg sync.WaitGroup
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			request := fmt.Sprintf(`<get><Id_%d/></get>`, id)
			replybody := fmt.Sprintf(`<data><Id_%d/></data>`, id)
			rchan := make(chan *common.RPCReply)
			for i := 0; i < 100; i++ {
				err := ncs.ExecuteAsync(common.Request(request), rchan)
				assert.NoError(t, err, "Not expecting exec to fail")
				reply := <-rchan

				assert.NotNil(t, reply, "Reply should not be nil")
				assert.Equal(t, replybody, reply.Data, "Reply should contain response data")
			}
		}(r)
	}
	wg.Wait()
	sh := ts.SessionHandler(ncs.ID())
	assert.Equal(t, 1000, sh.ReqCount(), "Unexpected request count")
}

func BenchmarkExecute(b *testing.B) {

	ncs := newNCClientSession(b, testserver.NewTestNetconfServer(b))

	for n := 0; n < b.N; n++ {
		_, _ = ncs.Execute(common.Request(`<get-config><source><running/></source></get-config>`))
	}
}

func BenchmarkTemplateParallel(b *testing.B) {

	ncs := newNCClientSession(b, testserver.NewTestNetconfServer(b))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ncs.Execute(common.Request(`<get-config><source><running/></source></get-config>`))
		}
	})
}

func notificationEvent() string {
	return `<netconf-session-start xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-notifications">` +
		`<username>XXxxxx</username>` +
		`<session-id>321</session-id>` +
		`<source-host>172.26.136.66</source-host>` +
		`</netconf-session-start>`
}

func newNCClientSession(t assert.TestingT, ts *testserver.TestNCServer) Session {
	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewRPCSession(context.Background(), sshConfig, serverAddress)
	assert.NoError(t, err, "Failed to create session")
	return s
}

func newNCClientSessionWithConfig(t assert.TestingT, ts *testserver.TestNCServer, cfg *Config) Session {
	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewRPCSessionWithConfig(context.Background(), sshConfig, serverAddress, cfg)
	assert.NoError(t, err, "Failed to create session")
	return s
}

// Simple real NE access tests

//func TestRealNewSession(t *testing.T) {
//
//	sshConfig := &ssh.ClientConfig{
//		User:            "regress",
//		Auth:            []ssh.AuthMethod{ssh.Password("MaRtInI")},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	}
//
//	ctx := WithClientTrace(context.Background(), DefaultLoggingHooks)
//	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("10.228.63.5:%d", 830), "netconf")
//	assert.NoError(t, err, "Not expecting new transport to fail")
//	defer tr.Close()
//
//	ncs, err := NewSession(ctx, tr, DefaultConfig)
//	assert.NoError(t, err, "Not expecting new session to fail")
//	assert.NotNil(t, ncs, "Session should be non-nil")
//
//	var wg sync.WaitGroup
//	for n := 0; n < 1; n++ {
//		wg.Add(1)
//		go func(z int) {
//			defer wg.Done()
//			for c := 0; c < 1; c++ {
//				reply, err := ncs.Execute(`<get/>`)
//				assert.NoError(t, err, "Not expecting exec to fail")
//				assert.NotNil(t, reply, "Reply should be non-nil")
//			}
//		}(n)
//	}
//	wg.Wait()
//}
//
//func TestRealSubscription(t *testing.T) {
//
//	sshConfig := &ssh.ClientConfig{
//		User:            "XXxxxx",
//		Auth:            []ssh.AuthMethod{ssh.Password("XXxxxxxxxx")},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	}
//
//	ctx := WithClientTrace(context.Background(), DefaultLoggingHooks)
//
//	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
//	assert.NoError(t, err, "Not expecting new transport to fail")
//	defer tr.Close()
//
//	ncs, err := NewSession(ctx, tr, DefaultConfig)
//	assert.NoError(t, err, "Not expecting new session to fail")
//	assert.NotNil(t, ncs, "Session should be non-nil")
//
//	nchan := make(chan *common.Notification)
//	reply, err := ncs.Subscribe(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`, nchan)
//	assert.NotNil(t, reply, "Reply should be non-nil")
//	assert.NoError(t, err, "Not expecting exec to fail")
//
//	time.AfterFunc(time.Second*2, func() {
//		tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
//		assert.NoError(t, err, "Not expecting new transport to fail")
//		ns, _ := NewSession(ctx, tr, DefaultConfig)
//		ns.Close()
//	})
//
//	n := <-nchan
//
//	assert.NotNil(t, n, "Reply should be non-nil")
//	fmt.Printf("%v\n", n)
//}
