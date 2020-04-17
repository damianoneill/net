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

// Session provides an interface for SNMP device management.
type Session interface {
	// Issues an SNMP GET request for the specified oids.
	// Get request processing is described at https://tools.ietf.org/html/rfc1905#section-4.2.1.
	Get(ctx context.Context, oids []string) (*PDU, error)

	// Issues an SNMP GET NEXT request for the specified oids.
	// Get Bext request processing is described athttps://tools.ietf.org/html/rfc1905#section-4.2.2.
	GetNext(ctx context.Context, oids []string) (*PDU, error)

	// Issues an SNMP GET BULK request for the specified oids.
	// Get Bulk request processing is described at https://tools.ietf.org/html/rfc1905#section-4.2.3
	GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*PDU, error)

	// Issues SNMP GET NEXT requests starting from the specified root oid, invoking the function walker for each
	// variable that is a descendant of the root oid.
	Walk(ctx context.Context, rootOid string, walker Walker) error

	// Issues SNMP GET BULK requests starting from the specified root oid, invoking the function walker for each
	// variable that is a descendant of the root oid.
	BulkWalk(ctx context.Context, rootOid string, maxRepetitions int, walker Walker) error
}

// Walker defines a function that will be called for each variable processed by the Walk/BulkWalk methods.
// If the function returns an error, the walk will be terminated.
type Walker func(vb *Varbind) error

// PDU defines an SNMP PDU, as returned by the Get/GetNext methods.
// Note that it differs from rawPDU in that the variable bindings define value using golang types, rather than
// the ASN.1 transport format.
type PDU struct {
	RequestID int32
	// Non-zero used to indicate that an exception occurred to prevent the processing of the request
	Error int
	// If Error is non-zero, identifies which variable binding in the list caused the exception
	ErrorIndex  int
	VarbindList []Varbind
}

type Varbind struct {
	OID        asn1.ObjectIdentifier
	TypedValue *TypedValue
}

// Encapsulates the data type and value of a variable received in a variable binding from an agent.
type TypedValue struct {
	Type  DataType
	Value interface{}
}

type sessionImpl struct {
	conn          net.Conn
	config        *sessionConfig
	nextRequestID int32
}

// rawPDU defines the pdu that is used to passed to/from an SNMP agent.
type rawPDU struct {
	RequestID int32
	// Non-zero used to indicate that an exception occurred to prevent the processing of the request
	Error int
	// If Error is non-zero, identifies which variable binding in the list caused the exception
	ErrorIndex  int
	VarbindList []rawVarbind
}

type rawVarbind struct {
	OID   asn1.ObjectIdentifier
	Value asn1.RawValue
}

// Defines the SNMP packet passed over the network to/from an SNMP agent.
// Note the pdu is initially unmarshalled as a raw value, so that the SNMP message type can be replaced by
// the ASN1 sequence tag before the variable bindings it contains are unmarshalled.
type packet struct {
	Version   SNMPVersion
	Community []byte
	RawPdu    asn1.RawValue
}

const maxInputBufferSize = 65535

// Supported SNMP message types.
type messageType byte

const getMessage = 0xA0
const getNextMessage = 0xA1
const getBulkMessage = 0xA5

func (m *sessionImpl) Get(ctx context.Context, oids []string) (*PDU, error) {
	return m.executeGet(ctx, getMessage, oids, 0, 0)
}

func (m *sessionImpl) GetNext(ctx context.Context, oids []string) (*PDU, error) {
	return m.executeGet(ctx, getNextMessage, oids, 0, 0)
}

func (m *sessionImpl) GetBulk(ctx context.Context, oids []string, nonRepeaters, maxRepetitions int) (*PDU, error) {
	return m.executeGet(ctx, getBulkMessage, oids, nonRepeaters, maxRepetitions)
}

func (m *sessionImpl) Walk(ctx context.Context, rootOid string, walker Walker) error {
	return m.executeWalk(ctx, getNextMessage, 0, rootOid, walker)
}

func (m *sessionImpl) BulkWalk(ctx context.Context, rootOid string, maxRepetitions int, walker Walker) error {
	return m.executeWalk(ctx, getBulkMessage, maxRepetitions, rootOid, walker)
}

// Generic Get execution.
// Generates a packet to define the type of Get, the required oids and, in the case of a bulk get, the associated
// non-repeaters and max-repetitions values.
// Returns a PDU with the resolved variable bindings.
func (m *sessionImpl) executeGet(ctx context.Context, getType messageType, oids []string, nonRepeaters, maxRepetitions int) (*PDU, error) {

	// TODO Validate OIDs on entry.

	// Keep trying until we succeed, a non-timeout error occurs or the retry limit is reached.
	for i := 0; ; i++ {
		ctx, cancel := context.WithTimeout(ctx, m.config.timeout)
		defer cancel()
		deadline, _ := ctx.Deadline()
		err := m.conn.SetDeadline(deadline)
		if err != nil {
			return nil, err
		}

		b, err := m.buildPacket(oids, getType, nonRepeaters, maxRepetitions)
		if err != nil {
			return nil, err
		}

		err = m.writePacket(b)
		if err != nil {
			return nil, err
		}

		input, err := m.readResponse()
		if err != nil {
			// Check for a timeout and retry if allowed.
			e, ok := err.(net.Error)
			if ok && e.Timeout() && i < m.config.retries {
				continue
			}
			return nil, err
		}
		return m.parseResponse(input)
	}
}

