package util

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"hash"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

const (
	RsaOaep = "RSA-OAEP"
	Pkcs    = "PKCS"
	SHA1    = "SHA1"
	SHA256  = "SHA256"
)

// Encryptor is an interface for encrypting strings
type Encryptor interface {
	Encrypt(str string) (string, error)
}

// encryptor implements the Encryptor interface
type encryptor struct {
	alg  string
	key  *rsa.PublicKey
	hash hash.Hash
}

type decryptor struct {
	alg  string
	key  *rsa.PrivateKey
	hash hash.Hash
}

// NewEncryptor creates a struct to handle encryption based on the provided key, algorithm, and hash.
func NewEncryptor(key, alg, hash string) (Encryptor, error) {
	enc := &encryptor{
		alg: alg,
	}

	pub, err := newPublicKey(key)
	if err != nil {
		return nil, err
	}

	h, err := newHash(hash)
	if err != nil {
		return nil, err
	}

	ok := validateAlg(alg)
	if !ok {
		return nil, fmt.Errorf("unexpected encryption algorithm: %s", alg)
	}

	enc.hash = h
	enc.key = pub
	return enc, nil
}

func newPublicKey(key string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("failed to decode public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %s", err)
	}

	p, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected public key type to be *rsa.PublicKey but received %T", pub)
	}

	return p, nil
}

func newHash(hash string) (hash.Hash, error) {
	switch hash {
	case SHA1:
		return sha1.New(), nil
	case "":
		fallthrough
	case SHA256:
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unexpected encryption hash: %s", hash)
	}
}

func validateAlg(alg string) bool {
	switch alg {
	case "":
		fallthrough
	case RsaOaep:
		return true
	case Pkcs:
		return true
	default:
		return false
	}
}

// Encrypt encrypts a string based on the provided public key and algorithm
// returns either JWE or base64 encoded encrypted content
func (e *encryptor) Encrypt(str string) (string, error) {
	msgSize := e.keyMaxMessageSize()
	if len(str) > msgSize {
		return e.applyJWEEncryption(str)
	}

	encrypted, err := e.applyEncryption(str)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(encrypted)), nil
}

func (e *encryptor) keyMaxMessageSize() int {
	switch e.alg {
	case Pkcs:
		// KeySize - PKCS Padding of 11 bytes
		return e.key.Size() - 11
	case RsaOaep:
		fallthrough
	default:
		// KeySize - 2*HashSize - 2 zero octets
		return e.key.Size() - 2*e.hash.Size() - 2
	}
}

func (e *encryptor) jweKeyAlgorithm() jwa.KeyAlgorithm {
	e.hash.Size()
	switch e.alg {
	case Pkcs:
		return jwa.RSA1_5
	case RsaOaep:
		fallthrough
	default:
		// RSA-OAEP-SHA1
		if e.hash.Size() == 20 {
			return jwa.RSA_OAEP
		}
		// RSA-OAEP-SHA256
		return jwa.RSA_OAEP_256
	}
}

func (e *encryptor) applyEncryption(str string) (string, error) {
	bts, err := e.encryptWithKey(str)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %s", err)
	}
	return string(bts), nil
}

func (e *encryptor) encryptWithKey(msg string) ([]byte, error) {
	switch e.alg {
	case Pkcs:
		return rsa.EncryptPKCS1v15(rand.Reader, e.key, []byte(msg))
	case RsaOaep:
		fallthrough
	default:
		return rsa.EncryptOAEP(e.hash, rand.Reader, e.key, []byte(msg), nil)
	}
}

func (e *encryptor) applyJWEEncryption(str string) (string, error) {
	options := []jwe.EncryptOption{
		jwe.WithKey(e.jweKeyAlgorithm(), e.key),
		jwe.WithContentEncryption(jwa.A256GCM),
	}
	bts, err := jwe.Encrypt([]byte(str), options...)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %s", err)
	}

	return string(bts), nil
}

func NewKeyPair(keySize int) (public string, private string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return "", "", err
	}

	pkBts := x509.MarshalPKCS1PrivateKey(priv)
	pvBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pkBts,
	}

	privBuff := bytes.NewBuffer([]byte{})
	err = pem.Encode(privBuff, pvBlock)
	if err != nil {
		return "", "", err
	}

	pubKeyBts, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBts,
	}

	pubKeyBuff := bytes.NewBuffer([]byte{})
	err = pem.Encode(pubKeyBuff, pubKeyBlock)
	if err != nil {
		return "", "", err
	}

	return pubKeyBuff.String(), privBuff.String(), nil
}
