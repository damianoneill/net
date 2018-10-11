package netconf

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	mocks "github.com/damianoneill/net/netconf/mocks"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	mockt.On("Read", mock.Anything).Return(0, io.EOF)

	hellobuf := expectToSendMessage(mockt)

	ncs, err := NewSession(mockt, l, l)

	assert.NoError(t, err, "Not expecting new session to fail")
	assert.NotNil(t, ncs, "Session should be non-nil")
	assert.Contains(t, string(*hellobuf), "<hello ")
	assert.Contains(t, string(*hellobuf), "]]>]]>")

	mockt.On("Close").Return(nil)
	ncs.Close()
}

func TestExecute(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, err := NewSession(mockt, l, l)

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
}

func TestExecuteAsync(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, _ := NewSession(mockt, l, l)

	rch1 := make(chan *RPCReply)
	rch2 := make(chan *RPCReply)
	rch3 := make(chan *RPCReply)
	ncs.ExecuteAsync(Request(`<get><test1/></get>`), rch1)
	ncs.ExecuteAsync(Request(`<get><test2/></get>`), rch2)
	ncs.ExecuteAsync(Request(`<get><test3/></get>`), rch3)

	reply := <-rch3
	assert.Equal(t, `<data><test3/></data>`, reply.Data, "Reply should contain response data")
	reply = <-rch2
	assert.Equal(t, `<data><test2/></data>`, reply.Data, "Reply should contain response data")
	reply = <-rch1
	assert.Equal(t, `<data><test1/></data>`, reply.Data, "Reply should contain response data")
}

func TestSubscribe(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 1)
	expectToReadOneNotification(mockt, notificationMessage())

	ncs, _ := NewSession(mockt, l, l)

	nch := make(chan *Notification)
	reply, _ := ncs.Subscribe(Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nch)

	assert.NotNil(t, reply.Data, "create-subscription failed")

	result := <-nch
	assert.NotNil(t, result, "Expected notification")
	assert.Equal(t, "netconf-session-start", result.XMLName.Local, "Unexpected event type")
	assert.Equal(t, "urn:ietf:params:xml:ns:yang:ietf-netconf-notifications", result.XMLName.Space, "Unexpected event NS")
	assert.Equal(t, "2018-10-10T09:23:07Z", result.EventTime, "Unexpected event time")
	assert.Equal(t, notificationEvent(), result.Event, "Unexpected event XML")

	result = <-nch
	assert.Nil(t, result, "No more notifications expected")

}

func TestConcurrentExecute(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, _ := NewSession(mockt, l, l)

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
}

func TestConcurrentExecuteAsync(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, _ := NewSession(mockt, l, l)

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

				assert.Equal(t, replybody, reply.Data, "Reply should contain response data")
			}
		}(r)
	}
	wg.Wait()
}

func BenchmarkExecute(b *testing.B) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, _ := NewSession(mockt, l, l)

	for n := 0; n < b.N; n++ {
		ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	}
}

func BenchmarkTemplateParallel(b *testing.B) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToNRequests(mockt, 0)

	ncs, _ := NewSession(mockt, l, l)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
		}
	})
}

func expectToReplyToRequest(mockt *mocks.Transport) {
	expectToReplyToNRequests(mockt, 1)
}

func expectToReplyToNRequests(mockt *mocks.Transport, n int) {
	ch := make(chan string)

	call := mockt.On("Read", mock.Anything).Return(func(buf []byte) int {
		body := <-ch
		i := rpcReply(body)
		copy(buf, i)
		return len(i)
	}, nil)
	if n > 0 {
		call.Times(n)
	}

	var body string
	call = mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		if strings.Contains(string(buf), "]]>]]>") {
			ch <- body
		} else {
			// Remember body so that we can build it into the reply
			body = extractRequestBody(buf)
		}
		return len(buf)
	}, nil)
}

func expectToReadServerHello(mockt *mocks.Transport) {
	mockt.On("Read", mock.Anything).Return(func(buf []byte) int {
		i := serverHello()
		copy(buf, i)
		return len(i)
	}, nil).Once()
}

