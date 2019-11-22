package netconf

// Defines structs describing netconf configuration.

// ClientConfig defines properties that configure netconf session behaviour.
type ClientConfig struct {
	// Defines the time in seconds that the client will wait to receive a hello message from the server.
	setupTimeoutSecs int
}

var defaultConfig = &ClientConfig{
	setupTimeoutSecs: 5,
}
