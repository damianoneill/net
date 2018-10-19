package netconf

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/damianoneill/net/testutil"
	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNewSessionWithChunkedEncoding(t *testing.T) {

	server, tr := testNetconfServer(t)
	ncs, err := NewSession(context.Background(), tr, defaultConfig)

	assert.NoError(t, err, "Not expecting new session to fail")
	assert.NotNil(t, ncs, "Session should be non-nil")
	assert.Equal(t, 4, ncs.ID(), "Session id not defined correctly")

	server.waitStart()
	assert.NotNil(t, server.clientHello, "Should have sent hello")
	assert.Equal(t, server.clientHello.Capabilities, DefaultCapabilities, "Did not send expected server capabilities")

	ncs.Close()
}

func TestExecute(t *testing.T) {

	_, tr := testNetconfServer(t)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)
	defer ncs.Close()

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
}

func TestExecuteFailure(t *testing.T) {

	server, tr := testNetconfServer(t)
	server.withRequestHandler(FailingRequestHandler)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)
	defer ncs.Close()

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.Error(t, err, "Expecting exec to fail")
	assert.Equal(t, "netconf rpc [error] 'oops'", err.Error(), "Expected error")
	assert.NotNil(t, reply, "Reply should be non-nil")
}

func TestNewSessionWithEndOfMessageEncoding(t *testing.T) {

	ncServer := newHandler(t, 4).withCapabilities([]string{CapBase10})
	tr := getSSHTransport(t, ncServer)

	ncs, err := NewSession(WithClientTrace(context.Background(), DiagnosticLoggingHooks), tr, defaultConfig)

	assert.NoError(t, err, "Not expecting new session to fail")
	assert.NotNil(t, ncs, "Session should be non-nil")

	assert.False(t, peerSupportsChunkedFraming(ncs.(*sesImpl).hello.Capabilities), "Server not expected to support chunked framing")

	reply, _ := ncs.Execute(Request(`<get><response/></get>`))
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")

	ncs.Close()
}

func TestExecuteAsync(t *testing.T) {

	_, tr := testNetconfServer(t)
	ncs, _ := NewSession(WithClientTrace(context.Background(), DiagnosticLoggingHooks), tr, defaultConfig)

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

	server, tr := testNetconfServer(t)
	server.withRequestHandler(CloseRequestHandler)
	ncs, _ := NewSession(WithClientTrace(context.Background(), DiagnosticLoggingHooks), tr, defaultConfig)

	rch1 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)

	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestExecuteAsyncInterrupted(t *testing.T) {

	server, tr := testNetconfServer(t)
	server.withRequestHandler(IgnoreRequestHandler)
	ncs, _ := NewSession(WithClientTrace(context.Background(), DiagnosticLoggingHooks), tr, defaultConfig)

	rch1 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)

	time.AfterFunc(time.Second*time.Duration(2), func() { ncs.Close() })
	reply := <-rch1
	assert.Nil(t, reply, "Reply should be nil")
}

func TestSubscribe(t *testing.T) {

	server, tr := testNetconfServer(t)
	ncs, _ := NewSession(WithClientTrace(context.Background(), DiagnosticLoggingHooks), tr, defaultConfig)

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

	server.sendNotification(notificationEvent())

	// Wait for notification.
	wg.Wait()
	assert.NotNil(t, result, "Expected notification")
	assert.Equal(t, "netconf-session-start", result.XMLName.Local, "Unexpected event type")
	assert.Equal(t, "urn:ietf:params:xml:ns:yang:ietf-netconf-notifications", result.XMLName.Space, "Unexpected event NS")
	assert.NotNil(t, result.EventTime, "Unexpected nil event time")
	assert.Equal(t, notificationEvent(), result.Event, "Unexpected event XML")

	// Get server to send notifications, wait a while for them to arrive and confirm they've been dropped.
	server.sendNotification(notificationEvent())
	server.sendNotification(notificationEvent())
	time.Sleep(time.Millisecond * time.Duration(500))
	assert.Equal(t, 2, ncs.(*sesImpl).notificationDropCount, "Expected notification to have been dropped")

	server.close()
	result = <-nch
	assert.Nil(t, result, "No more notifications expected")
}

func TestConcurrentExecute(t *testing.T) {

	server, tr := testNetconfServer(t)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)

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
	assert.Equal(t, 1000, server.reqCount, "Unexpected request count")
}

func TestConcurrentExecuteAsync(t *testing.T) {

	server, tr := testNetconfServer(t)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)

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

	assert.Equal(t, 1000, server.reqCount, "Unexpected request count")
}

func BenchmarkExecute(b *testing.B) {

	_, tr := testNetconfServer(b)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)

	for n := 0; n < b.N; n++ {
		ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	}
}

func BenchmarkTemplateParallel(b *testing.B) {

	_, tr := testNetconfServer(b)
	ncs, _ := NewSession(context.Background(), tr, defaultConfig)

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

func testNetconfServer(t assert.TestingT) (*netconfSessionHandler, Transport) {
	server := newHandler(t, 4)
	tr := getSSHTransport(t, server)
	return server, tr
}

func getSSHTransport(t assert.TestingT, handler *netconfSessionHandler) Transport {

	ts := testutil.NewSSHServerHandler(t, "testUser", "testPassword", handler)

	sshConfig := &ssh.ClientConfig{
		User:            "testUser",
		Auth:            []ssh.AuthMethod{ssh.Password("testPassword")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	tr, err := NewSSHTransport(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()), "netconf")

	assert.NoError(t, err, "Failed to connect to server")
	return tr
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
