package rfc6242

import (
	"bytes"
	"testing"
)

func TestEOMEncoding(t *testing.T) {

	tests := []struct {
		name   string
		inputs []string
		eom    bool
		expect string
	}{
		{"SimpleMessagePart", []string{"ABC"}, false, "ABC"},
		{"MultiPartMessage", []string{"ABC", "XYZ"}, false, "ABCXYZ"},
		{"TerminatedMessage", []string{"ABC", "XYZ"}, true, "ABCXYZ" + EOM},
		{"EmptyMessage", []string{""}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			buf := bytes.NewBuffer([]byte{})
			e := NewEncoder(buf)

			for _, i := range tt.inputs {
				_, _ = e.Write([]byte(i))
			}
			if tt.eom {
				_ = e.EndOfMessage()
			}

			result := buf.String()
			if tt.expect != result {
				t.Errorf("Encoder %s: buffer mismatch wanted >%s< got >%s<", tt.name, tt.expect, result)
			}

			e.Close()
		})
	}
}

func TestChunkedEncoding(t *testing.T) {
	tests := []struct {
		name    string
		chunksz uint32
		inputs  []string
		eom     bool
		expect  string
	}{
		{"SimpleMessagePart", 0, []string{"ABC"}, false, "\n#3\nABC"},
		{"SimpleTerminatedMessage", 0, []string{"ABC"}, true, "\n#3\n" + "ABC" + "\n##\n"},
		{"ChunkedMessage", 5, []string{"ABCDEFGH"}, true, "\n#5\n" + "ABCDE" + "\n#3\n" + "FGH" + "\n##\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			buf := bytes.NewBuffer([]byte{})
			e := NewEncoder(buf, WithMaximumChunkSize(tt.chunksz))
			SetChunkedFraming(e)

			for _, i := range tt.inputs {
				_, _ = e.Write([]byte(i))
			}
			if tt.eom {
				_ = e.EndOfMessage()
			}

			result := buf.String()
			if tt.expect != result {
				t.Errorf("Encoder %s: buffer mismatch wanted >%s< got >%s<", tt.name, tt.expect, result)
			}
		})
	}
}
