package snmp

import (
	"io"
	"net"

	"github.com/pkg/errors"

	"github.com/geoffgarside/ber"
)

// Server provides an interface for receiving Trap and Inform messages.
// This is only defined because it will facilitate unit testing of calling code that might want to mock the server
// factory.
type Server io.Closer

// Handler is the interface that needs to be supported by the callback provided when a server is instantiated.
type Handler interface {
	// NewMessage is called when a trap/inform message has been received.
	// pdu defines the content of the message.
	// isInform defines the message type.
	// sourceAddr is the address which originated the message
	// Note that a NewMessage invocation will block the receipt of other messages.
	// In the case of an inform message, it will also block the transmission of the acknowledgement message.
	// It is the responsibility of the Handler implementation to return in a timely fashion.
	NewMessage(pdu *PDU, isInform bool, sourceAddr net.Addr)
}

type serverImpl struct {
	conn    net.PacketConn
	config  *serverConfig
	handler Handler
}

func (s *serverImpl) Close() error {
	return s.conn.Close()
}

// Launches a goroutine to process incoming messages.
func (s *serverImpl) handleMessages() {
	go func() {
		s.config.trace.StartListening(s.conn.LocalAddr())
		err := s.listen()
		s.config.trace.StopListening(s.conn.LocalAddr(), err)
	}()
}

// Processes incoming messages.
func (s *serverImpl) listen() error {

	for {
		input, addr, err := s.readMessage()
		if err != nil {
			return err
		}

		err = s.processMessage(input, addr)
		if err != nil {
			s.config.trace.Error(s.config, err)
		}
	}
}

func (s *serverImpl) processMessage(input []byte, addr net.Addr) error {
	pkt := &packet{}
	if _, err := ber.Unmarshal(input, pkt); err != nil {
		return errors.Wrap(err, "failed to unmarshal packet")
	}

	mType := pkt.RawPdu.FullBytes[0]
	if mType != inform && mType != v2Trap {
		return errors.Errorf("unrecognised message type %d", mType)
	}

	rawResponsePDU := make([]byte, len(pkt.RawPdu.FullBytes))
	copy(rawResponsePDU, pkt.RawPdu.FullBytes)
	// Replace SNMP PDU Type with ASN1 sequence tag.
	rawResponsePDU[0] = 0x30

	rawPDU := &rawPDU{}
	if _, err := ber.Unmarshal(rawResponsePDU, rawPDU); err != nil {
		return errors.Wrap(err, "failed to unmarshal pdu")
	}

	pdu, err := unmarshalValues(rawPDU)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal values")
	}

	s.handler.NewMessage(pdu, mType == inform, addr)

	if mType == inform {
		err = s.acknowledgeInform(pkt, addr)
	}
	return err
}

func (s *serverImpl) acknowledgeInform(pkt *packet, addr net.Addr) error {
	pkt.RawPdu.FullBytes[0] = getResponse
	resp, err := ber.Marshal(*pkt)
	if err != nil {
		return errors.Wrap(err, "failed to marshal response")
	}

	err = s.writeMessage(resp, addr)
	return err
}

func (s *serverImpl) writeMessage(message []byte, addr net.Addr) error {
	_, err := s.conn.WriteTo(message, addr)
	s.config.trace.WriteComplete(s.config, addr, message, err)
	return err
}

func (s *serverImpl) readMessage() (input []byte, addr net.Addr, err error) {
	input = make([]byte, maxInputBufferSize)

	n, addr, err := s.conn.ReadFrom(input)
	defer s.config.trace.ReadComplete(s.config, addr, input[0:n], err)
	if err != nil {
		return nil, nil, err
	}

	return input[0:n], addr, nil
}
