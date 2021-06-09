package snmp

import (
	"context"
	"errors"
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
		mockConn.EXPECT().Close().Return(nil),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = NoOpLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}
	defer m.Close()

	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.1.5.0"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, OctetString, tv.Type)
	assert.Equal(t, "cisco-7513", string(tv.Value.([]uint8)))
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
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.GetNext(context.Background(), []string{"1.3.6.1.2.1.2.2.1.2.1"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	oid := pdu.VarbindList[0].OID
	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.2", oid.String())
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, "FastEthernet1/0/0", string(tv.Value.([]uint8)))
}

func TestGetBulk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 53
		0x30, 0x35,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetBulkRequest, Length = 40
		0xa5, 0x28,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Non-Repeaters = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Max Repetitions Type = Integer, Length = 1, Value = 3
		0x02, 0x01, 0x03,
		// Varbind List Type = Sequence, Length = 29
		0x30, 0x1d,
		// Varbind Type = Sequence, Length = 12
		0x30, 0x0c,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.4.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04, 0x00,
		// Value Type = Null, Length = 0
		0x05, 0x00,
		// Varbind Type = Sequence, Length = 13
		0x30, 0x0d,
		// Object Identifier Type = Object Identifier, Length = 9, Value = 1.3.6.1.2.1.2.2.1.2
		0x06, 0x09, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 149
		0x30, 0x82, 0x00, 0x95,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 134
		0xa2, 0x82, 0x00, 0x86,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 121
		0x30, 0x82, 0x00, 0x79,

		// Varbind Type = Sequence, Length = 22
		0x30, 0x82, 0x00, 0x16,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.5.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x05, 0x00,
		// Value Type = Octet String, Length = 10, Value = cisco-7513
		0x04, 0x0a, 0x63, 0x69, 0x73, 0x63, 0x6f, 0x2d, 0x37, 0x35, 0x31, 0x33,

		// Varbind Type = Sequence, Length = 21
		0x30, 0x82, 0x00, 0x15,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.2.1
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02, 0x01,
		// Value Type = Octet String, Length = 7, Value = Fddi0/0
		0x04, 0x07, 0x46, 0x64, 0x64, 0x69, 0x30, 0x2f, 0x30,

		// Varbind Type = Sequence, Length = 31
		0x30, 0x82, 0x00, 0x1f,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.2.2
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02, 0x02,
		// Value Type = Octet String, Length = 17, Value = FastEthernet1/0/0
		0x04, 0x11, 0x46, 0x61, 0x73, 0x74, 0x45, 0x74, 0x68, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x31, 0x2f, 0x30, 0x2f, 0x30,

		// Varbind Type = Sequence, Length = 31
		0x30, 0x82, 0x00, 0x1f,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.2.3
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x02, 0x03,
		// Value Type = Octet String, Length = 17, Value = FastEthernet1/1/0
		0x04, 0x11, 0x46, 0x61, 0x73, 0x74, 0x45, 0x74, 0x68, 0x65, 0x72, 0x6e, 0x65, 0x74, 0x31, 0x2f, 0x31, 0x2f, 0x30,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(55, nil),
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
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.GetBulk(context.Background(), []string{"1.3.6.1.2.1.1.4.0", "1.3.6.1.2.1.2.2.1.2"}, 1, 3)

	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 4)
	vbs := pdu.VarbindList
	assert.Equal(t, "1.3.6.1.2.1.1.5.0", vbs[0].OID.String())
	assert.Equal(t, "cisco-7513", string(vbs[0].TypedValue.Value.([]uint8)))

	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.1", vbs[1].OID.String())
	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.2", vbs[2].OID.String())
	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.3", vbs[3].OID.String())
	assert.Equal(t, "Fddi0/0", string(vbs[1].TypedValue.Value.([]uint8)))
	assert.Equal(t, "FastEthernet1/0/0", string(vbs[2].TypedValue.Value.([]uint8)))
	assert.Equal(t, "FastEthernet1/1/0", string(vbs[3].TypedValue.Value.([]uint8)))
}

