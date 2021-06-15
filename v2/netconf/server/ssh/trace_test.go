package ssh

import (
	"errors"
	"testing"
)

func TestDefaultHooksForUntestableExceptions(t *testing.T) {
	hooks := DefaultLoggingHooks
	hooks.SSHChannelAccept(nil, errors.New("failed"))
	hooks.SubsystemRequestReply(errors.New("failed"))
}
