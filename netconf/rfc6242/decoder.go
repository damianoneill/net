// Copyright 2018 Andrew Fort
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package rfc6242

import (
	"bufio"
	"io"

	"github.com/pkg/errors"
)

// FramerFn is the input tokenization function used by a Decoder.
type FramerFn func(d *Decoder, data []byte, atEOF bool) (advance int, token []byte, err error)

// Decoder is an RFC6242 transport framing decoder filter.
//
// Decoder operates as an inline filter, taking a io.Reader as input
// and providing io.Reader as well as the low-overhead io.WriterTo.
//
// When used with io.Copy, the io.Reader interface is expected of the
// source argument, but if the value passed also supports io.WriterTo
// (as does Decoder), the WriteTo function will instead be called with
// the copy destination as its writer argument.
//
// Decoder is not safe for concurrent use.
type Decoder struct {
	// Input is the input source for the Decoder. The input stream
	// must consist of RFC6242 encoded data according to the current
	// Framer.
	Input io.Reader

	framer FramerFn
	// Pending framer will take effect after end of message has been processed.
	pendingFramer FramerFn

	s  *bufio.Scanner
	pr *io.PipeReader
	pw *io.PipeWriter

	// Defines the number of bytes still to be read from the pipe reader.
	pipedCount int

	scanErr       error
	chunkDataLeft uint64 // state
	bufSize       int    // config
	anySeen       bool
	eofOK         bool
}

// NewDecoder creates a new RFC6242 transport framing decoder reading from
// input, configured with any options provided.
func NewDecoder(input io.Reader, options ...DecoderOption) *Decoder {
	d := &Decoder{
		Input:   input,
		framer:  decoderEndOfMessage,
		bufSize: defaultReaderBufferSize,
		// Added this setting of eofOK to true, to avoid 'unexpected EOF' failure (vs. standard EOF) being
		// reported when stream is closed before any data is received.
		eofOK: true,
	}
	for _, option := range options {
		option(d)
	}
	d.pr, d.pw = io.Pipe()
	if d.s == nil {
		d.s = bufio.NewScanner(input)
		tmp := make([]byte, d.bufSize)
		d.s.Buffer(tmp, d.bufSize)
	}
	d.s.Split(d.split)
	return d
}

// Read reads from the Decoder's input and copies the data into b,
// implementing io.Reader.
func (d *Decoder) Read(b []byte) (n int, err error) {
	// Deliver pending data from pipe, if there is any.
	if d.pipedCount > 0 {
		n, err = d.pr.Read(b)
		d.pipedCount -= n
	} else if d.s.Scan() {
		token := d.s.Bytes()
		if len(token) <= len(b) {
			copy(b, token)
			return len(token), nil
		}
		d.pipedCount = len(token)
		go func() {
			if _, err = d.pw.Write(token); err != nil {
				d.pr.CloseWithError(err) // nolint : gosec
			}
		}()
		n, err = d.pr.Read(b)
		d.pipedCount -= n
	} else if err = d.s.Err(); err == nil {
		if d.eofOK {
			err = io.EOF
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	return
}

// WriteTo reads from the Decoder's input, strips the transport
// encoding and writes the decoded data to w, implementing
// io.WriterTo.
func (d *Decoder) WriteTo(w io.Writer) (n int64, err error) {
	for err == nil && d.s.Scan() {
		b := d.s.Bytes()
		_, err = w.Write(b)
		n += int64(len(b))
	}
	if err = d.s.Err(); err == nil && !d.eofOK {
		err = errors.WithStack(io.ErrUnexpectedEOF)
	}
	return
}

func (d *Decoder) split(b []byte, eof bool) (a int, t []byte, err error) {
	if eof && len(b) == 0 {
		err = d.scanErr
		return
	}
	return d.framer(d, b, eof)
}

func (d *Decoder) setFramer(f FramerFn) {

	// If we have not yet seen an End of Message, set the new framer as pending, so that it only
	// takes effect after End of Message is detected.
	// This allows for the sequence:
	// - transport reader delivers complete hello message, i.e. <hello>....</hello>
	// - decoder delivers token (the hello message) to xml decoder
	// - xml decoder delivers decoded hello to application code
	// - application code inspects hello, enables chunked framing and calls the xml decoder
	// - transport reader delivers 'missing' end of message
	if !d.anySeen {
		d.pendingFramer = f
	} else {
		d.framer = f
	}
}

const (
	// RFC6242 section 4.2 defines the "maximum allowed chunk-size".
	rfc6242maximumAllowedChunkSize = 4294967295
	// the length of `rfc6242maximumAllowedChunkSize` in bytes on the wire.
	rfc6242maximumAllowedChunkSizeLength = 10
	// defaultReaderBufferSize is the default read buffer capacity size.
	defaultReaderBufferSize = 65536
)
