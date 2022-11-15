package ops

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/damianoneill/net/v2/netconf/client"

	"github.com/damianoneill/net/v2/netconf/common"
)

type Namespace struct {
	ID   string
	Path string
}

// OpSession represents a Netconf Operations OpSession
type OpSession interface {
	client.Session

	// GetSubtree issues a GET request, with the supplied subtree filter and stores the response in the result, which
	// should be the address of either:
	// - a string, in which case it will hold the response body, or
	// - a struct with xml tags.
	GetSubtree(filter interface{}, result interface{}) error

	// GetXpath issues a GET request, with the supplied xpath filter and namespace list and stores the response in the result, which
	// should be the address of either:
	// - a string, in which case it will hold the response body, or
	// - a struct with xml tags.
	GetXpath(xpath string, nslist []Namespace, result interface{}) error

	// GetConfigSubtree issues a GET-CONFIG request, with the supplied subtree filter and source, and stores the
	// response in the result, which should be the address of either:
	// - a string, in which case it will hold the response body, or
	// - a struct with xml tags.
	GetConfigSubtree(filter interface{}, source string, result interface{}) error

	// GetConfigXpath issues a GET-CONFIG request, with the supplied xpath filter, source and namespace list and stores the
	// response in the result, which should be the address of either:
	// - a string, in which case it will hold the response body, or
	// - a struct with xml tags.
	GetConfigXpath(xpath string, nslist []Namespace, source string, result interface{}) error

	// GetSchemas returns an array of schemas supported by the device.
	GetSchemas() ([]Schema, error)

	// GetSchema returns the text of the schema identified by id and version, in the format defined by fmt.
	GetSchema(id, version, fmt string) (string, error)

	// EditConfig issues an edit-config request defined by config to be applied to the target configuration.
	// EditOptions can be added to qualify the operation.
	// config will be defined by a ConfigOption, which can be one of:
	// - Cfg(cfg), where cfg is
	//   o   an xml string, in which case it will be used verbatim as the content of the <config> element.
	//   o   a struct with xml tags that will be marshalled as the child of the <config> element.
	// - CfgURL(url), in which case the configuration is defined by a <url> element.
	EditConfig(target string, config ConfigOption, options ...EditOption) error

	// EditConfigCfg issues an edit-config request defined by config to be applied to the target configuration.
	// EditOptions can be added to qualify the operation.
	// Convenience method to avoid complications with function arguments when using EditConfig() with a mock object
	EditConfigCfg(target string, config interface{}, options ...EditOption) error

	// CopyConfig issues a copy-config request.
	// source and target are defined by a CfgDsOpt, which can be one of:
	// - DsName(name) where name defines the configuration data store name (Running, Candidate ...)
	// - DsURL(url) where url defines the url of the datastore
	CopyConfig(source, target CfgDsOpt) error

	// DeleteConfig issues a delete-config request.
	// target is defined by a CfgDsOpt, which can be one of:
	// - DsName(name) where name defines the configuration data store name (Running, Candidate ...)
	// - DsURL(url) where url defines the url of the datastore to be deleted
	DeleteConfig(target CfgDsOpt) error

	// Lock issues a lock request on the target configuration.
	Lock(target string) error

	// Unlock issues an unlock request on the target configuration.
	Unlock(target string) error

	// Discard issues a discard changes request.
	Discard() error

	// CloseSession issues a close session request.
	CloseSession() error

	// KillSession issues a kill session request for the specified session id.
	KillSession(id uint64) error
}

type sImpl struct {
	client.Session
}

func (s *sImpl) Close() {
	s.Session.Close()
}

func (s *sImpl) GetSubtree(filter, result interface{}) error {
	return s.handleGetRequest(createGetSubtreeRequest(filter), result)
}

func (s *sImpl) GetXpath(xpath string, nslist []Namespace, result interface{}) error {
	return s.handleGetRequest(createGetXpathRequest(xpath, nslist), result)
}

func (s *sImpl) GetConfigSubtree(filter interface{}, source string, result interface{}) error {
	return s.handleGetRequest(createGetConfigSubtreeRequest(filter, source), result)
}

func (s *sImpl) GetConfigXpath(xpath string, nslist []Namespace, source string, result interface{}) error {
	return s.handleGetRequest(createGetConfigXpathRequest(xpath, source, nslist), result)
}

func (s *sImpl) EditConfig(target string, config ConfigOption, options ...EditOption) error {
	_, err := s.Session.Execute(createEditConfigRequest(target, config, options...))
	return err
}

