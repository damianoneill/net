package ops

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/damianoneill/net/v2/netconf/testserver"

	"golang.org/x/crypto/ssh"
)

func ExampleSession_GetSubTreeUsingStrings() {

	ts := testserver.NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, err := NewSession(context.Background(), sshConfig, serverAddress)

	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}

	response := ""
	err = s.GetSubtree("<top><sub/></top>", &response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response)

	s.Close()

	// Output: <filter type="subtree"><top><sub/></top></filter>
}

func ExampleSession_GetSubTreeUsingStructs() {

	ts := testserver.NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, err := NewSession(context.Background(), sshConfig, serverAddress)

	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}

	type testStruct struct {
		XMLName xml.Name `xml:"top"`
		Sub     string   `xml:"sub"`
	}

	type testResponse struct {
		XMLName xml.Name `xml:"filter"`
		Type    string   `xml:"type,attr"`
		Select  string   `xml:"select,attr,omitempty"`
		SubTree *testStruct
	}
	response := &testResponse{}

	err = s.GetSubtree(&testStruct{Sub: "DummyValue"}, response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response.Type)
	fmt.Printf("%s\n", response.SubTree.Sub)

	s.Close()

	// Output: subtree
	// DummyValue
}

func ExampleSession_GetXpath() {

	ts := testserver.NewTestNetconfServer(nil)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	serverAddress := fmt.Sprintf("localhost:%d", ts.Port())
	s, err := NewSession(context.Background(), sshConfig, serverAddress)

	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}

	response := ""
	err = s.GetXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, &response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response)

	s.Close()

	// Output: <filter xmlns:tns="urn:tns" type="xpath" select="/tns:element"/>
}
