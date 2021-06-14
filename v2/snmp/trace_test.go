package snmp

import (
	"errors"
	"testing"
)

func TestDiagnosticHooksForUntestableExceptions(t *testing.T) {
	hooks := DiagnosticLoggingHooks
	hooks.Error("Context", &SessionConfig{}, errors.New("problem"))
}

func TestNoLoggingHooks(t *testing.T) {
	hooks := NoOpLoggingHooks
	hooks.Error("Context", &SessionConfig{}, errors.New("problem"))
}