func expectToReadOneNotification(mockt *mocks.Transport, n string) {
	mockt.On("Read", mock.Anything).Return(func(buf []byte) int {
		i := []byte(n)
		copy(buf, i)
		return len(i)
	}, nil).Once()
	mockt.On("Read", mock.Anything).Return(0, io.EOF)
}

func expectToSendMessage(mockt *mocks.Transport) *[]byte {
	var msg []byte
	mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		msg = append(msg, buf...)
		return len(buf)
	}, nil).Twice()
	return &msg
}

func extractRequestBody(buf []byte) string {
	re := regexp.MustCompile("<get>(.*)</get>")
	matches := re.FindStringSubmatch(string(buf))
	if matches != nil && len(matches) > 0 {
		return matches[1]
	}
	return ""
}

func serverHello() []byte {
	return []byte(`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">` +
		`<capabilities>` +
		`<capability>` +
		`urn:ietf:params:netconf:base:1.0` +
		`</capability>` +
		`<capability>` +
		`urn:ietf:params:netconf:capability:startup:1.0` +
		`</capability>` +
		`<capability>` +
		`http://example.net/router/2.3/myfeature` +
		`</capability>` +
		`</capabilities>` +
		`<session-id>4</session-id>` +
		`</hello>` +
		`]]>]]>`)
}

func rpcReply(body string) []byte {
	return []byte(` <rpc-reply message-id="101"` +
		`xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"` +
		`xmlns:ex="http://example.net/content/1.0"` +
		`ex:user-id="fred">` +
		`<data>` +
		body +
		`</data>` +
		`</rpc-reply>` +
		`]]>]]>`)
}

func notificationMessage() string {
	return `<notification xmlns="urn:ietf:params:xml:ns:netconf:notification:1.0">` +
		`<eventTime>2018-10-10T09:23:07Z</eventTime>` +
		notificationEvent() +
		`</notification>` +
		`]]>]]>`
}

func notificationEvent() string {
	return `<netconf-session-start xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-notifications">` +
		`<username>WRuser</username>` +
		`<session-id>321</session-id>` +
		`<source-host>172.26.136.66</source-host>` +
		`</netconf-session-start>`
}

// Simple real NE access tests

// func TestRealNewSession(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "WRuser",
// 		Auth:            []ssh.AuthMethod{ssh.Password("WRuser123")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 	assert.NoError(t, err, "Not expecting new transport to fail")
// 	defer tr.Close()

// 	l := log.New(os.Stderr, "logger:", log.Lshortfile)
// 	ncs, err := NewSession(tr, l, l)
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, ncs, "Session should be non-nil")

// 	var wg sync.WaitGroup
// 	for n := 0; n < 1; n++ {
// 		wg.Add(1)
// 		go func(z int) {
// 			defer wg.Done()
// 			for c := 0; c < 1; c++ {
// 				reply, err := ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 				assert.NoError(t, err, "Not expecting exec to fail")
// 				assert.NotNil(t, reply, "Reply should be non-nil")
// 			}
// 		}(n)
// 	}
// 	wg.Wait()
// }

// func TestRealSubscription(t *testing.T) {

// 	sshConfig := &ssh.ClientConfig{
// 		User:            "WRuser",
// 		Auth:            []ssh.AuthMethod{ssh.Password("WRuser123")},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}

// 	tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 	assert.NoError(t, err, "Not expecting new transport to fail")
// 	defer tr.Close()

// 	l := log.New(os.Stderr, "logger:", log.Lshortfile)
// 	ncs, err := NewSession(tr, l, l)
// 	assert.NoError(t, err, "Not expecting new session to fail")
// 	assert.NotNil(t, ncs, "Session should be non-nil")

// 	nchan := make(chan *Notification)
// 	reply, err := ncs.Subscribe(Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nchan)
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// 	assert.NoError(t, err, "Not expecting exec to fail")

// 	time.AfterFunc(time.Second*2, func() {
// 		tr, err := NewSSHTransport(sshConfig, fmt.Sprintf("172.26.138.57:%d", 830), "netconf")
// 		assert.NoError(t, err, "Not expecting new transport to fail")
// 		ns, _ := NewSession(tr, l, l)
// 		ns.Close()
// 	})

// 	n := <-nchan

// 	assert.NotNil(t, n, "Reply should be non-nil")
// 	fmt.Printf("%v\n", n)

// }
