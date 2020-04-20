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

	getRequest := []byte{
		// Message Type = Sequence, Length = 38
		0x30, 0x26,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetRequest, Length = 25
		0xa0, 0x19,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 14
		0x30, 0x0e,
		// Varbind Type = Sequence, Length = 12
		0x30, 0x0c,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.5.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x05, 0x00,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 54
		0x30, 0x82, 0x00, 0x36,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 39
		0xa2, 0x82, 0x00, 0x27,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 26
		0x30, 0x82, 0x00, 0x1a,
		// Varbind Type = Sequence, Length = 22
		0x30, 0x82, 0x00, 0x16,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.5.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x05, 0x00,
		// Value Type = Octet String, Length = 10, Value = cisco-7513
		0x04, 0x0a, 0x63, 0x69, 0x73, 0x63, 0x6f, 0x2d, 0x37, 0x35, 0x31, 0x33,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(40, nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = NoOpLoggingHooks
	m := &managerImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.1.5.0"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	value := pdu.VarbindList[0].Value
	assert.Equal(t, "cisco-7513", string(value.([]uint8)))
}

func TestGetNext(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 40
		0x30, 0x28,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 27
		0xa1, 0x1b,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 16
		0x30, 0x10,
		// Varbind Type = Sequence, Length = 14
		0x30, 0x0e,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.2.1
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02, 0x01,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 63
		0x30, 0x82, 0x00, 0x3f,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 48
		0xa2, 0x82, 0x00, 0x30,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 35
		0x30, 0x82, 0x00, 0x23,
		// Varbind Type = Sequence, Length = 31
		0x30, 0x82, 0x00, 0x1f,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.2.2
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02, 0x02,
		// Value Type = Octet String, Length = 17, Value = FastEthernet1/0/0
		0x04, 0x11, 0x46, 0x61, 0x73, 0x74, 0x45, 0x74, 0x68, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x31, 0x2f, 0x30, 0x2f, 0x30,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(40, nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = DiagnosticLoggingHooks
	m := &managerImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.GetNext(context.Background(), []string{"1.3.6.1.2.1.2.2.1.2.1"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	oid := pdu.VarbindList[0].OID
	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.2", oid.String())
	value := pdu.VarbindList[0].Value
	assert.Equal(t, "FastEthernet1/0/0", string(value.([]uint8)))
}

func TestNoOpImplementations(t *testing.T) {
	m, err := NewFactory().NewManager(context.Background(), "localhost:161")
	assert.NoError(t, err)

	r, err := m.GetBulk(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"}, 5, 10)
	assert.Nil(t, r)
	assert.Nil(t, err)

	walker := func(p *PDU) error {
		return nil
	}
	err = m.GetWalk(context.Background(), "1.3.6.1.2.1.2.2.1.1", walker)
	assert.Nil(t, err)
}

// Tests against real SNMP agent. Useful for diagnostics.
//
//func TestRealGet(t *testing.T) {
//
//	m, err := NewFactory().NewManager(context.Background(), "snmp.live.gambitcommunications.com:161")
//	assert.NoError(t, err)
//
//	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.7.0"})
//
//	assert.NoError(t, err)
//	assert.NotNil(t, pdu)
//	assert.Len(t, pdu.VarbindList, 2)
//	value1 := pdu.VarbindList[0].Value
//	assert.Equal(t, "cisco-7513", string(value1.([]uint8)))
//	value2 := pdu.VarbindList[1].Value
//	assert.Equal(t, int64(78), value2.(int64))
//}
//
//func TestRealGetNext(t *testing.T) {
//
//	m, err := NewFactory().NewManager(context.Background(), "snmp.live.gambitcommunications.com:161",
//		LoggingHooks(DiagnosticLoggingHooks))
//	assert.NoError(t, err)
//
//	pdu, err := m.GetNext(context.Background(), []string{"1.3.6.1.2.1.2.2.1.2.1"})
//
//	assert.NoError(t, err)
//	assert.NotNil(t, pdu)
//	assert.Len(t, pdu.VarbindList, 1)
//	oid := pdu.VarbindList[0].OID
//	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.2", oid.String())
//	value := pdu.VarbindList[0].Value
//	assert.Equal(t, "FastEthernet1/0/0", string(value.([]uint8)))
//}
