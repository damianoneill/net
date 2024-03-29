package testserver

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"

	assert "github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// SSHServer represents a test SSH Server
type SSHServer struct {
	listener net.Listener
}

// SSHHandler is the interface that is implemented to handle an SSH channel.
type SSHHandler interface {
	// Handler is a function that handles i/o to/from an SSH channel
	Handle(t assert.TestingT, ch ssh.Channel)
}

// HandlerFactory is a test function that will deliver an SSHHandler.
type HandlerFactory func(t assert.TestingT) SSHHandler

// NewSSHServer deflivers a new test SSH Server, with a Handler that simply echoes lines received.
// The server implements password authentication with the given credentials.
func NewSSHServer(t assert.TestingT, uname, password string) *SSHServer {
	return NewSSHServerHandler(t, uname, password, func(t assert.TestingT) SSHHandler { return &echoer{} })
}

// NewSSHServerHandler deflivers a new test SSH Server, with a custom channel handler.
// The server implements password authentication with the given credentials.
func NewSSHServerHandler(t assert.TestingT, uname, password string, factory HandlerFactory, opts ...ServerOption) *SSHServer {
	serverOptions := &serverOptions{requestTypes: []string{"subsystem"}}
	for _, opt := range opts {
		opt(serverOptions)
	}

	listener, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err, "Listen failed")

	go acceptConnections(t, listener, newSSHServerConfig(t, uname, password), factory, serverOptions)

	return &SSHServer{listener: listener}
}

// ServerOption implements options for configuring test server behaviour.
type ServerOption func(*serverOptions)

// serverOptions defines properties controlling test server behaviour.
type serverOptions struct {
	requestTypes []string
}

// RequestTypes defines the request types that will be 'accepted' - i.e. the request response will be 'ok' (true).
// Defaults to {"subsystem"}
func RequestTypes(types []string) ServerOption {
	return func(c *serverOptions) {
		c.requestTypes = types
	}
}

// Port delivers the tcp port number on which the server is listening.
func (ts *SSHServer) Port() int {
	return ts.listener.Addr().(*net.TCPAddr).Port
}

// Close closes any resources used by the server.
func (ts *SSHServer) Close() {
	_ = ts.listener.Close()
}

func acceptConnections(t assert.TestingT, listener net.Listener, config *ssh.ServerConfig, factory HandlerFactory, options *serverOptions) {
	for {
		nConn, err := listener.Accept()
		if err != nil {
			return
		}

		_, chch, reqch, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			return
		}

		go ssh.DiscardRequests(reqch)

		// Service the incoming Channel channel.
		for newChannel := range chch {
			dataChan, requests, err := newChannel.Accept()
			assert.NoError(t, err, "Failed to accept new channel")

			// Handle requests - subsystem, pty-req, shell etc.
			go func(in <-chan *ssh.Request) {
				for req := range in {
					typeOk := false
					for _, ty := range options.requestTypes {
						if req.Type == ty {
							typeOk = true
							break
						}
					}

					_ = req.Reply(typeOk, nil)
				}
			}(requests)

			go func() {
				defer dataChan.Close()
				factory(t).Handle(t, dataChan)
			}()
		}
	}
}

func newSSHServerConfig(t assert.TestingT, uname, password string) *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == uname && string(pass) == password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	config.AddHostKey(generateHostKey(t))
	return config
}

func generateHostKey(t assert.TestingT) (hostkey ssh.Signer) { //nolint:interfacer
	reader := rand.Reader
	bitSize := 2048
	var err error
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(reader, bitSize); err == nil {
		privateBytes := encodePrivateKeyToPEM(key)
		if hostkey, err = ssh.ParsePrivateKey(privateBytes); err == nil {
			return
		}
	}
	t.Errorf("Failed to generate host key %v", err)
	return
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

type echoer struct{}

// Simple Handler implementation that echoes lines.
func (e *echoer) Handle(t assert.TestingT, ch ssh.Channel) {
	chReader := bufio.NewReader(ch)
	chWriter := bufio.NewWriter(ch)
	for {
		input, err := chReader.ReadString('\n')
		if err != nil {
			return
		}
		_, err = chWriter.WriteString(fmt.Sprintf("GOT:%s", input))
		assert.NoError(t, err, "Write failed")
		err = chWriter.Flush()
		assert.NoError(t, err, "Flush failed")
	}
}
