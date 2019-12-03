package ops

import "encoding/xml"

const (
	// Configuration Datastores
	RunningCfg   = "running"
	CandidateCfg = "candidate"
	StartupCfg   = "startup"

	// Edit Config Error Options
	StopOnErrorErrOpt     = "stop-on-error"
	ContinueOnErrorErrOpt = "continue-on-error"
	RollbackOnErrorErrOpt = "rollback-on-error"

	// Edit Config Operation Types
	MergeOp   = "merge"
	ReplaceOp = "replace"
	NoneOp    = "none"

	// Edit Config Test Options
	TestThenSetOpt = "test-then-set"
	SetOpt         = "set"
	TestOnlyOpt    = "test-only"
)

type Data struct {
	XMLName xml.Name    `xml:"data"`
	Body    interface{} `xml:",any""`
	Content string      `xml:",innerxml"`
}

type Schema struct {
	Identifier string `xml:"identifier"`
	Version    string `xml:"version"`
	Format     string `xml:"format"`
	Namespace  string `xml:"namespace"`
	Location   string `xml:"location"`
}

type NetconfState struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring netconf-state`
	Xmlns   string   `xml:"xmlns,attr"`
	Schemas struct {
		Schema []Schema `xml:"schema"`
	} `xml:"schemas"`
	Capabilities struct {
		Text       string   `xml:",chardata"`
		Capability []string `xml:"capability"`
	} `xml:"capabilities"`
	Statistics struct {
		Text             string `xml:",chardata"`
		NetconfStartTime string `xml:"netconf-start-time"`
		InBadHellos      string `xml:"in-bad-hellos"`
		InSessions       string `xml:"in-sessions"`
		DroppedSessions  string `xml:"dropped-sessions"`
		InRpcs           string `xml:"in-rpcs"`
		InBadRpcs        string `xml:"in-bad-rpcs"`
		OutRpcErrors     string `xml:"out-rpc-errors"`
		OutNotifications string `xml:"out-notifications"`
	} `xml:"statistics"`
	Sessions struct {
		Text    string `xml:",chardata"`
		Session struct {
			Text             string `xml:",chardata"`
			SessionID        string `xml:"session-id"`
			Transport        string `xml:"transport"`
			Username         string `xml:"username"`
			SourceHost       string `xml:"source-host"`
			LoginTime        string `xml:"login-time"`
			InRpcs           string `xml:"in-rpcs"`
			InBadRpcs        string `xml:"in-bad-rpcs"`
			OutRpcErrors     string `xml:"out-rpc-errors"`
			OutNotifications string `xml:"out-notifications"`
		} `xml:"session"`
	} `xml:"sessions"`
}
