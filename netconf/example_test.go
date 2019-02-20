package netconf

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

func ExampleSession_Execute() {

	ts := NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, err := NewRPCSession(context.Background(), sshConfig, serverAddress)

	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}

	r, err := s.Execute(Request("<get><expectResponse/></get>"))
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", r.Data)

	s.Close()

	// Output: <data><expectResponse/></data>
}

func ExampleSession_ExecuteAsync() {

	ts := NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, _ := NewRPCSession(context.Background(), sshConfig, serverAddress)

	rch1 := make(chan *RPCReply)
	_ = s.ExecuteAsync(Request("<get><expectResponse1/></get>"), rch1)
	rch2 := make(chan *RPCReply)
	_ = s.ExecuteAsync(Request("<get><expectResponse2/></get>"), rch2)

	r := <-rch2
	fmt.Printf("%s\n", r.Data)
	r = <-rch1
	fmt.Printf("%s\n", r.Data)

	s.Close()

	// Output: <data><expectResponse2/></data>
	// <data><expectResponse1/></data>
}

func ExampleSession_Subscribe() {

	ts := NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, _ := NewRPCSession(context.Background(), sshConfig, serverAddress)
	sh := ts.SessionHandler(s.ID())

	nch := make(chan *Notification)
	_, _ = s.Subscribe(Request(`<ncEvent:create-subscription xmlns:ncEvent="urn:ietf:params:xml:ns:netconf:notification:1.0"></ncEvent:create-subscription>`), nch)

	// Get test server to send a notification in 500ms.
	time.AfterFunc(time.Millisecond*time.Duration(500), func() {
		sh.SendNotification(`<typeA xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-notifications"><name>XXX</name></typeA>`)
	})

	n := <-nch
	fmt.Printf("Type:%s Event:%s\n", n.XMLName.Local, n.Event)

	s.Close()

	// Output: Type:typeA Event:<typeA xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-notifications"><name>XXX</name></typeA>
}
