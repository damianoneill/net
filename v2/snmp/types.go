package snmp

import (
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/geoffgarside/ber"
)

// Definitions and methods used to unmarshal ASN1 and SNMP datatypes from ASN1 raw values.
// Refer to http://luca.ntop.org/Teaching/Appunti/asn1.html.

// Define the mask used to filter data types from the ASN1 tag, excluding the class bits.
const tagMask = 0x1f

// SNMP data type tags.
const (
	ipTag                = 0x40
	resolvedIPTag        = ipTag & tagMask
	counter32Tag         = 0x41
	resolvedCounter32Tag = counter32Tag & tagMask
	gauge32Tag           = 0x42
	resolvedGauge32Tag   = gauge32Tag & tagMask
	timeTag              = 0x43
	resolvedTimeTag      = timeTag & tagMask
	opaqueTag            = 0x44
	resolvedOpaqueTag    = opaqueTag & tagMask
	counter64Tag         = 0x46
	resolvedCounter64Tag = counter64Tag & tagMask

	endOfMibTag               = 0x82
	resolvedEndOfMibTag       = endOfMibTag & tagMask
	noSuchObjectTag           = 0x80
	resolvedNoSuchObjectTag   = noSuchObjectTag & tagMask
	noSuchInstanceTag         = 0x81
	resolvedNoSuchInstanceTag = noSuchInstanceTag & tagMask
)

// DataType is used to define the different types of variable found in variable bindings.
type DataType int

const (
	Integer DataType = iota
	OctetString
	OID

	IPAdddress
	Time
	Counter32
	Counter64
	Gauge32
	Opaque

	EndOfMib
	NoSuchObject
	NoSuchInstance
)

// Unmarshals an asn1 RawValue contqining a single variable to deliver a TypedValue that encapsulates the variable type and
// the golang representation of the variable value.
//nolint: gocyclo
func unmarshalVariable(raw *asn1.RawValue) (*TypedValue, error) {
	switch raw.Class {
	case asn1.ClassUniversal:
		switch raw.Tag {
		case asn1.TagInteger:
			return unmarshalInteger(raw, Integer)
		case asn1.TagOctetString:
			return unmarshalOctetString(raw, OctetString)
		case asn1.TagOID:
			return unmarshalOID(raw)
		}

	case asn1.ClassApplication:
		switch raw.Tag {
		case resolvedIPTag:
			return unmarshalOctetString(raw, IPAdddress)
		case resolvedCounter32Tag:
			return unmarshalInteger(raw, Counter32)
		case resolvedCounter64Tag:
			return unmarshalInteger(raw, Counter64)
		case resolvedGauge32Tag:
			return unmarshalInteger(raw, Gauge32)
		case resolvedTimeTag:
			return unmarshalInteger(raw, Time)
		case resolvedOpaqueTag:
			return unmarshalOctetString(raw, Opaque)
		}
	case asn1.ClassContextSpecific:
		switch raw.Tag {
		case resolvedEndOfMibTag:
			return &TypedValue{Type: EndOfMib}, nil
		case resolvedNoSuchInstanceTag:
			return &TypedValue{Type: NoSuchInstance}, nil
		case resolvedNoSuchObjectTag:
			return &TypedValue{Type: NoSuchObject}, nil
		}
	}

	return nil, fmt.Errorf("unsupported class %d tag %d", raw.Class, raw.Tag)
}

// Unmarshals an SNMP integer-based variable into a TypedValue.
func unmarshalInteger(raw *asn1.RawValue, dataType DataType) (*TypedValue, error) {
	var value int64
	// Replace SNMP-tag with the generic Integer tag, so ASN1 unmarshalling works.
	raw.FullBytes[0] = asn1.TagInteger
	_, err := ber.Unmarshal(raw.FullBytes, &value)
	if err != nil {
		return nil, err
	}
	return &TypedValue{Type: dataType, Value: integerValue(value, dataType)}, nil
}

// Casts an integer value to the integer type that corresponds to the SNMP data type.
func integerValue(v int64, dataType DataType) interface{} {
	switch dataType { //nolint: exhaustive
	case Counter32, Gauge32, Time:
		return uint32(v)

	case Counter64:
		return uint64(v)
	}
	return v
}

// Unmarshals an SNMP octetstring-based variable into a TypedValue.
func unmarshalOctetString(raw *asn1.RawValue, dataType DataType) (*TypedValue, error) {
	value := &TypedValue{Type: dataType, Value: []byte{}}
	// Replace SNMP-tag with the generic OctetString tag, so ASN1 unmarshalling works.
	raw.FullBytes[0] = asn1.TagOctetString
	_, err := ber.Unmarshal(raw.FullBytes, &value.Value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Unmarshals an OID octetstring-based variable into a TypedValue.
func unmarshalOID(raw *asn1.RawValue) (*TypedValue, error) {
	var value interface{}
	_, err := ber.Unmarshal(raw.FullBytes, &value)
	if err != nil {
		return nil, err
	}
	return &TypedValue{Type: OID, Value: asn1.ObjectIdentifier(value.([]int))}, nil
}

// Encapsulates the data type and value of a variable received in a variable binding from an agent.
type TypedValue struct {
	Type  DataType
	Value interface{}
}

// Delivers value of a typed value as a string.
func (tv *TypedValue) String() string {
	switch tv.Type {
	case Integer:
		return strconv.FormatInt(tv.Value.(int64), 10)
	case OctetString:
		return string(tv.Value.([]uint8))
	case OID:
		return tv.Value.(asn1.ObjectIdentifier).String()
	case Time:
		t := int64(tv.Value.(uint32)) * 10000
		return time.Duration(t).String()
	case Counter32, Gauge32:
		return strconv.FormatInt(int64(tv.Value.(uint32)), 10)
	case Counter64:
		return strconv.FormatInt(int64(tv.Value.(uint64)), 10)
	case IPAdddress:
		address := tv.Value.([]uint8)
		str := make([]string, len(address))
		for x, octet := range address {
			str[x] = strconv.Itoa(int(octet))
		}
		return strings.Join(str, ".")
	case Opaque:
		return hex.EncodeToString(tv.Value.([]uint8))

	case EndOfMib:
		return "End of Mib"
	case NoSuchObject:
		return "No such Object"
	case NoSuchInstance:
		return "No such Instance"
	}
	return fmt.Sprintf("unrecognised data type %d", tv.Type)
}

// Delivers value of a typed value as an ObjectIdentifier.
// Value type must be OID!
func (tv *TypedValue) OID() asn1.ObjectIdentifier {
	return tv.Value.(asn1.ObjectIdentifier)
}

// Delivers value of a typed value as an int.
// Value type must be integer-based.
func (tv *TypedValue) Int() int {
	switch tv.Type { //nolint: exhaustive
	case Integer:
		return int(tv.Value.(int64))
	case Counter64:
		return int(tv.Value.(uint64))
	case Counter32, Gauge32, Time:
		return int(tv.Value.(uint32))
	}
	panic(fmt.Errorf("non-integer data type %d", tv.Type))
}
