package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func PasswordConfig(uname, password string) (*ssh.ServerConfig, error) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return checkCredentials(uname, password, c, pass)
		},
	}

	hostKey, err := generateHostKey()
	if err != nil {
		return nil, err
	}
	config.AddHostKey(hostKey)
	return config, nil
}

func checkCredentials(uname, password string, c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	if c.User() == uname && string(pass) == password {
		return nil, nil
	}
	return nil, fmt.Errorf("password rejected for %q", c.User())
}

func generateHostKey() (hostkey ssh.Signer, err error) {
	reader := rand.Reader
	bitSize := 2048
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(reader, bitSize); err == nil {
		privateBytes := encodePrivateKeyToPEM(key)
		if hostkey, err = ssh.ParsePrivateKey(privateBytes); err == nil {
			return
		}
	}
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
