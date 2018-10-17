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
	"io"
	"strconv"
)

// NewEncoder returns a new RFC6242 transport encoding writer with underlying
// writer output, configured with any options provided.
func NewEncoder(output io.Writer, opts ...EncoderOption) *Encoder {
	e := &Encoder{Output: output, MaxChunkSize: rfc6242maximumAllowedChunkSize}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Encoder is a filtering writer. By default it acts as a pass through writer.
// If chunked mode is enabled (see SetChunkedFramingMode), input to Write calls
// is chunked and the RFC6242 chunked encoding output written to the underlying
// writer.
type Encoder struct {
	// Output is the underlying Writer to receive encoded output
	Output io.Writer
	// ChunkedFraming sets whether the next call to Write should use
	// chunked-message framing (true) or end-of-message framing (false)
	ChunkedFraming bool
	// MaxChunkSize is the maximum size of chunks the encoder will Encode. If
	// zero, the Encoder places no artificial ceiling on the chunk size.
	MaxChunkSize uint32
}

// Write writes the framed output for b to the underlying writer
func (e *Encoder) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}
	if e.ChunkedFraming {
		return e.writeChunked(b)
	}
	return e.Output.Write(b)
}

// EndOfMessage must be called after each conceptual message (or XML document) is
// written to the Encoder. It writes the appropriate NETCONF message ending,
// either "]]>]]>" or if chunked framing is enabled, "\n##\n".
func (e *Encoder) EndOfMessage() error {
	var err error
	if e.ChunkedFraming {
		_, err = e.Output.Write([]byte("\n##\n"))
	} else {
		_, err = e.Output.Write(tokenEOM)
	}
	return err
}

// Close attempts to close the underlying writer.
func (e *Encoder) Close() error {
	// always be closing
	if closer, ok := e.Output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (e *Encoder) writeChunked(b []byte) (n int, err error) {
	// encode b, in chunks, to the underlying writer
	for n < len(b) {
		chunksize := len(b) - n
		if uint32(chunksize) > e.MaxChunkSize {
			chunksize = int(e.MaxChunkSize)
		}

		// chunk encoding:
		// \n#<x>\n<x bytes data...>

		// write "\n#", the chunk header start
		_, err = e.Output.Write([]byte("\n#"))
		if err == nil {
			// write the chunk length and a newline to end the chunk header
			_, err = e.Output.Write([]byte(strconv.Itoa(chunksize) + "\n"))
		}
		var wn int
		if err == nil {
			// write the chunk data
			wn, err = e.Output.Write(b[n : n+chunksize])
			// io.Writer requires not returning nil error for short writes,
			// so we do not check for them.
			n += wn
		}
		if err != nil {
			break
		}
	}
	return
}