func (s *sImpl) EditConfigCfg(target string, config interface{}, options ...EditOption) error {
	return s.EditConfig(target, Cfg(config), options...)
}

func (s *sImpl) CopyConfig(source, target CfgDsOpt) error {
	_, err := s.Session.Execute(createCopyConfigRequest(source, target))
	return err
}

func (s *sImpl) DeleteConfig(target CfgDsOpt) error {
	_, err := s.Session.Execute(createDeleteConfigRequest(target))
	return err
}

func (s *sImpl) Lock(target string) error {
	_, err := s.Session.Execute(createLockRequest(target))
	return err
}

func (s *sImpl) Unlock(target string) error {
	_, err := s.Session.Execute(createUnlockRequest(target))
	return err
}

func (s *sImpl) Discard() error {
	_, err := s.Session.Execute(createDiscardRequest())
	return err
}

func (s *sImpl) CloseSession() error {
	_, err := s.Session.Execute(createCloseSessionRequest())
	return err
}

func (s *sImpl) KillSession(id uint64) error {
	_, err := s.Session.Execute(createKillSessionRequest(id))
	return err
}

func (s *sImpl) GetSchemas() ([]Schema, error) {
	ncs := &NetconfState{}
	err := s.handleGetRequest(createGetShemasRequest(), ncs)
	if err != nil {
		return nil, err
	}
	return ncs.Schemas.Schema, nil
}

func (s *sImpl) GetSchema(id, version, format string) (string, error) {
	req := createGetShemaRequest(id, version, format)
	rply, err := s.Session.Execute(req)
	if err != nil {
		return "", err
	}
	data := &Data{}
	err = xml.Unmarshal([]byte(rply.Data), data)
	return data.Content, err
}

// Request structs.

type Filter struct {
	XMLName xml.Name `xml:"filter"`
	Type    string   `xml:"type,attr"`
	Select  string   `xml:"select,attr,omitempty"`
	*common.Union
}

type Config struct {
	XMLName xml.Name `xml:"config"`
	*common.Union
}

type GetReq struct {
	XMLName xml.Name `xml:"get"`
	Filter  *Filter
}

type ConfigType struct {
	Type string `xml:",innerxml"`
	URL  string `xml:"url,omitempty"`
}

type GetConfigReq struct {
	XMLName    xml.Name    `xml:"get-config"`
	Source     *ConfigType `xml:"source"`
	Filter     *Filter
	FilterBody string `xml:",innerxml"`
}

type EditConfigReq struct {
	XMLName          xml.Name    `xml:"edit-config"`
	Target           *ConfigType `xml:"target"`
	ErrorOption      string      `xml:"error-option,omitempty"`
	TestOption       string      `xml:"test-option,omitempty"`
	DefaultOperation string      `xml:"default-operation,omitempty"`
	Config           *Config
	ConfigURL        string `xml:"url,omitempty"`
}

type CopyConfigReq struct {
	XMLName xml.Name    `xml:"copy-config"`
	Target  *ConfigType `xml:"target"`
	Source  *ConfigType `xml:"source"`
}

type DeleteConfigReq struct {
	XMLName xml.Name    `xml:"delete-config"`
	Target  *ConfigType `xml:"target"`
}

type LockReq struct {
	XMLName xml.Name    `xml:"lock"`
	Target  *ConfigType `xml:"target"`
}

type UnlockReq struct {
	XMLName xml.Name    `xml:"unlock"`
	Target  *ConfigType `xml:"target"`
}

type DiscardReq struct {
	XMLName xml.Name `xml:"discard-changes"`
}

type CloseSessionReq struct {
	XMLName xml.Name `xml:"close-session"`
}

type KillSessionReq struct {
	XMLName xml.Name `xml:"kill-session"`
	ID      uint64   `xml:"session-id"`
}

type GetSchema struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring get-schema"`
	ID      string   `xml:"identifier"`
	Vsn     string   `xml:"version"`
	Fmt     string   `xml:"format"`
}

// ConfigOption defines the configuration to be applied by an edit config operation
type ConfigOption func(*EditConfigReq)

func Cfg(cfg interface{}) ConfigOption {
	return func(req *EditConfigReq) {
		req.Config = &Config{Union: common.GetUnion(cfg)}
	}
}

func CfgURL(url string) ConfigOption {
	return func(req *EditConfigReq) {
		req.ConfigURL = url
	}
}

// CfgDsOpt
type CfgDsOpt func(*ConfigType)

