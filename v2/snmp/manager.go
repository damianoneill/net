package snmp

import (
	"context"
	"encoding/asn1"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/geoffgarside/ber"
)

// Manager provides an interface for SNMP device management.
type Manager interface {
	// Issues an SNMP GET request for the specified oids.
	Get(ctx context.Context, oids []string) (*PDU, error)

	// Issues an SNMP GET NEXT request for the specified oids.
	GetNext(ctx context.Context, oids []string) (*PDU, error)

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

// PDU defines an SNMP PDU.
type PDU struct {
	RequestID   int32
	Error       int
	ErrorIndex  int
	VarbindList []Varbind
}

type managerImpl struct {
	conn          net.Conn
	config        *managerConfig
	nextRequestID int32
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

const maxInputBufferSize = 65535

type messageType byte

const getMessage = 0xA0
const getNextMessage = 0xA1

func (m *managerImpl) Get(ctx context.Context, oids []string) (*PDU, error) {
	return m.executeGet(ctx, getMessage, oids)
}

func (m *managerImpl) GetNext(ctx context.Context, oids []string) (*PDU, error) {
	return m.executeGet(ctx, getNextMessage, oids)
}

func (m *managerImpl) GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetWalk(ctx context.Context, oid string, walker Walker) error {
	// TODO
	return nil
}

func (m *managerImpl) executeGet(ctx context.Context, mType messageType, oids []string) (*PDU, error) {
	ctx, cancel := context.WithTimeout(ctx, m.config.timeout)
	defer cancel()
	deadline, _ := ctx.Deadline()
	err := m.conn.SetDeadline(deadline)
	if err != nil {
		return nil, err
	}

	b, err := m.buildPacket(oids, mType)
	if err != nil {
		return nil, err
	}

	n, err := m.conn.Write(b)
	if err != nil {
		return nil, err
	}

	input, err := m.readResponse(n, err)
	if err != nil {
		// TODO Handle EOF
		return nil, err
	}

	return m.parseResponse(input)
}

func (m *managerImpl) readResponse(n int, err error) ([]byte, error) {
	input := make([]byte, maxInputBufferSize)

	n, err = m.conn.Read(input[:])

	if err != nil {
		return nil, err
	}

	if n == maxInputBufferSize {
		// Never expect this to happen
		panic(fmt.Errorf("overflowing response buffer"))
	}
	return input, nil
}

func (m *managerImpl) parseResponse(input []byte) (*PDU, error) {

	pkt := &packet{}

	_, err := ber.Unmarshal(input, pkt)

	// Replace SNMP PDU Type with ASN1 sequence tag.
	pkt.RequestType.FullBytes[0] = 0x30

	pdu := &PDU{}
	_, err = ber.Unmarshal(pkt.RequestType.FullBytes, pdu)
	if err != nil {
		return nil, err
	}
	return pdu, nil
}

func (m *managerImpl) buildPacket(oids []string, mType messageType) ([]byte, error) {
	pdu1 := PDU{
		RequestID:   m.nextID(),
		Error:       0,
		ErrorIndex:  0,
		VarbindList: buildVarbindList(oids),
	}

	b, err := ber.Marshal(pdu1)
	if err != nil {
		return nil, err
	}

	b[0] = byte(mType)

	p := packet{
		Version:     m.config.version,
		Community:   []byte(m.config.community),
		RequestType: asn1.RawValue{FullBytes: b},
	}

	b, err = ber.Marshal(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *managerImpl) nextID() (id int32) {
	id = m.nextRequestID
	m.nextRequestID++
	return
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
