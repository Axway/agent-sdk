package util

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

var chars = []byte(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`)

func generateString(size int) string {
	buffer := make([]byte, size)
	for i := range buffer {
		buffer[i] = chars[rand.Intn(len(chars))]
	}
	return string(buffer)
}

func TestEncryptor(t *testing.T) {

	pub2048, priv2048, err := NewKeyPair(2048)
	assert.Nil(t, err)
	pub4096, priv4096, err := NewKeyPair(4096)
	assert.Nil(t, err)
	tests := []struct {
		name          string
		data          string
		publicKey     string
		privateKey    string
		alg           string
		hash          string
		hasErr        bool
		hasEncryptErr bool
		expectJWE     bool
	}{
		{
			name:       "encrypt using PKCS-SHA1 with 2048 key",
			data:       generateString(100),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "PKCS",
			hash:       "SHA1",
		},
		{
			name:       "encrypt using PKCS-SHA1 with 2048 key",
			data:       generateString(246),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "PKCS",
			hash:       "SHA1",
			expectJWE:  true,
		},
		{
			name:       "encrypt using PKCS-SHA256 with 4096 key",
			data:       generateString(100),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "PKCS",
			hash:       "SHA256",
		},
		{
			name:       "encrypt using PKCS-SHA256 with 4096 key",
			data:       generateString(502),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "PKCS",
			hash:       "SHA256",
			expectJWE:  true,
		},
		{
			name:       "encrypt using RSA-OAEP-SHA1 with 2048 key",
			data:       generateString(100),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "RSA-OAEP",
			hash:       "SHA1",
		},
		{
			name:       "encrypt using RSA-OAEP-SHA1 with 2048 key",
			data:       generateString(215),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "RSA-OAEP",
			hash:       "SHA1",
			expectJWE:  true,
		},
		{
			name:       "encrypt using RSA-OAEP-SHA1 with 4096 key",
			data:       generateString(100),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "RSA-OAEP",
			hash:       "SHA1",
		},
		{
			name:       "encrypt using RSA-OAEP-SHA1 with 4096 key",
			data:       generateString(471),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "RSA-OAEP",
			hash:       "SHA1",
			expectJWE:  true,
		},
		{
			name:       "encrypt using RSA-OAEP-SHA256 with 2048 key",
			data:       generateString(100),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
		},
		{
			name:       "encrypt using RSA-OAEP-SHA256 with 2048 key",
			data:       generateString(191),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
			expectJWE:  true,
		},
		{
			name:       "encrypt using RSA-OAEP-SHA256 with 2048 key",
			data:       generateString(191),
			publicKey:  pub2048,
			privateKey: priv2048,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
			expectJWE:  true,
		},
		{
			name:       "encrypt using RSA-OAEP-SHA256 with 4096 key",
			data:       generateString(100),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
		},
		{
			name:       "encrypt using RSA-OAEP-SHA256 with 4096 key",
			data:       generateString(447),
			publicKey:  pub4096,
			privateKey: priv4096,
			alg:        "RSA-OAEP",
			hash:       "SHA256",
			expectJWE:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := NewEncryptor(tc.publicKey, tc.alg, tc.hash)
			assert.Nil(t, err)
			encrypted, err := enc.Encrypt(tc.data)
			assert.Nil(t, err)

			isJWE := isJWE(encrypted)
			assert.Equal(t, tc.expectJWE, isJWE)

			dec, _ := NewDecryptor(tc.privateKey, tc.alg, tc.hash)
			decrypted, _ := dec.Decrypt(encrypted)
			assert.Equal(t, tc.data, decrypted)
		})
	}
}
