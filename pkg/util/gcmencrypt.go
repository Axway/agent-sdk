package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// gcmEncryption implements the Encryptor/Decryptor interface
type gcmEncoder struct {
	gcm cipher.AEAD
}

func newGCMEncoder(key []byte) (*gcmEncoder, error) {
	block, err := aes.NewCipher(hashedKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	enc := &gcmEncoder{
		gcm: gcm,
	}
	return enc, nil
}

func hashedKey(key []byte) []byte {
	hash := sha256.New()
	hash.Write(key)
	return hash.Sum(nil)
}

// NewGCMEncryptor creates a struct to handle encryption based on the provided key.
func NewGCMEncryptor(key []byte) (Encryptor, error) {
	return newGCMEncoder(key)
}

// NewGCMDecryptor creates a struct to handle decryption based on the provided key.
func NewGCMDecryptor(key []byte) (Decryptor, error) {
	return newGCMEncoder(key)
}

// Encrypt encrypts string with a secret key and returns encrypted string.
func (e *gcmEncoder) Encrypt(str string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	encrypted := e.gcm.Seal(nonce, nonce, []byte(str), nil)
	return base64.URLEncoding.EncodeToString(encrypted), nil
}

// Decrypt decrypts encrypted string with a secret key and returns plain string.
func (e *gcmEncoder) Decrypt(str string) (string, error) {
	ed, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	nonceSize := e.gcm.NonceSize()
	if len(ed) < nonceSize {
		return "", err
	}

	nonce, cipherText := ed[:nonceSize], ed[nonceSize:]
	decrypted, err := e.gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}
