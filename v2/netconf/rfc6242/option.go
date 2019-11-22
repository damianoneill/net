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

const (
	// DecoderMinScannerBufferSize is the scanner buffer size floor.
	DecoderMinScannerBufferSize = 20
)

// DecoderOption is a constructor option function for the Decoder type.
type DecoderOption func(*Decoder)

// EncoderOption is a consturctor option function for the Encoder type.
type EncoderOption func(*Encoder)

// WithScannerBufferSize configures the buffer size of the
// bufio.Scanner used by the decoder to scan input tokens.  If bytes
// is smaller than the constant DecoderMinScannerBufferSize, the
// buffer size will be set to DecoderMinScannerBufferSize.
func WithScannerBufferSize(bytes int) DecoderOption {
	return func(d *Decoder) {
		if bytes < DecoderMinScannerBufferSize {
			bytes = DecoderMinScannerBufferSize
		}
		d.bufSize = bytes
	}
}

// WithFramer sets the Decoder's initial Framer.
func WithFramer(f FramerFn) DecoderOption { return func(d *Decoder) { d.framer = f } }

// WithMaximumChunkSize sets an upper bound on the chunk size used
// when writing data to an Encoder. If 0 is passed, the upper bound
// reverts to the maximum chunk size permitted by RFC6242.
func WithMaximumChunkSize(size uint32) EncoderOption {
	return func(e *Encoder) {
		if size < 1 {
			size = rfc6242maximumAllowedChunkSize
		}
		e.MaxChunkSize = size
	}
}
