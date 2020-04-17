package snmp

import (
	"context"
	"encoding/asn1"
	"net"
	"strconv"
	"strings"
)

// Manager provides an interface for SNMP device management.
type Manager interface {
	// Issues an SNMP GET request for the specified oids.
	Get(ctx context.Context, oids []string) (*Response, error)

	// Issues an SNMP GET NEXT request for the specified oids.
	GetNext(ctx context.Context, oids []string) (*Response, error)

	// Issues an SNMP GET BULK request for the specified oids.
	GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*Response, error)

	// Issues an SNMP GET BULK request starting from the specified oid, invoking the function walker for each PDU.
	GetWalk(ctx context.Context, oid string, walker Walker) error
}

// Response defines the response to a Get... request.
type Response struct {
	// TBD
}

// Walker defines a function that will be called for each PDU processed by the GetWalk method.
// If the function returns an error, the walk will be terminated.
type Walker func(pdu *PDU) error

// PDU defines ...
type PDU struct {
	// TBD
}

type managerImpl struct {
	conn   net.Conn
	config *managerConfig
}

type Varbind struct {
	OID   asn1.ObjectIdentifier
	Value interface{}
}
type packet struct {
	Version     SNMPVersion
	Community   []byte
	RequestType asn1.RawValue
}

type pdu struct {
	RequestID   int
	Error       int
	ErrorIndex  int
	VarbindList []Varbind
}

func (m *managerImpl) Get(ctx context.Context, oids []string) (*Response, error) {

	pdu1 := pdu{
		RequestID:   1,
		Error:       0,
		ErrorIndex:  0,
		VarbindList: buildVarbindList(oids),
	}

	b, err := asn1.Marshal(pdu1)
	if err != nil {
		return nil, err
	}

	b[0] = 0xA0 // GetRequest

	p := packet{
		Version:     m.config.version,
		Community:   []byte("private"),
		RequestType: asn1.RawValue{FullBytes: b},
	}

	b, err = asn1.Marshal(p)
	if err != nil {
		return nil, err
	}
	//for i := range b {
	//	fmt.Printf("%02x ", b[i])
	//}

	m.conn.Write(b)
	return nil, nil
}

func (m *managerImpl) GetNext(ctx context.Context, oids []string) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetWalk(ctx context.Context, oid string, walker Walker) error {
	// TODO
	return nil
}

func buildVarbindList(oids []string) []Varbind {
	vbl := make([]Varbind, len(oids))
	for i := 0; i < len(oids); i++ {
		vbl[i] = Varbind{OID: oidToInts(oids[0]), Value: asn1.NullRawValue}
	}
	return vbl
}

func oidToInts(input string) []int {

	// TODO - prevalidate OIDS on entry.
	// Remove leading/trailing periods and split into oid components.
	oidValues := strings.Split(strings.Trim(input, "."), ".")

	// Convert to ints.
	oidInts := make([]int, len(oidValues))
	for i := 0; i < len(oidValues); i++ {
		var err error
		oidInts[i], err = strconv.Atoi(oidValues[i])
		if err != nil {
			panic(err)
		}
	}
	return oidInts
}
