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