// Generic Walk execution.
func (m *sessionImpl) executeWalk(ctx context.Context, mType messageType, maxRepetitions int, rootOid string, walker Walker) error {

	nextOid := rootOid
	for {
		pdu, err := m.executeGet(ctx, mType, []string{nextOid}, 0, maxRepetitions)
		if err != nil {
			// TODO More intelligence!
			return err
		}
		for i := range pdu.VarbindList {
			vb := &pdu.VarbindList[i]
			if !isOidDescendantOfRoot(vb.OID, rootOid) {
				return nil
			}
			err = walker(vb)
			if err != nil {
				return err
			}
			if vb.TypedValue.Type == EndOfMib {
				return nil
			}
		}
		nextOid = pdu.VarbindList[len(pdu.VarbindList)-1].OID.String()
	}
}

// Determines whether oid is a 'descendant' of the rootOid.
func isOidDescendantOfRoot(oid asn1.ObjectIdentifier, rootOid string) bool {
	return strings.HasPrefix(oid.String(), rootOid+".")
}

func (m *sessionImpl) writePacket(b []byte) (err error) {
	n, err := m.conn.Write(b)
	m.config.trace.WriteComplete(m.config, b[0:n], err)
	return
}

func (m *sessionImpl) readResponse() (input []byte, err error) {
	input = make([]byte, maxInputBufferSize)

	n, err := m.conn.Read(input[:])
	defer m.config.trace.ReadComplete(m.config, input[0:n], err)
	if err != nil {
		return nil, err
	}

	if n == maxInputBufferSize {
		// Never expect this to happen
		return nil, fmt.Errorf("overflowing response buffer")
	}

	return input[0:n], nil
}

// Parses the packet returned by a get request, returning the PDU with the resolved variable bindings.
func (m *sessionImpl) parseResponse(input []byte) (*PDU, error) {

	// We use a BER unmarshaler; this is unaware of SNMP RawPdu and data types.
	// Consequently, there are 3 stages to the unmarshalling.
	// Stage 1: the packet envelope is unmarshalled but the PDU is left as a raw ASN1 value.
	// The first byte of the raw RawPdu is changed from the SNMP message tag to the ASN1 Sequence tag.
	// Stage 2: the raw RawPdu and its variable bindings are unmarshalled. However, the values of the variable
	// bindings are left as ASN1 raw values.
	// Stage 3: get the datatype tag of each raw variable binding to determine what golang scalar type should be used to
	// represent the variable, then replace the tag with the appropriate ASN1 tag and unmarshal the value.

	pkt := &packet{}
	_, err := ber.Unmarshal(input, pkt)
	if err != nil {
		return nil, err
	}

	// Replace SNMP PDU Type with ASN1 sequence tag.
	pkt.RawPdu.FullBytes[0] = 0x30

	rawPDU := &rawPDU{}
	_, err = ber.Unmarshal(pkt.RawPdu.FullBytes, rawPDU)
	if err != nil {
		return nil, err
	}

	return unmarshalValues(rawPDU)
}

func unmarshalValues(raw *rawPDU) (*PDU, error) {

	pdu := &PDU{
		RequestID:   raw.RequestID,
		Error:       raw.Error,
		ErrorIndex:  raw.ErrorIndex,
		VarbindList: make([]Varbind, len(raw.VarbindList)),
	}
	for i := range raw.VarbindList {
		value, err := unmarshalVariable(&raw.VarbindList[i].Value)
		if err != nil {
			return nil, err
		}
		pdu.VarbindList[i].OID = raw.VarbindList[i].OID
		pdu.VarbindList[i].TypedValue = value
	}
	return pdu, nil
}

func (m *sessionImpl) buildPacket(oids []string, mType messageType, nonRepeaters, maxRepetitions int) ([]byte, error) {
	pdu := rawPDU{
		RequestID:   m.nextID(),
		VarbindList: buildVarbindList(oids),
	}

	if mType == getBulkMessage {
		pdu.Error = nonRepeaters
		pdu.ErrorIndex = maxRepetitions
	}
	b, err := ber.Marshal(pdu)
	if err != nil {
		return nil, err
	}

	b[0] = byte(mType)

	p := packet{
		Version:   m.config.version,
		Community: []byte(m.config.community),
		RawPdu:    asn1.RawValue{FullBytes: b},
	}

	b, err = ber.Marshal(p)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *sessionImpl) nextID() (id int32) {
	id = m.nextRequestID
	m.nextRequestID++
	return
}

func buildVarbindList(oids []string) []rawVarbind {
	vbl := make([]rawVarbind, len(oids))
	for i := 0; i < len(oids); i++ {
		vbl[i].OID = oidToInts(oids[i])
		vbl[i].Value = asn1.NullRawValue
	}
	return vbl
}

func oidToInts(input string) []int {

	// Remove leading/trailing periods and split into oid components.
	oidValues := strings.Split(strings.Trim(input, "."), ".")

	// Convert to ints.
	oidInts := make([]int, len(oidValues))
	for i := 0; i < len(oidValues); i++ {
		var err error
		oidInts[i], err = strconv.Atoi(oidValues[i])
		if err != nil {
			// This is acceptable, provided we validate all OID values on entry; see earlier TODO.
			panic(err)
		}
	}
	return oidInts
}
