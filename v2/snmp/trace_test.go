package snmp

import (
	"errors"
	"testing"
)

func TestDiagnosticHooksForUntestableExceptions(t *testing.T) {

	hooks := DiagnosticLoggingHooks
	hooks.Error("Context", &sessionConfig{}, errors.New("problem"))
}

func TestNoLoggingHooks(t *testing.T) {

	hooks := NoOpLoggingHooks
	hooks.Error("Context", &sessionConfig{}, errors.New("problem"))
}
