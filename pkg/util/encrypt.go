package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash"
)

// Encryptor is an interface for encrypting strings
type Encryptor interface {
	Encrypt(str string) (string, error)
}

// Decryptor is an interface for Decrypting strings
type Decryptor interface {
	Decrypt(str string) (string, error)
}

// encryptor implements the Encryptor interface
type encryptor struct {
	alg  string
	key  *rsa.PublicKey
	hash hash.Hash
}

// NewEncryptor creates a struct to handle encryption based on the provided key, algorithm, and hash.
func NewEncryptor(key, alg, hash string) (Encryptor, error) {
	enc := &encryptor{
		alg: alg,
	}

	pub, err := enc.newPub(key)
	if err != nil {
		return nil, err
	}

	h, err := enc.newHash(hash)
	if err != nil {
		return nil, err
	}

	ok := enc.validateAlg()
	if !ok {
		return nil, fmt.Errorf("unexpected encryption algorithm: %s", alg)
	}

	enc.hash = h
	enc.key = pub
	return enc, nil
}

// Encrypt encrypts a string based on the provided public key and algorithm.
func (e *encryptor) Encrypt(str string) (string, error) {
	bts, err := e.encAlgorithm(str)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %s", err)
	}

	return string(bts), nil
}

func (e *encryptor) newPub(key string) (*rsa.PublicKey, error) {
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

func (e *encryptor) newHash(hash string) (hash.Hash, error) {
	switch hash {
	case "":
		fallthrough
	case "SHA256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("unexpected encryption hash: %s", hash)
	}
}

func (e *encryptor) validateAlg() bool {
	switch e.alg {
	case "":
		fallthrough
	case "RSA-OAEP":
		return true
	case "PKCS":
		return true
	default:
		return false
	}
}

func (e *encryptor) encAlgorithm(msg string) ([]byte, error) {
	switch e.alg {
	case "":
		fallthrough
	case "RSA-OAEP":
		return rsa.EncryptOAEP(e.hash, rand.Reader, e.key, []byte(msg), nil)
	case "PKCS":
		return rsa.EncryptPKCS1v15(rand.Reader, e.key, []byte(msg))
	default:
		return nil, fmt.Errorf("unexpected encryption algorithm: %s", e.alg)
	}
}
