package snmp

import (
	"errors"
	"testing"
)

func TestNoOpServerHooks(t *testing.T) {

	hooks := NoOpServerHooks
	hooks.WriteComplete(&serverConfig{}, nil, nil, errors.New("problem"))
	hooks.Error(&serverConfig{}, errors.New("problem"))
}
