package codec

import (
	"errors"
	"testing"

	"github.com/damianoneill/net/netconf/mocks"
	"github.com/stretchr/testify/mock"
	assert "github.com/stretchr/testify/require"
)

type testStr struct {
	Field string
}

func TestEncoderFailures(t *testing.T) {

	// Failure on write of message
	mockt := &mocks.Transport{}
	mockt.On("Write", mock.Anything).Return(0, errors.New("Failed"))
	enc := NewEncoder(mockt)
	err := enc.Encode(&testStr{})
	assert.Error(t, err, "Expect failure")

	// Failure on write of message delimiter
	mockt = &mocks.Transport{}
	mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		return len(buf)
	}, nil).Once()
	mockt.On("Write", mock.Anything).Return(0, errors.New("Failed"))
	enc = NewEncoder(mockt)
	err = enc.Encode(&testStr{})
	assert.Error(t, err, "Expect failure")
}

func TestEnableChunkedFraming(t *testing.T) {

	enc := NewEncoder(nil)
	dec := NewDecoder(nil)

	assert.False(t, enc.ncEncoder.ChunkedFraming)

	EnableChunkedFraming(dec, enc)

	assert.True(t, enc.ncEncoder.ChunkedFraming)
}
