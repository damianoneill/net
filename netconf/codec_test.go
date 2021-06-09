package netconf

import (
	"errors"
	"testing"

	mocks "github.com/damianoneill/net/netconf/mocks"
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
	enc := newEncoder(mockt)
	err := enc.encode(&testStr{})
	assert.Error(t, err, "Expect failure")

	// Failure on write of message delimiter
	mockt = &mocks.Transport{}
	mockt.On("Write", mock.Anything).Return(func(buf []byte) int {
		return len(buf)
	}, nil).Once()
	mockt.On("Write", mock.Anything).Return(0, errors.New("Failed"))
	enc = newEncoder(mockt)
	err = enc.encode(&testStr{})
	assert.Error(t, err, "Expect failure")
}
