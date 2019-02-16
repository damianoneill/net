package netconf

import (
	"context"
	"fmt"
	"sync"
	"testing"

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

	fmt.Println("<<< waited")

	for i := 0; i < len(ts); i++ {
		assert.Equal(t, reqCount, ts[i].ReqCount())
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

	fmt.Println("<<< waited")

	for i := 0; i < len(ts); i++ {
		assert.Equal(t, reqCount, ts[i].ReqCount())
	}
}

func exSession(t *testing.T, s Session, wg *sync.WaitGroup, reqCount int) {
	defer wg.Done()
	defer s.Close()
	for e := 0; e < reqCount; e++ {

		reply, _ := s.Execute(Request(req))

		assert.NotNil(t, reply, "Execute failed unexpectedly")

	}
}

func createServersWithoutChunkedEncoding(t *testing.T, count int) []*TestNCServer {
	ts := make([]*TestNCServer, count)
	for i := 0; i < count; i++ {
		ts[i] = NewTestNetconfServer(t).WithCapabilities([]string{
			CapBase10,
		})
	}
	return ts
}

func createServersWithChunkedEncoding(t *testing.T, count int) []*TestNCServer {
	ts := make([]*TestNCServer, count)
	for i := 0; i < count; i++ {
		ts[i] = NewTestNetconfServer(t).WithCapabilities([]string{
			CapBase10,
			CapBase11,
		})
	}
	return ts
}

func createSessions(t *testing.T, ts []*TestNCServer) []Session {
	ss := make([]Session, len(ts))
	for i := 0; i < len(ts); i++ {
		s, err := NewRPCSession(context.Background(), sshConfig(), fmt.Sprintf("localhost:%d", ts[i].Port()))
		assert.NoError(t, err, "Expecting new session to succeed")
		ss[i] = s
	}
	return ss
}

func sshConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}
