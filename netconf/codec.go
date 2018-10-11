package netconf

import (
	"encoding/xml"
	"io"

	"github.com/damianoneill/net/netconf/rfc6242"
)

// Define encoder and decoder that wrap the standard xml Codec (for XML en/decoding)
// and RFC6242-compliant Codec (for netconf message framing)

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
	return e.ncEncoder.EndOfMessage()
}

func newDecoder(t io.Reader) *decoder {
	ncDecoder := rfc6242.NewDecoder(t)
	return &decoder{Decoder: xml.NewDecoder(ncDecoder), ncDecoder: ncDecoder}
}

func newEncoder(t io.Writer) *encoder {
	ncEncoder := rfc6242.NewEncoder(t)
	return &encoder{xmlEncoder: xml.NewEncoder(ncEncoder), ncEncoder: ncEncoder}
}

func enableChunkedFraming(d *decoder, e *encoder) {
	rfc6242.SetChunkedFraming(d.ncDecoder, e.ncEncoder)
}
