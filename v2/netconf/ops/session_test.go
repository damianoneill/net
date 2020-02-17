package ops

import (
	"encoding/xml"
	"errors"
	"testing"

	"github.com/damianoneill/net/v2/netconf/common"

	"github.com/damianoneill/net/v2/netconf/mocks"

	assert "github.com/stretchr/testify/require"
)

func TestGetSubtreeToString(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	defer ncs.Close()
	mcli.On("Execute", createGetSubtreeRequest(`<subtree-element/>`)).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)
	mcli.On("Close")

	var result string
	err := ncs.GetSubtree(`<subtree-element/>`, &result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `<element attr1="ABC"/>`, result, "Reply should contain response data")
}

func TestGetSubtreeToStruct(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetSubtreeRequest(`<subtree-element/>`)).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result = &Element{}
	err := ncs.GetSubtree(`<subtree-element/>`, result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `ABC`, result.Attr1, "Reply should contain response data")
}

func TestGetSubtreeExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetSubtreeRequest(`<subtree-element/>`)).Return(nil, errors.New("failed"))

	var result string
	err := ncs.GetSubtree(`<subtree-element/>`, &result)
	assert.Error(t, err, "expecting call to fail")
}

func TestGetXpathToString(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetXpathRequest(`/tns:element`, []Namespace{{"tns", "urn:tns"}})).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result string
	err := ncs.GetXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, &result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `<element attr1="ABC"/>`, result, "Reply should contain response data")
}

func TestGetXpathToStruct(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetXpathRequest(`/tns:element`, []Namespace{{"tns", "urn:tns"}})).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result = &Element{}
	err := ncs.GetXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `ABC`, result.Attr1, "Reply should contain response data")
}

func TestGetXpathExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetXpathRequest(`/tns:element`, []Namespace{{"tns", "urn:tns"}})).Return(nil, errors.New("failed"))

	var result string
	err := ncs.GetXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, &result)
	assert.Error(t, err, "Expecting call to fail")
}

func TestGetConfigSubtreeToString(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigSubtreeRequest(`<subtree-element/>`, RunningCfg)).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result string
	err := ncs.GetConfigSubtree(`<subtree-element/>`, RunningCfg, &result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `<element attr1="ABC"/>`, result, "Reply should contain response data")
}

func TestGetConfigSubtreeToStruct(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigSubtreeRequest(`<subtree-element/>`, RunningCfg)).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result = &Element{}
	err := ncs.GetConfigSubtree(`<subtree-element/>`, RunningCfg, result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `ABC`, result.Attr1, "Reply should contain response data")
}

func TestGetConfigSubtreeExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigSubtreeRequest(`<subtree-element/>`, RunningCfg)).Return(nil, errors.New("failed"))

	var result string
	err := ncs.GetConfigSubtree(`<subtree-element/>`, RunningCfg, &result)
	assert.Error(t, err, "Expecting call to fail")
}

func TestGetConfigXpathToString(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigXpathRequest(`/tns:element`, RunningCfg, []Namespace{{"tns", "urn:tns"}})).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result string
	err := ncs.GetConfigXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, RunningCfg, &result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `<element attr1="ABC"/>`, result, "Reply should contain response data")
}

func TestGetConfigXpathToStruct(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigXpathRequest(`/tns:element`, RunningCfg, []Namespace{{"tns", "urn:tns"}})).Return(&common.RPCReply{Data: `<data><element attr1="ABC"/></data>`}, nil)

	var result = &Element{}
	err := ncs.GetConfigXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, RunningCfg, result)
	assert.NoError(t, err, "Not expecting call to fail")
	assert.Equal(t, `ABC`, result.Attr1, "Reply should contain response data")
}

func TestGetConfigXpathExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetConfigXpathRequest(`/tns:element`, RunningCfg, []Namespace{{"tns", "urn:tns"}})).Return(nil, errors.New("failed"))

	var result string
	err := ncs.GetConfigXpath(`/tns:element`, []Namespace{{"tns", "urn:tns"}}, RunningCfg, &result)
	assert.Error(t, err, "Expecting call to fail")
}

func TestEditConfigString(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createEditConfigRequest(CandidateCfg, Cfg(`<configuration/>`))).Return(&common.RPCReply{}, nil)

	err := ncs.EditConfig(CandidateCfg, Cfg(`<configuration/>`))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

type testConfig struct {
	XMLName xml.Name `xml:"configuration"`
}

