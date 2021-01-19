package codec

import (
	"encoding/xml"
	"io"

	"github.com/damianoneill/net/v2/netconf/common/codec/rfc6242"
)

// Decoder wraps the standard xml Codec (for XML decoding)
// and RFC6242-compliant Codec (for netconf message framing)
type Decoder struct {
	*xml.Decoder
	ncDecoder *rfc6242.Decoder
}

// Encoder wraps the standard xml Codec (for XML encoding)
// and RFC6242-compliant Codec (for netconf message framing)
type Encoder struct {
	xmlEncoder *xml.Encoder
	ncEncoder  *rfc6242.Encoder
}

// Encode encodes netconf message.
func (e *Encoder) Encode(msg interface{}) error {
	// Prepend xml document declaration to each message.
	_, err := e.ncEncoder.Write([]byte(xml.Header))
	if err != nil {
		return err
	}

	err = e.xmlEncoder.Encode(msg)
	if err != nil {
		return err
	}
	return e.ncEncoder.EndOfMessage()
}

// NewDecoder delivers a new decoder.
func NewDecoder(t io.Reader) *Decoder {
	ncDecoder := rfc6242.NewDecoder(t)
	return &Decoder{Decoder: xml.NewDecoder(ncDecoder), ncDecoder: ncDecoder}
}

// NewEncoder delivers a new encoder.
func NewEncoder(t io.Writer) *Encoder {
	ncEncoder := rfc6242.NewEncoder(t)
	return &Encoder{xmlEncoder: xml.NewEncoder(ncEncoder), ncEncoder: ncEncoder}
}

// EnableChunkedFraming enables chunked framing on the specified decoder and encoder.
func EnableChunkedFraming(d *Decoder, e *Encoder) {
	rfc6242.SetChunkedFraming(d.ncDecoder, e.ncEncoder)
}