func TestWalk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest1 := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 24
		0xa1, 0x18,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 13
		0x30, 0x0d,
		// Varbind Type = Sequence, Length = 11
		0x30, 0x0b,
		// Object Identifier Type = Object Identifier, Length = 7, Value = 1.3.6.1.2.1.1.4
		0x06, 0x07, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getRequest2 := []byte{
		// Message Type = Sequence, Length = 38
		0x30, 0x26,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 25
		0xa1, 0x19,
		// Request ID Type = Integer, Length = 1, Value = 2
		0x02, 0x01, 0x02,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 14
		0x30, 0x0e,
		// Varbind Type = Sequence, Length = 12
		0x30, 0x0c,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.4.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04, 0x00,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse1 := []byte{
		// Message Type = Sequence, Length = 66
		0x30, 0x82, 0x00, 0x42,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 51
		0xa2, 0x82, 0x00, 0x33,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 38
		0x30, 0x82, 0x00, 0x26,
		// Varbind Type = Sequence, Length = 34
		0x30, 0x82, 0x00, 0x22,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.4.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04, 0x00,
		// Value Type = Octet String, Length = 22, Value = support@gambitcomm.com
		0x04, 0x16, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x40, 0x67, 0x61, 0x6d, 0x62, 0x69, 0x74, 0x63, 0x6f, 0x6d, 0x6d, 0x2e, 0x63, 0x6f, 0x6d,
	}

	getResponse2 := []byte{
		// Message Type = Sequence, Length = 54
		0x30, 0x82, 0x00, 0x36,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 39
		0xa2, 0x82, 0x00, 0x27,
		// Request ID Type = Integer, Length = 1, Value = 2
		0x02, 0x01, 0x02,
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
		mockConn.EXPECT().Write(getRequest1).Return(39, nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse1)
				return len(getResponse1), nil
			}),
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest2).Return(40, nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse2)
				return len(getResponse2), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = DiagnosticLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.NoError(t, err)
	assert.Len(t, varbinds, 1)
	assert.Equal(t, "1.3.6.1.2.1.1.4.0", varbinds[0].OID.String())
}

func TestNetworkWriteFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(gomock.Any()).Return(0, errors.New("snmp failure")),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = NoOpLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.EqualError(t, err, "snmp failure")
}

func TestSetDeadlineFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(errors.New("snmp failure")),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = NoOpLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.EqualError(t, err, "snmp failure")
}

func TestNetworkReadFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 24
		0xa1, 0x18,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 13
		0x30, 0x0d,
		// Varbind Type = Sequence, Length = 11
		0x30, 0x0b,
		// Object Identifier Type = Object Identifier, Length = 7, Value = 1.3.6.1.2.1.1.4
		0x06, 0x07, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(39, nil),
		mockConn.EXPECT().Read(gomock.Any()).Return(0, errors.New("snmp failure")),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = DiagnosticLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.EqualError(t, err, "snmp failure")
}

func TestUnmarshalPacketFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 24
		0xa1, 0x18,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 13
		0x30, 0x0d,
		// Varbind Type = Sequence, Length = 11
		0x30, 0x0b,
		// Object Identifier Type = Object Identifier, Length = 7, Value = 1.3.6.1.2.1.1.4
		0x06, 0x07, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Nonsense...
		0xFF, 0xFF, 0xFF,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(len(getRequest), nil),
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
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.Contains(t, err.Error(), "asn1: syntax error:")
}

