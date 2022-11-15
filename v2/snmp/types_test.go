package snmp

import (
	"encoding/asn1"
	"reflect"
	"testing"

	assert "github.com/stretchr/testify/require"
)

//nolint:funlen
func TestUnmarshalVariable(t *testing.T) {
	tests := []struct {
		name      string
		input     *asn1.RawValue
		wantType  DataType
		wantValue interface{}
		wantErr   bool
	}{
		{
			"Integer", &asn1.RawValue{Tag: asn1.TagInteger, FullBytes: []byte{asn1.TagInteger, 1, 0x05}},
			Integer, int64(5), false,
		},
		{
			"OctestString", &asn1.RawValue{Tag: asn1.TagOctetString, FullBytes: []byte{asn1.TagOctetString, 3, 0x01, 0x02, 0x03}},
			OctetString,
			[]byte{1, 2, 3},
			false,
		},
		{
			"OID", &asn1.RawValue{Tag: asn1.TagOID, FullBytes: []byte{asn1.TagOID, 2, 0x2b, 0x0a}},
			OID,
			asn1.ObjectIdentifier{1, 3, 10},
			false,
		},
		{
			"IpAddress", &asn1.RawValue{Tag: resolvedIPTag, Class: asn1.ClassApplication, FullBytes: []byte{ipTag, 4, 10, 11, 12, 13}},
			IPAdddress,
			[]uint8{10, 11, 12, 13},
			false,
		},
		{
			"Counter32", &asn1.RawValue{
				Tag: resolvedCounter32Tag, Class: asn1.ClassApplication,
				FullBytes: []byte{counter32Tag, 4, 13, 76, 167, 11},
			},
			Counter32, uint32(223127307), false,
		},
		{"Counter64", &asn1.RawValue{
			Tag: resolvedCounter64Tag, Class: asn1.ClassApplication,
			FullBytes: []byte{counter64Tag, 5, 3, 29, 251, 66, 37},
		}, Counter64, uint64(13387907621), false},
		{
			"Gauge32", &asn1.RawValue{Tag: resolvedGauge32Tag, Class: asn1.ClassApplication, FullBytes: []byte{gauge32Tag, 3, 13, 76, 167, 2}},
			Gauge32, uint32(871591), false,
		},
		{
			"Time", &asn1.RawValue{Tag: resolvedTimeTag, Class: asn1.ClassApplication, FullBytes: []byte{timeTag, 5, 0, 138, 103, 191, 17}},
			Time, uint32(2322054929), false,
		},
		{
			"Opaque", &asn1.RawValue{Tag: resolvedOpaqueTag, Class: asn1.ClassApplication, FullBytes: []byte{opaqueTag, 3, 0xFF, 0xFE, 0xFD}},
			Opaque,
			[]byte{0xff, 0xfe, 0xfd},
			false,
		},
		{
			"EndOfMib", &asn1.RawValue{Tag: resolvedEndOfMibTag, Class: asn1.ClassContextSpecific, FullBytes: []byte{endOfMibTag, 0}},
			EndOfMib, nil, false,
		},
		{
			"NoSuchObject", &asn1.RawValue{Tag: resolvedNoSuchObjectTag, Class: asn1.ClassContextSpecific, FullBytes: []byte{noSuchObjectTag, 0}},
			NoSuchObject, nil, false,
		},
		{"NoSuchInstance", &asn1.RawValue{
			Tag: resolvedNoSuchInstanceTag, Class: asn1.ClassContextSpecific,
			FullBytes: []byte{noSuchInstanceTag, 0},
		}, NoSuchInstance, nil, false},
		{
			"Unknown", &asn1.RawValue{Tag: 0xff, Class: 0xff, FullBytes: []byte{opaqueTag, 3, 0xFF, 0xFE, 0xFD}},
			Opaque, nil, true,
		},
		{
			"InvalidString", &asn1.RawValue{Tag: asn1.TagOctetString, FullBytes: []byte{asn1.TagOctetString, 0xFF, 0x01, 0x02, 0x03}},
			OctetString, nil, true,
		},
		{
			"InvalidInteger", &asn1.RawValue{Tag: asn1.TagInteger, FullBytes: []byte{asn1.TagInteger, 0xFF, 0x01, 0x02, 0x03}},
			Integer, nil, true,
		},
		{
			"InvalidOID", &asn1.RawValue{Tag: asn1.TagOID, FullBytes: []byte{asn1.TagOID, 0xFF, 0x01, 0x02, 0x03}},
			OID, nil, true,
		},
	}
	//nolint: scopelint
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv, err := unmarshalVariable(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("unmarshalVariable error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				return
			}
			if tv.Type != tt.wantType {
				t.Errorf("unmarshalVariable type = %v, want %v", tv.Type, tt.wantType)
			}
			if !reflect.DeepEqual(tv.Value, tt.wantValue) {
				t.Errorf("unmarshalVariable value = %v, want %v", tv.Value, tt.wantValue)
			}
		})
	}
}

