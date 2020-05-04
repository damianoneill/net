package ops

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/damianoneill/net/v2/netconf/testserver"

	"golang.org/x/crypto/ssh"
)

func ExampleOpSession_GetSubtree_usingStrings() {

	ts := testserver.NewTestNetconfServer(nil).WithRequestHandler(testserver.SmartRequesttHandler)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}
	defer s.Close()

	response := ""
	err = s.GetSubtree("<top><sub/></top>", &response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response)

	s.Close()

	// Output: <top><sub attr="avalue"><child1>cvalue</child1><child2/></sub></top>
}

func ExampleOpSession_GetSubtree_usingStructs() {

	ts := testserver.NewTestNetconfServer(nil).WithRequestHandler(testserver.SmartRequesttHandler)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}
	defer s.Close()

	type testSub struct {
		XMLName xml.Name `xml:"sub"`
		Attr    string   `xml:"attr,attr"`
		Child1  string   `xml:"child1"`
		Child2  string   `xml:"child2"`
	}

	type testStruct struct {
		XMLName xml.Name `xml:"top"`
		Sub     *testSub
	}

	response := &testStruct{}

	err = s.GetSubtree(&testStruct{}, response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response.Sub.Attr)
	fmt.Printf("%s\n", response.Sub.Child1)

	s.Close()

	// Output: avalue
	// cvalue
}

func ExampleOpSession_GetXpath() {

	ts := testserver.NewTestNetconfServer(nil).WithRequestHandler(testserver.SmartRequesttHandler)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}
	defer s.Close()

	response := ""
	err = s.GetXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, &response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response)

	s.Close()

	// Output: <top><sub attr="avalue"><child1>cvalue</child1><child2/></sub></top>
}

func ExampleOpSession_GetConfigSubtree() {

	ts := testserver.NewTestNetconfServer(nil).WithRequestHandler(testserver.SmartRequesttHandler)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}
	defer s.Close()

	type testSub struct {
		XMLName xml.Name `xml:"sub"`
		Attr    string   `xml:"attr,attr"`
		Child1  string   `xml:"child1"`
		Child2  string   `xml:"child2"`
	}

	type subCfg struct {
		XMLName xml.Name `xml:"top"`
		Sub     *testSub
	}

	response := &subCfg{}

	err = s.GetConfigSubtree(&subCfg{}, CandidateCfg, response)
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Printf("%s\n", response.Sub.Attr)
	fmt.Printf("%s\n", response.Sub.Child1)

	s.Close()

	// Output: cfgval1
	// cfgval2
}

func ExampleOpSession_GetSchema() {

	ts := testserver.NewTestNetconfServer(nil).WithRequestHandler(testserver.SmartRequesttHandler)

	sshConfig := &ssh.ClientConfig{
		User:            testserver.TestUserName,
		Auth:            []ssh.AuthMethod{ssh.Password(testserver.TestPassword)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	s, err := NewSession(context.Background(), sshConfig, fmt.Sprintf("localhost:%d", ts.Port()))
	if err != nil {
		fmt.Printf("Failed to start session %s\n", err)
		return
	}
	defer s.Close()

	schema, err := s.GetSchema("id", "version", "yang")
	if err != nil {
		fmt.Printf("Failed to execute RPC:%s\n", err)
		return
	}
	fmt.Println(schema)
	// Output: module junos-rpc-vpls {
	//   namespace "http://yang.juniper.net/junos/rpc/vpls";
	//
	//   prefix vpls;
	//
	//// etc…
}
