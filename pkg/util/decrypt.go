package util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

const (
	jweHeaderIdx = iota
	jweCekIdx
	jweIvIdx
	jweCipherTextIdx
	jweTagsIdx
)

// Decryptor is an interface for Decrypting strings
type Decryptor interface {
	Decrypt(str string) (string, error)
}

// NewDecryptor creates a struct to handle decryption based on the provided key.
func NewDecryptor(key, alg, hash string) (Decryptor, error) {
	dec := &decryptor{
		alg: alg,
	}

	pk, err := newPrivateKey(key)
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
	dec.hash = h
	dec.key = pk

	return dec, nil
}

func newPrivateKey(key string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("failed to decode private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (d *decryptor) Decrypt(msg string) (string, error) {
	decrypted, err := d.applyDecryption(msg)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

func isJWE(msg string) bool {
	elements := strings.Split(msg, ".")
	if _, err := base64.RawURLEncoding.DecodeString(elements[jweHeaderIdx]); err == nil && len(elements) >= 4 {
		return true
	}
	return false
}

func (d *decryptor) applyDecryption(msg string) ([]byte, error) {
	if isJWE(msg) {
		return d.applyJWEDecrypt(msg)
	}
	bts, _ := base64.StdEncoding.DecodeString(msg)
	return d.decryptWithKey(bts)
}

func (d *decryptor) decryptWithKey(bts []byte) ([]byte, error) {
	switch d.alg {
	case Pkcs:
		return rsa.DecryptPKCS1v15(rand.Reader, d.key, bts)
	case RsaOaep:
		fallthrough
	default:
		return rsa.DecryptOAEP(d.hash, rand.Reader, d.key, bts, nil)
	}
}

func parseJweElement(elements []string, index int, b64Decode bool) ([]byte, error) {
	if index < len(elements) {
		val := []byte(elements[index])
		if b64Decode {
			return base64.RawURLEncoding.DecodeString(string(val))
		}
		return val, nil
	}
	return nil, errors.New("invalid JWE element")
}

func (d *decryptor) applyJWEDecrypt(msg string) ([]byte, error) {
	elements := strings.Split(msg, ".")
	header, err := parseJweElement(elements, jweHeaderIdx, false)
	if err != nil {
		return nil, err
	}

	encryptedCek, err := parseJweElement(elements, jweCekIdx, true)
	if err != nil {
		return nil, err
	}

	iv, err := parseJweElement(elements, jweIvIdx, true)
	if err != nil {
		return nil, err
	}

	cipherText, err := parseJweElement(elements, jweCipherTextIdx, true)
	if err != nil {
		return nil, err
	}

	tags, err := parseJweElement(elements, jweTagsIdx, true)
	if err != nil {
		return nil, err
	}

	cek, err := d.decryptWithKey(encryptedCek)
	if err != nil {
		return nil, err
	}

	encoder, err := newGCMEncoderWithHashedKey(cek)
	if err != nil {
		return nil, err
	}

	combined := make([]byte, len(cipherText)+len(tags))
	copy(combined, cipherText)
	copy(combined[len(cipherText):], tags)

	buf, err := encoder.DecryptWithNonce(iv, combined, header)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
