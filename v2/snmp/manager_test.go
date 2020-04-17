package snmp

import (
	"context"
	"testing"

	"github.com/damianoneill/net/v2/snmp/mocks"
	"github.com/golang/mock/gomock"

	assert "github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	// Based on example at https://www.ranecommercial.com/legacy/pdf/ranenotes/SNMP_Simple_Network_Management_Protocol.pdf
	getMessage := []byte{
		// Message Type = Sequence, Length = 44
		0x30, 0x2c,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetRequest, Length = 30
		0xa0, 0x1e,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 19
		0x30, 0x13,
		// Varbind Type = Sequence, Length = 17
		0x30, 0x11,
		// Object Identifier Type = Object Identifier, Length = 13, Value = 1.3.6.1.4.1.2680.1.2.7.3.2.0
		0x06, 0x0d, 0x2b, 0x06, 0x01, 0x04, 0x01, 0x94, 0x78, 0x01, 0x02, 0x07, 0x03, 0x02, 0x00,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	gomock.InOrder(
		mockConn.EXPECT().Write(getMessage).Return(44, nil),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "private"
	m := &managerImpl{config: &config, conn: mockConn}

	r, err := m.Get(context.Background(), []string{"1.3.6.1.4.1.2680.1.2.7.3.2.0"})
	assert.Nil(t, r)
	assert.Nil(t, err)

}

func TestNoOpImplementations(t *testing.T) {
	m, err := NewFactory().NewManager(context.Background(), "localhost:161")
	assert.NoError(t, err)

	r, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"})
	assert.Nil(t, r)
	assert.Nil(t, err)

	r, err = m.GetNext(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"})
	assert.Nil(t, r)
	assert.Nil(t, err)

	r, err = m.GetBulk(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"}, 5, 10)
	assert.Nil(t, r)
	assert.Nil(t, err)

	walker := func(p *PDU) error {
		return nil
	}
	err = m.GetWalk(context.Background(), "1.3.6.1.2.1.2.2.1.1", walker)
	assert.Nil(t, err)
}