func TestWalkWalkerFailure(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest1 := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetNextRequest, Length = 24
		0xa1, 0x18,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 13
		0x30, 0x0d,
		// Varbind Type = Sequence, Length = 11
		0x30, 0x0b,
		// Object Identifier Type = Object Identifier, Length = 7, Value = 1.3.6.1.2.1.1.4
		0x06, 0x07, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse1 := []byte{
		// Message Type = Sequence, Length = 66
		0x30, 0x82, 0x00, 0x42,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 51
		0xa2, 0x82, 0x00, 0x33,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 38
		0x30, 0x82, 0x00, 0x26,
		// Varbind Type = Sequence, Length = 34
		0x30, 0x82, 0x00, 0x22,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.4.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04, 0x00,
		// Value Type = Octet String, Length = 22, Value = support@gambitcomm.com
		0x04, 0x16, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x40, 0x67, 0x61, 0x6d, 0x62, 0x69, 0x74, 0x63, 0x6f, 0x6d, 0x6d, 0x2e, 0x63, 0x6f, 0x6d,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest1).Return(39, nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse1)
				return len(getResponse1), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = DiagnosticLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	walker := func(v *Varbind) error {
		return errors.New("walker error")
	}
	err := m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walker)
	assert.EqualError(t, err, "walker error")
}

func TestBulkWalk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetBulkRequest, Length = 24
		0xa5, 0x18,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Non-Repeaters = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Max Repetitions Type = Integer, Length = 1, Value = 2
		0x02, 0x01, 0x02,
		// Varbind List Type = Sequence, Length = 13
		0x30, 0x0d,
		// Varbind Type = Sequence, Length = 11
		0x30, 0x0b,
		// Object Identifier Type = Object Identifier, Length = 7, Value = 1.3.6.1.2.1.1.4
		0x06, 0x07, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 92
		0x30, 0x82, 0x00, 0x5c,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetResponse, Length = 77
		0xa2, 0x82, 0x00, 0x4d,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 64
		0x30, 0x82, 0x00, 0x40,

		// Varbind Type = Sequence, Length = 34
		0x30, 0x82, 0x00, 0x22,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.4.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x04, 0x00,
		// Value Type = Octet String, Length = 22, Value = support@gambitcomm.com
		0x04, 0x16, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x40, 0x67, 0x61, 0x6d, 0x62, 0x69, 0x74, 0x63, 0x6f, 0x6d, 0x6d, 0x2e, 0x63, 0x6f, 0x6d,

		// Varbind Type = Sequence, Length = 22
		0x30, 0x82, 0x00, 0x16,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.5.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x05, 0x00,
		// Value Type = Octet String, Length = 10, Value = cisco-7513
		0x04, 0x0a, 0x63, 0x69, 0x73, 0x63, 0x6f, 0x2d, 0x37, 0x35, 0x31, 0x33,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(len(getRequest), nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "public"
	config.trace = MetricLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	varbinds := []*Varbind{}
	walker := func(v *Varbind) error {
		varbinds = append(varbinds, v)
		return nil
	}

	err := m.BulkWalk(context.Background(), "1.3.6.1.2.1.1.4", 2, walker)

	assert.NoError(t, err)
	assert.Len(t, varbinds, 1)
	assert.Equal(t, "1.3.6.1.2.1.1.4.0", varbinds[0].OID.String())
}

type timeoutError struct{}

func (to *timeoutError) Error() string {
	return "timeout"
}

func (to *timeoutError) Timeout() bool {
	return true
}

func (to *timeoutError) Temporary() bool {
	return false
}

