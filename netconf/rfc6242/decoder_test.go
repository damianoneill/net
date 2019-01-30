package rfc6242

import (
	"io"
	"strings"
	"testing"
)

var EOM = string(tokenEOM)

func TestEOMDecoding(t *testing.T) {

	type decresp struct {
		inputs      []string
		buffer string
		err    error
	}

	tests := []struct {
		name      string
		buflen    int
		responses []decresp
	}{
		{"MessageWithEOM", 100,
			[]decresp{
				{[]string{"123456_abcde" + EOM},"123456_abcde", nil},
				{[]string{"XYZ1" + EOM} ,"XYZ1", nil},
				{nil, "", io.EOF},
			},
		},
		{"SeparatePayload_EOM", 100,
			[]decresp{
				{[]string{"123456_abcde", EOM}, "123456_abcde", nil},
				{[]string{"XYZ1", EOM},"XYZ1", nil},
				{nil,"", io.EOF},
			},
		},
		{"MessageSplitOverBuffer", 7,
			[]decresp{
				{ []string{"1234567"},"1234567", nil},
				{[]string{"AB", EOM}, "AB", nil},
				{[]string{"abcdefg"},"abcdefg", nil},
				{[]string{"h", EOM}, "h", nil},
				{nil, "", io.EOF},
			},
		},
		{"InputTooLongForBuffer", 8,

			[]decresp{
				{[]string{"1234567890" + EOM}, "12345678", nil},
				{nil,"90", nil},
			},
		},
		{"PartialEOM", 100,
			[]decresp{
				{ []string{"1234]]>]]XYZ" + EOM}, "1234]]>]]XYZ", nil},
				{nil, "", io.EOF},
			},
		},
		{"SmallWrites", 100,
			[]decresp{
				{ []string{"AB", "CD", "EF"}, "ABCDEF", nil},
				{ []string{"G", EOM}, "G", nil},
				{nil, "", io.EOF},
			},
		},
		{"MissingEOM", 100,
			[]decresp{
				{[]string{"ABCDEF"}, "ABCDEF", nil},
				{nil,"", io.ErrUnexpectedEOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			transport := newTransport()

			d := NewDecoder(transport.r)

			buffer := make([]byte, tt.buflen)
			for i, resp := range tt.responses {
				transport.Write(resp.inputs, i == len(tt.responses) - 1)

				count, err := d.Read(buffer)
				token := string(buffer[:count])
				if resp.buffer != token {
					t.Errorf("Decoder %s[%d]: buffer mismatch wanted >%s< got >%s<", tt.name, i, resp.buffer, token)
				} else if resp.err != err {
					t.Errorf("Decoder %s[%d]: error mismatch wanted %s got %s", tt.name, i, resp.err, err)
				}
			}

		})
	}
}

func TestFramerTransition(t *testing.T) {

	type decresp struct {
		inputs      []string
		buffer     string
		err        string
		setChunked bool
	}

	tests := []struct {
		name   string
		buflen int

		responses []decresp
	}{
		{"SimpleSwitch", 100,
			[]decresp{
				{[]string{"<hello/>" + EOM}, "<hello/>", "", true},
				{[]string{"\n#6\n", "<rpc/>", "\n##\n"},"<rpc/>", "", false}, // Multiple writes
				{nil, "", "EOF", false},
			},
		},
		{"SwitchWithDanglingEOM", 100,
			[]decresp{
				{[]string{"<hello/>"}, "<hello/>", "", true},
				{[]string{EOM + "\n#6\n" + "<rpc/>" + "\n##\n"}, "<rpc/>", "", false},  // Single write
				{nil, "", "EOF", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			transport := newTransport()

			d := NewDecoder(transport.r)

			buffer := make([]byte, tt.buflen)
			for i, resp := range tt.responses {

				transport.Write(resp.inputs, i == len(tt.responses) - 1)

				count, err := d.Read(buffer)
				token := string(buffer[:count])
				if resp.buffer != token {
					t.Errorf("Decoder %s[%d]: buffer mismatch wanted >%s< got >%s<", tt.name, i, resp.buffer, token)
				} else if err == nil && resp.err != "" ||
					      err != nil && !strings.Contains(err.Error(), resp.err) {
					t.Errorf("Decoder %s[%d]: error mismatch wanted %s got %s", tt.name, i, resp.err, err)
				}
				if resp.setChunked {
					SetChunkedFraming(d)
				}
			}
		})
	}
}

func TestChunkedFramer(t *testing.T) {

	type decresp struct {
		inputs      []string
		buffer     string
		err        string
		setChunked bool
	}

	tests := []struct {
		name   string
		buflen int

		responses []decresp
	}{
		{"SplitChunkMetadataLength", 100,
			[]decresp{
				{[]string{"\n#6", "\n" + "<rpc/>" + "\n#", "#\n"}, "<rpc/>", "", false},
				{nil, "", "EOF", false},
			},
		},
		{"SplitEndOfChunks", 100,
			[]decresp{
				{[]string{"\n#6", "\n" + "<rpc/>" + "\n##", "\n"}, "<rpc/>", "", false},
			},
		},
		{"EndOfChunksWithoutChunks", 100,
			[]decresp{
				{[]string{"\n##\n"}, "", "", false},
			},
		},
		{"InvalidChunkHeader", 100,
			[]decresp{
				{[]string{"\n#A"}, "", "", false},  // Single write
				{nil, "", "invalid chunk header", false},
			},
		},
		{"ChunkHeaderNotStartingWithNewline1", 100,
			[]decresp{
				{[]string{"X"}, "", "", false},  // Single write
				{nil, "", "invalid chunk header", false},
			},
		},
		{"ChunkHeaderNotStartingWithNewline2", 100,
			[]decresp{
				{[]string{"12345678"}, "", "", false},  // Single write
				{nil, "", "invalid chunk header", false},
			},
		},
		{"ChunkHeaderNotStartingWithNewline3", 100,
			[]decresp{
				{[]string{"123456789"}, "", "", false},  // Single write
				{nil, "", "invalid chunk header", false},
			},
		},
		{"ChunkHeaderNotStartingWithNewlineHash", 100,
			[]decresp{
				{[]string{"\nX"}, "", "", false},  // Single write
				{nil, "", "invalid chunk header", false},
			},
		},
		{"InvalidChunkSize1", 100,
			[]decresp{
				{[]string{"\n#4294967297", "\n" + "<rpc/>" + "\n#", "#\n"}, "", "", false},  // Single write
				{nil, "", "chunk size larger than maximum", false},
			},
		},
		{"InvalidChunkSize2", 100,
			[]decresp{
				{[]string{"\n#42949672978"}, "", "", false},
				{nil, "", "no valid chunk-size detected", false},
			},
		},
		{"InvalidChunkSize3", 100,
			[]decresp{
				{[]string{"\n#4294967297000\n" + "<rpc/>" + "\n#", "#\n"}, "", "", false},  // Single write
				{nil, "", "token too long", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			transport := newTransport()

			d := NewDecoder(transport.r, WithFramer(decoderChunked), WithScannerBufferSize(0))

			buffer := make([]byte, tt.buflen)
			for i, resp := range tt.responses {

				transport.Write(resp.inputs, i == len(tt.responses) - 1)

				count, err := d.Read(buffer)
				token := string(buffer[:count])
				if resp.buffer != token {
					t.Errorf("Decoder %s[%d]: buffer mismatch wanted >%s< got >%s<", tt.name, i, resp.buffer, token)
				} else if err == nil && resp.err != "" ||
					err != nil && !strings.Contains(err.Error(), resp.err) {
					t.Errorf("Decoder %s[%d]: error mismatch wanted %s got %s", tt.name, i, resp.err, err)
				}
				if resp.setChunked {
					SetChunkedFraming(d)
				}
			}
		})
	}
}

func newTransport() (*transport) {
	pr, pw := io.Pipe()
	t := &transport{r: pr, w: pw, ch: make(chan string, 5)}
	go func() {
		for s := range t.ch {
			t.w.Write([]byte(s))
		}
		t.w.Close()
	}()
	return t
}

type transport struct {
	r io.Reader
	w io.WriteCloser
	ch chan string
}

func (t *transport) Write(inputs []string, shouldClose bool) {

	if inputs == nil {
		close(t.ch)
	} else {
		for _, s := range inputs {
			t.ch <- s
		}
		if shouldClose {
			close(t.ch)
		}
	}
}

