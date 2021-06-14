package netconf

import (
	"errors"
	"testing"
)

func TestDefaultHooksForUntestableExceptions(t *testing.T) {
	hooks := DefaultLoggingHooks
	session := &SessionHandler{}
	hooks.ClientHello(session)
	hooks.EndSession(session, errors.New("failed"))
	hooks.Encoded(session, errors.New("failed"))
	hooks.Decoded(session, errors.New("failed"))
}

func TestNoLoggingHooks(t *testing.T) {
	hooks := NoOpLoggingHooks
	session := &SessionHandler{}
	hooks.ClientHello(session)
	hooks.EndSession(session, errors.New("failed"))
	hooks.Encoded(session, errors.New("failed"))
	hooks.Decoded(session, errors.New("failed"))
}
