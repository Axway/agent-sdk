package oauth

import (
	"crypto/rsa"

	"github.com/Axway/agent-sdk/pkg/util"
)

type KeyReader interface {
	GetPrivateKey() (*rsa.PrivateKey, error)
	GetPublicKey() ([]byte, error)
}

type keyReader struct {
	privKey   string // path to rsa encoded private key, used to sign platform tokens
	publicKey string // path to the rsa encoded public key
	password  string // path to password for private key
}

func NewKeyReader(privateKey, publicKey, password string) KeyReader {
	return &keyReader{
		privKey:   privateKey,
		publicKey: publicKey,
		password:  password,
	}
}

func (kr *keyReader) GetPrivateKey() (*rsa.PrivateKey, error) {
	return util.ReadPrivateKeyFile(kr.privKey, kr.password)
}

// getPublicKey from the path provided
func (kr *keyReader) GetPublicKey() ([]byte, error) {
	return util.ReadPublicKeyBytes(kr.publicKey)
}
