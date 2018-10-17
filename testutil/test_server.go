package testutil

import (
	"bufio"
	"encoding/pem"
	"fmt"
	"net"
	"testing"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

// SSHServer represents a test SSH Server
type SSHServer struct {
	listener net.Listener
}

// Handler is a function that handles i/o to/from an SSH channel
type Handler func(t *testing.T, ch ssh.Channel)

// NewSSHServer deflivers a new test SSH Server, with a Handler that simply echoes lines received.
// The server implements password authentication with the given credentials.
func NewSSHServer(t *testing.T, uname, password string) *SSHServer {

	return NewSSHServerHandler(t, uname, password, echoer)
}

// NewSSHServer deflivers a new test SSH Server, with a custom channel handler.
// The server implements password authentication with the given credentials.
func NewSSHServerHandler(t *testing.T, uname, password string, handler Handler) *SSHServer {

	listener, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err, "Listen failed")

	go acceptConnections(t, listener, newSSHServerConfig(t, uname, password), handler)

	return &SSHServer{listener: listener}
}

// Port delivers the tcp port number on which the server is listening.
func (ts *SSHServer) Port() int {
	return ts.listener.Addr().(*net.TCPAddr).Port
}

// Close closes any resources used by the server.
func (ts *SSHServer) Close() {
	// nolint: gosec, errcheck
	ts.listener.Close()
}

func acceptConnections(t *testing.T, listener net.Listener, config *ssh.ServerConfig, handler Handler) {
	// nolint: gosec, errcheck
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

			// Handle the "subsystem" request.
			go func(in <-chan *ssh.Request) {
				for req := range in {
					assert.NoError(t, req.Reply(req.Type == "subsystem", nil), "Request reply failed")
				}
			}(requests)

			go func() {
				defer dataChan.Close()
				handler(t, dataChan)
			}()
		}
	}
}

func newSSHServerConfig(t *testing.T, uname, password string) *ssh.ServerConfig {
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

func generateHostKey(t *testing.T) (hostkey ssh.Signer) {

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
	t.Error("Failed to generate host key", err)
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

// Simple Handler implementation that echoes lines.
func echoer(t *testing.T, ch ssh.Channel) {
	chReader := bufio.NewReader(ch)
	chWriter := bufio.NewWriter(ch)
	for {
		input, err := chReader.ReadString('\n')
		if err != nil {
			return
		}
		_, err = chWriter.WriteString(fmt.Sprintf("GOT:%s", input))
		assert.NoError(t, err, "Write failed")
		chWriter.Flush()
	}
}
