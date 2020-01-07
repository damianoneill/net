package client

// Defines structs describing netconf configuration.

// Config defines properties that configure netconf session behaviour.
type Config struct {
	// Defines the time in seconds that the client will wait to receive a hello message from the server.
	SetupTimeoutSecs int
}

var DefaultConfig = &Config{
	SetupTimeoutSecs: 5,
}