func TestRetry(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest1 := []byte{
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

	getRequest2 := []byte{
		// Message Type = Sequence, Length = 38
		0x30, 0x26,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = GetRequest, Length = 25
		0xa0, 0x19,
		// Request ID Type = Integer, Length = 1, Value = 2
		0x02, 0x01, 0x02,
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
		mockConn.EXPECT().Write(getRequest1).Return(40, nil),
		mockConn.EXPECT().Read(gomock.Any()).Return(0, &timeoutError{}),
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest2).Return(40, nil),
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
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.1.5.0"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, OctetString, tv.Type)
	assert.Equal(t, "cisco-7513", string(tv.Value.([]uint8)))
}

func TestEndOfMib(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 40
		0x30, 0x28,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetNextRequest, Length = 26
		0xa1, 0x1a,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 15
		0x30, 0x0f,
		// Varbind Type = Sequence, Length = 13
		0x30, 0x0d,
		// Object Identifier Type = Object Identifier, Length = 9, Value = 1.3.6.1.6.3.12.1.5.0
		0x06, 0x09, 0x2b, 0x06, 0x01, 0x06, 0x03, 0x0c, 0x01, 0x05, 0x00,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 40
		0x30, 0x28,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetResponse, Length = 26
		0xa2, 0x1a,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 15
		0x30, 0x0f,
		// Varbind Type = Sequence, Length = 13
		0x30, 0x0d,
		// Object Identifier Type = Object Identifier, Length = 9, Value = 1.3.6.1.6.3.12.1.5.0
		0x06, 0x09, 0x2b, 0x06, 0x01, 0x06, 0x03, 0x0c, 0x01, 0x05, 0x00,
		// Value Type = End Of Mib, Length = 0
		0x82, 0x00,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(len(getRequest), nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "private"
	config.trace = DiagnosticLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.GetNext(context.Background(), []string{"1.3.6.1.6.3.12.1.5.0"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	oid := pdu.VarbindList[0].OID
	assert.Equal(t, "1.3.6.1.6.3.12.1.5.0", oid.String())
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, EndOfMib, tv.Type)
	assert.Nil(t, tv.Value)
}

func TestNoSuchObject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetRequest, Length = 23
		0xa0, 0x17,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 12
		0x30, 0x0c,
		// Varbind Type = Sequence, Length = 10
		0x30, 0x0a,
		// Object Identifier Type = Object Identifier, Length = 6, Value = 1.3.6.1.2.1.47
		0x06, 0x06, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x2f,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 37
		0x30, 0x25,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetResponse, Length = 23
		0xa2, 0x17,
		// Request ID Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 12
		0x30, 0x0c,
		// Varbind Type = Sequence, Length = 10
		0x30, 0x0a,
		// Object Identifier Type = Object Identifier, Length = 6, Value = 1.3.6.1.2.1.47
		0x06, 0x06, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x2f,
		// Value Type = NoSuchObject, Length = 0
		0x80, 0x00,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(len(getRequest), nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "private"
	config.trace = NoOpLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.47"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, NoSuchObject, tv.Type)
	assert.Nil(t, tv.Value)
}

func TestNoSuchInstance(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockConn(mockCtrl)

	getRequest := []byte{
		// Message Type = Sequence, Length = 41
		0x30, 0x29,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetRequest, Length = 27
		0xa0, 0x1b,
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
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.1.1
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x01, 0x01,
		// Value Type = Null, Length = 0
		0x05, 0x00,
	}

	getResponse := []byte{
		// Message Type = Sequence, Length = 41
		0x30, 0x29,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 7, Value = private
		0x04, 0x07, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65,
		// PDU Type = GetResponse, Length = 27
		0xa2, 0x1b,
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
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.2.1.2.2.1.1.1
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x01, 0x01,
		// Value Type = NoSuchInstance, Length = 0
		0x81, 0x00,
	}

	gomock.InOrder(
		mockConn.EXPECT().SetDeadline(gomock.Any()).Return(nil),
		mockConn.EXPECT().Write(getRequest).Return(len(getRequest), nil),
		mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(
			func(input []byte) (int, error) {
				copy(input, getResponse)
				return len(getResponse), nil
			}),
	)

	config := defaultConfig
	config.address = "localhost:161"
	config.community = "private"
	config.trace = NoOpLoggingHooks
	m := &sessionImpl{config: &config, conn: mockConn, nextRequestID: 1}

	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1.1"})
	assert.NoError(t, err)
	assert.NotNil(t, pdu)
	assert.Len(t, pdu.VarbindList, 1)
	tv := pdu.VarbindList[0].TypedValue
	assert.Equal(t, NoSuchInstance, tv.Type)
	assert.Nil(t, tv.Value)
}

// Tests against real SNMP agent. Useful for diagnostics.
//
//func TestRealGet(t *testing.T) {
//
//	m, err := NewFactory().NewSession(context.Background(), "snmp.live.gambitcommunications.com:161")
//	assert.NoError(t, err)
//
//	pdu, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.7.0"})
//
//	assert.NoError(t, err)
//	assert.NotNil(t, pdu)
//	assert.Len(t, pdu.VarbindList, 2)
//	value1 := pdu.VarbindList[0].TypedValue.Value
//	assert.Equal(t, "cisco-7513", string(value1.([]uint8)))
//	value2 := pdu.VarbindList[1].TypedValue.Value
//	assert.Equal(t, int64(78), value2.(int64))
//}
//
//func TestRealGetNext(t *testing.T) {
//
//	m, err := NewFactory().NewSession(context.Background(), "snmp.live.gambitcommunications.com:161",
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
//	value := pdu.VarbindList[0].TypedValue.Value
//	assert.Equal(t, "FastEthernet1/0/0", string(value.([]uint8)))
//}
//
//func TestRealGetBulk(t *testing.T) {
//
//	m, err := NewFactory().NewSession(context.Background(), "snmp.live.gambitcommunications.com:161",
//		LoggingHooks(DiagnosticLoggingHooks))
//	assert.NoError(t, err)
//
//	pdu, err := m.GetBulk(context.Background(), []string{"1.3.6.1.2.1.1.4.0", "1.3.6.1.2.1.2.2.1.2"}, 1, 3)
//
//	assert.NoError(t, err)
//	assert.NotNil(t, pdu)
//	assert.Len(t, pdu.VarbindList, 4)
//	vbs := pdu.VarbindList
//	assert.Equal(t, "1.3.6.1.2.1.1.5.0", vbs[0].OID.String())
//	assert.Equal(t, "cisco-7513", string(vbs[0].TypedValue.Value.([]uint8)))
//
//	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.1", vbs[1].OID.String())
//	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.2", vbs[2].OID.String())
//	assert.Equal(t, "1.3.6.1.2.1.2.2.1.2.3", vbs[3].OID.String())
//	assert.Equal(t, "Fddi0/0", string(vbs[1].TypedValue.Value.([]uint8)))
//	assert.Equal(t, "FastEthernet1/0/0", string(vbs[2].TypedValue.Value.([]uint8)))
//	assert.Equal(t, "FastEthernet1/1/0", string(vbs[3].TypedValue.Value.([]uint8)))
//}
//
//func TestRealWalk(t *testing.T) {
//
//	m, err := NewFactory().NewSession(context.Background(), "snmp.live.gambitcommunications.com:161",
//		LoggingHooks(DiagnosticLoggingHooks))
//	assert.NoError(t, err)
//	m.(*sessionImpl).nextRequestID = 1
//
//	walk := func(vb *Varbind) error {
//		fmt.Println("Walk", vb.OID, vb.TypedValue.Type, vb.TypedValue.Value)
//		return nil
//	}
//
//	err = m.Walk(context.Background(), "1.3.6.1.2.1.1.4", walk)
//
//	assert.NoError(t, err)
//}
//
//func TestRealBulkWalk(t *testing.T) {
//
//	m, err := NewFactory().NewSession(context.Background(), "snmp.live.gambitcommunications.com:161",
//		LoggingHooks(DiagnosticLoggingHooks))
//	assert.NoError(t, err)
//	m.(*sessionImpl).nextRequestID = 1
//
//	walk := func(vb *Varbind) error {
//		fmt.Println("Walk", vb.OID, vb.TypedValue.Type, vb.TypedValue.Value)
//		return nil
//	}
//
//	err = m.BulkWalk(context.Background(), "1.3.6.1.2.1.1.4", 2, walk)
//
//	assert.NoError(t, err)
//}