func TestEditConfigStruct(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createEditConfigRequest(CandidateCfg, Cfg(&testConfig{}))).Return(&common.RPCReply{}, nil)

	err := ncs.EditConfig(CandidateCfg, Cfg(&testConfig{}))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestEditConfigUrl(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createEditConfigRequest(CandidateCfg, CfgUrl("file://checkpoint.conf"))).Return(&common.RPCReply{}, nil)

	err := ncs.EditConfig(CandidateCfg, CfgUrl("file://checkpoint.conf"))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestEditConfigOptions(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute",
		createEditConfigRequest(CandidateCfg, Cfg(`<configuration/>`), ErrorOption(StopOnErrorErrOpt), DefaultOperation(NoneOp), TestOption(TestThenSetOpt))).Return(&common.RPCReply{}, nil)

	err := ncs.EditConfig(CandidateCfg, Cfg(`<configuration/>`), ErrorOption(StopOnErrorErrOpt), DefaultOperation(NoneOp), TestOption(TestThenSetOpt))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestEditConfigCfg(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createEditConfigRequest(CandidateCfg, Cfg(`<configuration/>`))).Return(&common.RPCReply{}, nil)

	err := ncs.EditConfigCfg(CandidateCfg, `<configuration/>`)
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestCopyConfig(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createCopyConfigRequest(DsName(CandidateCfg), DsUrl("file://checkpoint.conf"))).Return(&common.RPCReply{}, nil)

	err := ncs.CopyConfig(DsName(CandidateCfg), DsUrl("file://checkpoint.conf"))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestDeleteConfig(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createDeleteConfigRequest(DsUrl("file://checkpoint.conf"))).Return(&common.RPCReply{}, nil)

	err := ncs.DeleteConfig(DsUrl("file://checkpoint.conf"))
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestLock(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createLockRequest(CandidateCfg)).Return(&common.RPCReply{}, nil)

	err := ncs.Lock(CandidateCfg)
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestUnlock(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createUnlockRequest(CandidateCfg)).Return(&common.RPCReply{}, nil)

	err := ncs.Unlock(CandidateCfg)
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestDiscard(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createDiscardRequest()).Return(&common.RPCReply{}, nil)

	err := ncs.Discard()
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestCloseSession(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createCloseSessionRequest()).Return(&common.RPCReply{}, nil)

	err := ncs.CloseSession()
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestKillSession(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createKillSessionRequest(999)).Return(&common.RPCReply{}, nil)

	err := ncs.KillSession(999)
	assert.NoError(t, err, "Not expecting call to fail")

	mcli.AssertExpectations(t)
}

func TestGetSchemas(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)

	mcli.On("Execute", createGetShemasRequest()).Return(&common.RPCReply{Data: `
    <data>
	<netconf-state xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring">
	<schemas>
	<schema>
	<identifier>junos-rpc-internal-invoke</identifier>
	<version>2019-01-01</version>
	<format>yang</format>
	<namespace>http://yang.juniper.net/junos/rpc/internal-invoke</namespace>
	<location>NETCONF</location>
	</schema>
	<schema>
	<identifier>junos-rpc-telemetry-agentd</identifier>
	<version>2019-01-01</version>
	<format>yang</format>
	<namespace>http://yang.juniper.net/junos/rpc/telemetry-agentd</namespace>
	<location>NETCONF</location>
	</schema>
    </schemas>
    </netconf-state>
    </data>`}, nil)

	reply, err := ncs.GetSchemas()
	assert.NoError(t, err, "Not expecting call to fail")
	assert.NotNil(t, reply, "Reply should not be nil")
	assert.Equal(t, 2, len(reply))
	assert.Equal(t, "junos-rpc-internal-invoke", reply[0].Identifier)
	assert.Equal(t, "junos-rpc-telemetry-agentd", reply[1].Identifier)
}

func TestGetSchemasExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetShemasRequest()).Return(nil, errors.New("failure"))

	_, err := ncs.GetSchemas()
	assert.Error(t, err, "Expecting exec to fail")
}

func TestGetSchema(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetShemaRequest("id", "vsn", "yang")).
		Return(&common.RPCReply{Data: `<data>Some Yang</data>`}, nil)

	reply, err := ncs.GetSchema("id", "vsn", "yang")
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotEmpty(t, reply, "Reply should not be empty")
	assert.Equal(t, "Some Yang", reply)
}

func TestGetSchemaExecuteError(t *testing.T) {

	ncs, mcli := newOpsSessionWithMockClient(t)
	mcli.On("Execute", createGetShemaRequest("id", "vsn", "yang")).
		Return(nil, errors.New("failed"))

	reply, err := ncs.GetSchema("id", "vsn", "yang")
	assert.Error(t, err, "Expecting exec to fail")
	assert.Empty(t, reply, "Reply should be empty")
}

func newOpsSessionWithMockClient(t assert.TestingT) (OpSession, *mocks.OpSession) {
	mockClient := &mocks.OpSession{}
	return &sImpl{mockClient}, mockClient
}

type Element struct {
	XMLName xml.Name `xml:"element"`
	Attr1   string   `xml:"attr1,attr"`
}