func TestTypedVariableStringRepresentation(t *testing.T) {
	tests := []struct {
		name       string
		input      *TypedValue
		wantString string
	}{
		{"Integer", &TypedValue{Integer, int64(17171)}, "17171"},
		{"OctetString", &TypedValue{OctetString, []uint8{0x61, 0x62, 0x63}}, "abc"},
		{"OID", &TypedValue{OID, asn1.ObjectIdentifier{1, 3, 10}}, "1.3.10"},
		{"IpAddress", &TypedValue{IPAdddress, []uint8{0x0a, 0x12, 0x55, 0x27}}, "10.18.85.39"},
		{"Counter64", &TypedValue{Counter64, uint64(91919111919)}, "91919111919"},
		{"Counter32", &TypedValue{Counter32, uint32(29292)}, "29292"},
		{"Time", &TypedValue{Time, uint32(18532)}, "185.32ms"},
		{"Opaque", &TypedValue{Opaque, []uint8{0x01, 0xFF, 0xFE}}, "01fffe"},
		{"EndOfMib", &TypedValue{EndOfMib, nil}, "End of Mib"},
		{"NoSuchObject", &TypedValue{NoSuchObject, nil}, "No such Object"},
		{"NoSuchInstance", &TypedValue{NoSuchInstance, nil}, "No such Instance"},
		{"InvalidType", &TypedValue{9999, nil}, "unrecognised data type 9999"},
	}
	//nolint: scopelint
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()

			if result != tt.wantString {
				t.Errorf("String type = %s, want %s got %s", tt.name, tt.wantString, result)
			}
		})
	}
}

func TestTypedVariableIntegerRepresentation(t *testing.T) {
	tests := []struct {
		name  string
		input *TypedValue
		want  int
	}{
		{"Integer", &TypedValue{Integer, int64(17171)}, 17171},
		{"Counter64", &TypedValue{Counter64, uint64(91919111919)}, 91919111919},
		{"Counter32", &TypedValue{Counter32, uint32(29292)}, 29292},
		{"Gauge32", &TypedValue{Gauge32, uint32(2020)}, 2020},
		{"Time", &TypedValue{Time, uint32(18532)}, 18532},
	}
	//nolint: scopelint
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Int()

			if result != tt.want {
				t.Errorf("Integer type = %s, want %d got %d", tt.name, tt.want, result)
			}
		})
	}

	assert.Panics(t, func() { (&TypedValue{Type: OctetString}).Int() }, "should panic with non-integer type")
}

func TestTypedVariableOIDRepresentation(t *testing.T) {
	assert.Equal(t, (&TypedValue{OID, asn1.ObjectIdentifier{1, 3, 500, 5}}).OID(), asn1.ObjectIdentifier{1, 3, 500, 5})
}
