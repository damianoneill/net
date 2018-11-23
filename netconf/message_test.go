package netconf

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNewSessionWithChunkedEncoding(t *testing.T) {

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	sh := ts.SessionHandler(ncs.ID())

	assert.NotNil(t, ncs, "Session should be non-nil")
	assert.Equal(t, uint64(1), ncs.ID(), "Session id not defined correctly")

	sh.WaitStart()
	assert.NotNil(t, sh.ClientHello, "Should have sent hello")
	assert.Equal(t, sh.ClientHello.Capabilities, DefaultCapabilities, "Did not send expected server capabilities")

	ncs.Close()
}

func TestExecute(t *testing.T) {

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	defer ncs.Close()

	sh := ts.SessionHandler(ncs.ID())
	assert.Nil(t, sh.LastReq(), "No requests should have been executed")

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
	assert.Equal(t, 1, sh.ReqCount(), "Expected request count to be 1")
	assert.Equal(t, "get", sh.LastReq().XMLName.Local, "Expected GET request")
	assert.Equal(t, "<response/>", sh.LastReq().Body, "Expected request body")
}

func TestExecuteWithFailingRequest(t *testing.T) {

	ncs := newNCClientSession(t, NewTestNetconfServer(t).WithRequestHandler(FailingRequestHandler))
	defer ncs.Close()

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.Error(t, err, "Expecting exec to fail")
	assert.Equal(t, "netconf rpc [error] 'oops'", err.Error(), "Expected error")
	assert.NotNil(t, reply, "Reply should be non-nil")
}

func TestExecuteFailure(t *testing.T) {

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	defer ncs.Close()

	// Close the transport - to force error when we try to use it.
	ts.Close()
	time.Sleep(time.Millisecond * time.Duration(250))

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.Error(t, err, "Expecting exec to fail")
	assert.Equal(t, "EOF", err.Error(), "Expected EOF error")
	assert.Nil(t, reply, "Reply should be nil")
}

func TestNewSessionWithEndOfMessageEncoding(t *testing.T) {

	ncs := newNCClientSession(t, NewTestNetconfServer(t).WithCapabilities([]string{CapBase10}))

	assert.False(t, peerSupportsChunkedFraming(ncs.(*sesImpl).hello.Capabilities), "Server not expected to support chunked framing")

	reply, _ := ncs.Execute(Request(`<get><response/></get>`))
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")

	ncs.Close()
}

func TestExecuteAsync(t *testing.T) {

	ncs := newNCClientSession(t, NewTestNetconfServer(t))
	defer ncs.Close()

	rch1 := make(chan *RPCReply)
	rch2 := make(chan *RPCReply)
	rch3 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)
	ncs.ExecuteAsync(Request(`<get><test2/></get>`), rch2)
	ncs.ExecuteAsync(Request(`<get><test3/></get>`), rch3)

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

	ncs := newNCClientSession(t, NewTestNetconfServer(t).WithRequestHandler(CloseRequestHandler))
	defer ncs.Close()

	rch1 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)

	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestExecuteAsyncInterrupted(t *testing.T) {

	ncs := newNCClientSession(t, NewTestNetconfServer(t).WithRequestHandler(IgnoreRequestHandler))
	defer ncs.Close()

	rch1 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)

	time.AfterFunc(time.Second*time.Duration(2), func() { ncs.Close() })
	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestSubscribe(t *testing.T) {

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)
	sh := ts.SessionHandler(ncs.ID())

	nch := make(chan *Notification)

	var wg sync.WaitGroup
	wg.Add(1)
	// Capture notification that we expect.
	var result *Notification
	go func() {
		result = <-nch
		wg.Done()
	}()

	reply, _ := ncs.Subscribe(Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nch)
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

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)

	var wg sync.WaitGroup
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			request := fmt.Sprintf(`<get><Id_%d/></get>`, id)
			replybody := fmt.Sprintf(`<data><Id_%d/></data>`, id)
			for i := 0; i < 100; i++ {
				reply, err := ncs.Execute(Request(request))
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

	ts := NewTestNetconfServer(t)
	ncs := newNCClientSession(t, ts)

	var wg sync.WaitGroup
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			request := fmt.Sprintf(`<get><Id_%d/></get>`, id)
			replybody := fmt.Sprintf(`<data><Id_%d/></data>`, id)
			rchan := make(chan *RPCReply)
			for i := 0; i < 100; i++ {
				err := ncs.ExecuteAsync(Request(request), rchan)
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

	ncs := newNCClientSession(b, NewTestNetconfServer(b))

	for n := 0; n < b.N; n++ {
		ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	}
}

func BenchmarkTemplateParallel(b *testing.B) {

	ncs := newNCClientSession(b, NewTestNetconfServer(b))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
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

func newNCClientSession(t assert.TestingT, ts *TestNCServer) Session {
	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewRPCSession(context.Background(), sshConfig, serverAddress)
	assert.NoError(t, err, "Failed to create session")
	return s
}

// Simple real NE access tests

// func TestRealNewSession(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "XXxxx",
// 		Auth:            []ssh.AuthMethod{ssh.Password("XXxxxxxxx")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	ctx := WithClientTrace(context.Background(), DefaultLoggingHooks)
// 	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 	assert.NoError(t, err, "Not expecting new transport to fail")
// 	defer tr.Close()

// 	ncs, err := NewSession(ctx, tr, defaultConfig)
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, ncs, "Session should be non-nil")

// 	var wg sync.WaitGroup
// 	for n := 0; n < 1; n++ {
// 		wg.Add(1)
// 		go func(z int) {
// 			defer wg.Done()
// 			for c := 0; c < 1; c++ {
// 				reply, err := ncs.Execute(Request(`<get/>`))
// 				assert.NoError(t, err, "Not expecting exec to fail")
// 				assert.NotNil(t, reply, "Reply should be non-nil")
// 			}
// 		}(n)
// 	}
// 	wg.Wait()
// }

// func TestRealSubscription(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "XXxxxx",
// 		Auth:            []ssh.AuthMethod{ssh.Password("XXxxxxxxxx")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	ctx := WithClientTrace(context.Background(), DefaultLoggingHooks)

// 	tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 	assert.NoError(t, err, "Not expecting new transport to fail")
// 	defer tr.Close()

// 	ncs, err := NewSession(ctx, tr, defaultConfig)
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, ncs, "Session should be non-nil")

// 	nchan := make(chan *Notification)
// 	reply, err := ncs.Subscribe(Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nchan)
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// 	assert.NoError(t, err, "Not expecting exec to fail")

// 	time.AfterFunc(time.Second*2, func() {
// 		tr, err := NewSSHTransport(ctx, sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 		assert.NoError(t, err, "Not expecting new transport to fail")
// 		ns, _ := NewSession(ctx, tr, defaultConfig)
// 		ns.Close()
// 	})

// 	n := <-nchan

// 	assert.NotNil(t, n, "Reply should be non-nil")
// 	fmt.Printf("%v\n", n)
// }
