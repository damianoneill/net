package snmp

import (
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/damianoneill/net/v2/snmp/mocks"
	"github.com/golang/mock/gomock"

	assert "github.com/stretchr/testify/require"
)

func TestHandleTrap(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockPacketConn(mockCtrl)

	trap := messageWithType(v2Trap)
	mockConn.EXPECT().LocalAddr().Return(nil).AnyTimes()
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			copy(input, trap)
			return len(trap), nil, nil
		}).Times(1)
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			return 0, nil, errors.New("read failed")
		}).MaxTimes(1)
	mockConn.EXPECT().Close().Return(nil)

	config := defaultServerConfig
	config.trace = NoOpServerHooks
	config.resolveServerHooks()
	h := newHandler()
	h.wg.Add(1)
	s := &serverImpl{config: &config, conn: mockConn, handler: h}
	defer s.Close()

	s.handleMessages()

	h.wg.Wait()
	assert.NotZero(t, h.pdu.VarbindList[0].TypedValue.Value, "upTime should be defined")
	assert.Equal(t, "1.3.6.1.1.2.3", h.pdu.VarbindList[1].TypedValue.String())
	assert.Equal(t, "123456", h.pdu.VarbindList[2].TypedValue.String())
}

func TestHandleInform(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockPacketConn(mockCtrl)

	iMessage := messageWithType(inform)
	mockConn.EXPECT().LocalAddr().Return(nil).AnyTimes()
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			copy(input, iMessage)
			return len(iMessage), nil, nil
		})
	mockConn.EXPECT().WriteTo(messageWithType(getResponse), gomock.Any()).Return(len(iMessage), nil)
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			return 0, nil, errors.New("read failed")
		}).MaxTimes(1)
	mockConn.EXPECT().Close().Return(nil)

	config := defaultServerConfig
	config.trace = DiagnosticServerHooks
	config.resolveServerHooks()
	h := newHandler()
	h.wg.Add(1)
	s := &serverImpl{config: &config, conn: mockConn, handler: h}
	defer s.Close()

	s.handleMessages()

	h.wg.Wait()
	assert.NotZero(t, h.pdu.VarbindList[0].TypedValue.Value, "upTime should be defined")
	assert.Equal(t, "1.3.6.1.1.2.3", h.pdu.VarbindList[1].TypedValue.String())
	assert.Equal(t, "123456", h.pdu.VarbindList[2].TypedValue.String())
}

func TestInformAcknwoledgementFailure(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockPacketConn(mockCtrl)

	iMessage := messageWithType(inform)
	mockConn.EXPECT().LocalAddr().Return(nil).AnyTimes()
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			copy(input, iMessage)
			return len(iMessage), nil, nil
		})
	mockConn.EXPECT().WriteTo(messageWithType(getResponse), gomock.Any()).Return(0, errors.New("write failure"))
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			return 0, nil, errors.New("read failed")
		}).MaxTimes(1)
	mockConn.EXPECT().Close().Return(nil)

	config := defaultServerConfig
	config.trace = DefaultServerHooks
	config.resolveServerHooks()
	h := newHandler()
	h.wg.Add(1)
	s := &serverImpl{config: &config, conn: mockConn, handler: h}
	defer s.Close()

	s.handleMessages()

	h.wg.Wait()
	assert.NotZero(t, h.pdu.VarbindList[0].TypedValue.Value, "upTime should be defined")
	assert.Equal(t, "1.3.6.1.1.2.3", h.pdu.VarbindList[1].TypedValue.String())
	assert.Equal(t, "123456", h.pdu.VarbindList[2].TypedValue.String())
}

func TestIgnoringUnsupportedMessageType(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockPacketConn(mockCtrl)

	h := newHandler()

	iMessage := messageWithType(getMessage) // Neither trap nor inform...
	mockConn.EXPECT().LocalAddr().Return(nil).AnyTimes()
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			copy(input, iMessage)
			return len(iMessage), nil, nil
		})
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			defer h.wg.Done() // Wake up main goroutine.
			return 0, nil, errors.New("read failed")
		}).MaxTimes(1)
	mockConn.EXPECT().Close().Return(nil)

	config := defaultServerConfig
	config.trace = DiagnosticServerHooks
	h.wg.Add(1)
	s := &serverImpl{config: &config, conn: mockConn, handler: h}
	defer s.Close()

	s.handleMessages()

	h.wg.Wait()
	assert.Nil(t, h.pdu)
}