// Simple real NE access tests
//
//func TestRealGetSubtree(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	netconfst := &NetconfState{}
//	err := ncs.GetSubtree("<netconf-state><schemas/></netconf-state>", netconfst)
//	assert.NoError(t, err, "Not expecting exec to fail")
//	assert.NotEmpty(t, netconfst.Schemas.Schema, "Reply should be non-nil")
//}
//
//func TestRealGetXpath(t *testing.T) {
//
//	ncs := setupServer2(t)
//	defer ncs.Close()
//
//	netconfst := &NetconfState{}
//	err := ncs.GetXpath("/nm:netconf-state", []Namespace{{"nm", "urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring"}}, netconfst)
//	assert.NoError(t, err, "Not expecting exec to fail")
//	assert.NotEmpty(t, netconfst.Schemas.Schema, "Reply should be non-nil")
//}
//
//func TestRealGetConfigSubtree(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	var result string
//	err := ncs.GetConfigSubtree(nil, RunningCfg, &result)
//	assert.NoError(t, err, "Not expecting exec to fail")
//	assert.NotEmpty(t, result, "Reply should be non-nil")
//}
//
//func TestRealEditConfig(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.EditConfig(CandidateCfg, Cfg("<configuration/>"), ErrorOption(StopOnErrorErrOpt), DefaultOperation(NoneOp), TestOption(TestThenSetOpt))
//	assert.NoError(t, err, "Not expecting exec to fail")
//
//	err = ncs.EditConfig(CandidateCfg, Cfg(&testConfig{}), ErrorOption(StopOnErrorErrOpt), DefaultOperation(NoneOp), TestOption(TestThenSetOpt))
//	assert.NoError(t, err, "Not expecting exec to fail")
//}
//
//func TestRealCopyConfig(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.CopyConfig(DsName(RunningCfg), DsUrl("file://checkpoint.conf"))
//	assert.Error(t, err, "Expecting exec to fail") // no such file...
//}
//
//func TestRealDeleteConfig(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.DeleteConfig(DsName(CandidateCfg))
//	assert.NoError(t, err, "Not expecting exec to fail") // no such file...
//}
//
//func TestRealDiscard(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.Discard()
//	assert.NoError(t, err, "Not expecting exec to fail")
//}
//
//func TestRealCloseSession(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.CloseSession()
//	assert.NoError(t, err, "Not expecting exec to fail")
//
//	err = ncs.CloseSession()
//	assert.Error(t, err, "Expecting exec to fail")
//}
//
//func TestRealLockUnlock(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	lkerr := ncs.Lock(CandidateCfg)
//	unlkerr := ncs.Unlock(CandidateCfg)
//	assert.NoError(t, unlkerr, "Not expecting unlock to fail")
//	assert.NoError(t, lkerr, "Not expecting lock to fail")
//}
//
//func TestRealGetSchemas(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	reply, err := ncs.GetSchemas()
//
//	assert.NoError(t, err, "Not expecting exec to fail")
//	assert.NotNil(t, reply, "Reply should be non-nil")
//	fmt.Println(reply)
//}
//
//func TestRealGetSchema(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	//reply, err := ncs.GetSchema("junos-rpc-vpls", "2019-01-01", "yang")
//	reply, err := ncs.GetSchema("junos-rpc-vpls", "", "")
//
//	assert.NoError(t, err, "Not expecting exec to fail")
//	assert.NotNil(t, reply, "Reply should be non-nil")
//  fmt.Println(reply)
//}
//
//func TestRealKillSession(t *testing.T) {
//
//	ncs := setupServer1(t)
//	defer ncs.Close()
//
//	err := ncs.KillSession(ncs.ID())
//
//	assert.Error(t, err, "Expecting exec to fail")
//	assert.Contains(t, err.Error(), "You do not want to kill yourself")
//}
//
//func setupServer1(t *testing.T) OpSession {
//	sshConfig := &ssh.ClientConfig{
//		User:            server1User,
//		Auth:            []ssh.AuthMethod{ssh.Password(server1Password)},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	}
//
//	ctx := client.WithClientTrace(context.Background(), client.DefaultLoggingHooks)
//
//	ncs, err := NewSession(ctx, sshConfig, fmt.Sprintf("%s:%d", server1Address, 830))
//	assert.NoError(t, err, "Not expecting new session to fail")
//	assert.NotNil(t, ncs, "OpSession should be non-nil")
//	return ncs
//}
//
//func setupServer2(t *testing.T) OpSession {
//	sshConfig := &ssh.ClientConfig{
//		User:            server2User,
//		Auth:            []ssh.AuthMethod{ssh.Password(server2Password)},
//		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
//	}
//
//	ctx := client.WithClientTrace(context.Background(), client.DefaultLoggingHooks)
//
//	ncs, err := NewSession(ctx, sshConfig, fmt.Sprintf("%s:%d", server2Address, 830))
//	assert.NoError(t, err, "Not expecting new session to fail")
//	assert.NotNil(t, ncs, "OpSession should be non-nil")
//	return ncs
//}
//
//const (
//	server1Address  = "10.228.63.5"
//	server1User     = "regress"
//	server1Password = "M......"
//
//	server2Address  = "172.26.138.58"
//	server2User     = "WRuser"
//	server2Password = "B......"
//)
