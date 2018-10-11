package netconf

import (
	"encoding/xml"

	"github.com/damianoneill/net/netconf/rfc6242"
)

// Define encoder and decoder that wrap the standard xml Codec (for XML en/decoding)
// and RFC6242-compliant Codec (for netconf message en/decoding)

type decoder struct {
	*xml.Decoder
	ncDecoder *rfc6242.Decoder
}

type encoder struct {
	xmlEncoder *xml.Encoder
	ncEncoder  *rfc6242.Encoder
}

func (e *encoder) encode(msg interface{}) error {

	err := e.xmlEncoder.Encode(msg)
	if err != nil {
		return err
	}
	err = e.ncEncoder.EndOfMessage()
	if err != nil {
		return err
	}
	return nil
}

func newDecoder(t Transport) *decoder {
	ncDecoder := rfc6242.NewDecoder(t)
	return &decoder{Decoder: xml.NewDecoder(ncDecoder), ncDecoder: ncDecoder}
}

func newEncoder(t Transport) *encoder {
	ncEncoder := rfc6242.NewEncoder(t)
	return &encoder{xmlEncoder: xml.NewEncoder(ncEncoder), ncEncoder: ncEncoder}
}