func TestMessageParseFailure(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockConn := mocks.NewMockPacketConn(mockCtrl)

	h := newHandler()

	garbageMessage := []byte{0xff, 0xff, 0xff}
	mockConn.EXPECT().LocalAddr().Return(nil).AnyTimes()
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			copy(input, garbageMessage)
			return len(garbageMessage), nil, nil
		})
	mockConn.EXPECT().ReadFrom(gomock.Any()).DoAndReturn(
		func(input []byte) (int, net.Addr, error) {
			defer h.wg.Done() // Wake up main goroutine.
			return 0, nil, errors.New("read failed")
		}).MaxTimes(1)
	mockConn.EXPECT().Close().Return(nil)

	config := defaultServerConfig
	config.trace = DiagnosticServerHooks
	h.wg.Add(1)
	s := &serverImpl{config: &config, conn: mockConn, handler: h}
	defer s.Close()

	s.handleMessages()

	h.wg.Wait()
	assert.Nil(t, h.pdu)
}

func messageWithType(mType byte) []byte {
	trap := []byte{
		// Message Type = Sequence, Length = 82
		0x30, 0x52,
		// Version Type = Integer, Length = 1, Value = 1
		0x02, 0x01, 0x01,
		// Community String Type = Octet String, Length = 6, Value = public
		0x04, 0x06, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
		// PDU Type = mType, Length = 69
		mType, 0x45,
		// Request ID Type = Integer, Length = 4, Value = ...
		0x02, 0x04, 0x3d, 0xcd, 0xa1, 0x06,
		// Error Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Error Index Type = Integer, Length = 1, Value = 0
		0x02, 0x01, 0x00,
		// Varbind List Type = Sequence, Length = 55
		0x30, 0x37,
		// Varbind Type = Sequence, Length = 16
		0x30, 0x10,
		// Object Identifier Type = Object Identifier, Length = 8, Value = 1.3.6.1.2.1.1.3.0
		0x06, 0x08, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x03, 0x00,
		// Value Type = Time, Length = 4, Value = ...
		0x43, 0x04, 0x03, 0x01, 0x7b, 0x89,
		// Varbind Type = Sequence, Length = 20
		0x30, 0x14,
		// Object Identifier Type = Object Identifier, Length = 10, Value = 1.3.6.1.6.3.1.1.4.1.0
		0x06, 0x0a, 0x2b, 0x06, 0x01, 0x06, 0x03, 0x01, 0x01, 0x04, 0x01, 0x00,
		// Value Type = Object Identifier, Length = 1, Value = 1.3.6.1.1.2.3
		0x06, 0x06, 0x2b, 0x06, 0x01, 0x01, 0x02, 0x03,
		// Varbind Type = Sequence, Length = 13
		0x30, 0x0d,
		// Object Identifier Type = Object Identifier, Length = 6, Value = 1.3.6.1.7.8.9
		0x06, 0x06, 0x2b, 0x06, 0x01, 0x07, 0x08, 0x09,
		// Value Type = Integer, Length = 3, Value = 123456
		0x02, 0x03, 0x01, 0xe2, 0x40,
	}
	return trap
}

type handler struct {
	wg  *sync.WaitGroup
	pdu *PDU
}

func newHandler() *handler {
	return &handler{wg: &sync.WaitGroup{}}
}

func (h *handler) NewMessage(pdu *PDU, isInform bool, addr net.Addr) {
	h.pdu = pdu
	h.wg.Done()
}

// Tests against real SNMP agent. Useful for diagnostics.
//
//
//func TestRealServer(t *testing.T) {
//
//	h := newHandler()
//	h.wg.Add(1)
//	m, err := NewServerFactory().NewServer(context.Background(), h, Hooks(DiagnosticServerHooks))
//	assert.NoError(t, err)
//	defer m.Close()
//
//	Generate an inform message:
//    	snmpinform -d -v 2c -c public localhost '' 1.3.6.1.1.2.3 1.3.6.1.7.8.9 i 123456
//
//	h.wg.Wait()
//	assert.NotZero(t, h.pdu.VarbindList[0].TypedValue.Value, "upTime should be defined")
//	assert.Equal(t, "1.3.6.1.1.2.3", h.pdu.VarbindList[1].TypedValue.String())
//	assert.Equal(t, "123456", h.pdu.VarbindList[2].TypedValue.String())
//}
