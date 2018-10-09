package netconf

import (
	"io"
	"log"
	"os"
	"strings"
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

	expectToReplyToRequest(mockt)

	ncs, err := NewSession(mockt, l, l)

	reply, err := ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
}

func TestExecuteAsync(t *testing.T) {

	mockt := &mocks.Transport{}

	l := log.New(os.Stderr, "logger:", log.Lshortfile)

	expectToReadServerHello(mockt)
	_ = expectToSendMessage(mockt)

	expectToReplyToRequest(mockt)

	ncs, err := NewSession(mockt, l, l)

	rch := make(chan *RPCReply)
	err = ncs.ExecuteAsync(Request(`<get-config><source><running/></source></get-config>`), rch)
	assert.NoError(t, err, "Not expecting exec to fail")

	reply := <-rch
	assert.NotNil(t, reply, "Reply should be non-nil")
	assert.Equal(t, `<data><response/></data>`, reply.Data, "Reply should contain response data")
}

func expectToReplyToRequest(mockt *mocks.Transport) {
	ch := make(chan bool)
	mockt.On("Read", mock.Anything).Return(func(buf []byte) int {
		<-ch
		i := rpcReply()
		copy(buf, i)
		return len(i)
	}, nil)

	mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		if strings.Contains(string(buf), "]]>]]>") {
			ch <- true
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

func expectToSendMessage(mockt *mocks.Transport) *[]byte {
	var msg []byte
	mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		msg = append(msg, buf...)
		return len(buf)
	}, nil).Twice()
	return &msg
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

func rpcReply() []byte {
	return []byte(` <rpc-reply message-id="101"` +
		`xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"` +
		`xmlns:ex="http://example.net/content/1.0"` +
		`ex:user-id="fred">` +
		`<data>` +
		`<response/>` +
		`</data>` +
		`</rpc-reply>` +
		`]]>]]>`)
}

// Simple real NE access test

// func TestNewSession(t *testing.T) {

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

// 	reply, err := ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 	assert.NoError(t, err, "Not expecting exec to fail")
// 	assert.NotNil(t, reply, "Reply should be non-nil")

// 	reply, err = ncs.Execute(Request(`<get-config><source><running/></source></get-config>`))
// 	assert.NoError(t, err, "Not expecting exec to fail")
// 	assert.NotNil(t, reply, "Reply should be non-nil")
// 	assert.Zero(t, len(reply.Errors), "Not expecting server errors")
// }
