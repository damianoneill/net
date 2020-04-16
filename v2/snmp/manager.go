package snmp

import (
	"context"
	"net"
)

// Manager provides an interface for SNMP device management.
type Manager interface {
	// Issues an SNMP GET request for the specified oids.
	Get(ctx context.Context, oids []string) (*Response, error)

	// Issues an SNMP GET NEXT request for the specified oids.
	GetNext(ctx context.Context, oids []string) (*Response, error)

	// Issues an SNMP GET BULK request for the specified oids.
	GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*Response, error)

	// Issues an SNMP GET BULK request starting from the specified oid, invoking the function walker for each PDU.
	GetWalk(ctx context.Context, oid string, walker Walker) error
}

// Response defines the response to a Get... request.
type Response struct {
	// TBD
}

// Walker defines a function that will be called for each PDU processed by the GetWalk method.
// If the function returns an error, the walk will be terminated.
type Walker func(pdu *PDU) error

// PDU defines ...
type PDU struct {
	// TBD
}

type managerImpl struct {
	conn   net.Conn
	config *managerConfig
}

func (m *managerImpl) Get(ctx context.Context, oids []string) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetNext(ctx context.Context, oids []string) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetBulk(ctx context.Context, oids []string, nonRepeaters int, maxRepetitions int) (*Response, error) {
	// TODO
	return nil, nil
}

func (m *managerImpl) GetWalk(ctx context.Context, oid string, walker Walker) error {
	// TODO
	return nil
}
