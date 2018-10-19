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

// SetChunkedFraming enables chunked framing mode on any non-nil
// *Decoder and *Encoder objects passed to it.
func SetChunkedFraming(objects ...interface{}) {
	for _, obj := range objects {
		switch obj := obj.(type) {
		case *Decoder:
			if obj != nil {
				obj.setFramer(decoderChunked)
			}
		case *Encoder:
			if obj != nil {
				obj.ChunkedFraming = true
			}
		}
	}
}

// ClearChunkedFraming disables chunked framing mode on any non-nil
// *Decoder and *Encoder objects passed to it.
func ClearChunkedFraming(objects ...interface{}) {
	for _, obj := range objects {
		switch obj := obj.(type) {
		case *Decoder:
			if obj != nil {
				obj.framer = decoderEndOfMessage
			}
		case *Encoder:
			if obj != nil {
				obj.ChunkedFraming = false
			}
		}
	}
}