func DsName(name string) CfgDsOpt {
	return func(t *ConfigType) {
		t.Type = "<" + name + "/>"
	}
}

func DsURL(url string) CfgDsOpt {
	return func(t *ConfigType) {
		t.URL = url
	}
}

// EditOption configures an edit config operation.
type EditOption func(*EditConfigReq)

func DefaultOperation(oper string) EditOption {
	return func(req *EditConfigReq) {
		req.DefaultOperation = oper
	}
}

func TestOption(opt string) EditOption {
	return func(req *EditConfigReq) {
		req.TestOption = opt
	}
}

func ErrorOption(opt string) EditOption {
	return func(req *EditConfigReq) {
		req.ErrorOption = opt
	}
}

func (r *EditConfigReq) applyOpts(options ...EditOption) {
	for _, opt := range options {
		opt(r)
	}
}

func createGetSubtreeRequest(s interface{}) common.Request {
	req := &GetReq{}
	if s != nil {
		req.Filter = &Filter{Type: "subtree", Union: common.GetUnion(s)}
	}
	return req
}

func createGetXpathRequest(xpath string, nslist []Namespace) common.Request {
	return fmt.Sprintf(`<get><filter %s type="xpath" select=%q/></get>`, getNamespaceAttributes(nslist), xpath)
}

func getNamespaceAttributes(nslist []Namespace) string {
	var attrs string
	for _, ns := range nslist {
		attrs = fmt.Sprintf(`%s xmlns:%s=%q`, attrs, ns.ID, ns.Path)
	}
	return strings.TrimSpace(attrs)
}

func createGetConfigSubtreeRequest(s interface{}, source string) common.Request {
	// xml Marshaller will not create self-closing tags (and some devices require it)...
	req := &GetConfigReq{Source: &ConfigType{Type: "<" + source + "/>"}}
	if s != nil {
		req.Filter = &Filter{Type: "subtree", Union: common.GetUnion(s)}
	}
	return req
}

func createGetConfigXpathRequest(xpath, source string, nslist []Namespace) common.Request {
	// xml Marshaller will not create self-closing tags....
	req := &GetConfigReq{Source: &ConfigType{Type: "<" + source + "/>"}}
	if xpath != "" {
		req.FilterBody = createXpathFilter(xpath, nslist)
	}
	return req
}

func createXpathFilter(xpath string, nslist []Namespace) string {
	return fmt.Sprintf(`<filter %s type="xpath" select=%q/>`, getNamespaceAttributes(nslist), xpath)
}

func createEditConfigRequest(target string, cfgOpt ConfigOption, options ...EditOption) *EditConfigReq {
	req := &EditConfigReq{Target: &ConfigType{Type: "<" + target + "/>"}}
	req.applyOpts(options...)
	cfgOpt(req)
	return req
}

func createCopyConfigRequest(source, target CfgDsOpt) *CopyConfigReq {
	req := &CopyConfigReq{Source: &ConfigType{}, Target: &ConfigType{}}
	source(req.Source)
	target(req.Target)
	return req
}

func createDeleteConfigRequest(target CfgDsOpt) *DeleteConfigReq {
	req := &DeleteConfigReq{Target: &ConfigType{}}
	target(req.Target)
	return req
}

func createLockRequest(target string) *LockReq {
	return &LockReq{Target: &ConfigType{Type: "<" + target + "/>"}}
}

func createUnlockRequest(target string) *UnlockReq {
	return &UnlockReq{Target: &ConfigType{Type: "<" + target + "/>"}}
}

func createDiscardRequest() *DiscardReq {
	return &DiscardReq{}
}

func createKillSessionRequest(id uint64) *KillSessionReq {
	return &KillSessionReq{ID: id}
}

func createCloseSessionRequest() *CloseSessionReq {
	return &CloseSessionReq{}
}

func createGetShemaRequest(id, version, format string) common.Request {
	return &GetSchema{ID: id, Vsn: version, Fmt: format}
}

func createGetShemasRequest() common.Request {
	return createGetSubtreeRequest("<netconf-state><schemas/></netconf-state>")
}

func (s *sImpl) handleGetRequest(req common.Request, result interface{}) error {
	reply, err := s.Session.Execute(req)
	if err != nil {
		return err
	}

	switch target := result.(type) {
	case *string:
		data := &Data{}
		err = xml.Unmarshal([]byte(reply.Data), data)
		*target = data.Content
	default:
		data := &Data{Body: result}
		err = xml.Unmarshal([]byte(reply.Data), data)
	}
	return err
}
