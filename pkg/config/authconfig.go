package config

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const tokenEndpoint = "/protocol/openid-connect/token"

// AuthConfig - Interface for service account config
type AuthConfig interface {
	GetTokenURL() string
	GetRealm() string
	GetAudience() string
	GetClientID() string
	GetPrivateKey() string
	GetPublicKey() string
	GetKeyPassword() string
	GetTimeout() time.Duration
	validate()
}

// AuthConfiguration -
type AuthConfiguration struct {
	AuthConfig
	URL            string        `config:"url"`
	Realm          string        `config:"realm"`
	ClientID       string        `config:"clientId"`
	PrivateKey     string        `config:"privateKey"`
	PublicKey      string        `config:"publicKey"`
	PrivateKeyData string        `config:"privateKeyData"`
	PublicKeyData  string        `config:"publicKeyData"`
	KeyPwd         string        `config:"keyPassword"`
	ClientIDReuse  bool          `config:"clientIdReuse"`
	Timeout        time.Duration `config:"timeout"`
}

func newAuthConfig() AuthConfig {
	return &AuthConfiguration{
		Timeout: 30 * time.Second,
	}
}

func newTestAuthConfig() AuthConfig {
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY_DATA", "1")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY_DATA", "1")
	return &AuthConfiguration{
		Timeout:    30 * time.Second,
		URL:        "https://auth.com",
		Realm:      "realm",
		ClientID:   "clientid",
		PrivateKey: "file",
		PublicKey:  "file",
	}
}

func (a *AuthConfiguration) validate() {
	if a.URL == "" {
		exception.Throw(ErrBadConfig.FormatError(pathAuthURL))
	} else if _, err := url.ParseRequestURI(a.URL); err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathAuthURL))
	}

	if a.GetRealm() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathAuthRealm))
	}

	if a.GetClientID() == "" {
		// raise deprecation warning for IDs prefixed DOSA_
		if strings.HasPrefix(a.GetClientID(), "DOSA_") {
			log.Warn("DOSA_* service accounts are deprecated, please migrate to an Amplify Platform Service account")
		}
		exception.Throw(ErrBadConfig.FormatError(pathAuthClientID))
	}

	a.validatePrivateKey()
	a.validatePublicKey()
}

func (a *AuthConfiguration) validatePrivateKey() {
	if a.GetPrivateKey() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathAuthPrivateKey))
	} else {
		if !fileExists(a.GetPrivateKey()) {
			privateKeyData := os.Getenv("CENTRAL_AUTH_PRIVATEKEY_DATA")
			if privateKeyData == "" {
				exception.Throw(ErrBadConfig.FormatError(pathAuthPrivateKey))
			}
			saveKeyData(a.GetPrivateKey(), privateKeyData)
		}
		// Validate that the file is readable
		if _, err := os.Open(a.GetPrivateKey()); err != nil {
			exception.Throw(ErrReadingKeyFile.FormatError("private key", a.GetPrivateKey()))
		}
	}
}

func (a *AuthConfiguration) validatePublicKey() {
	if a.GetPublicKey() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathAuthPublicKey))
	} else {
		if !fileExists(a.GetPublicKey()) {
			publicKeyData := os.Getenv("CENTRAL_AUTH_PUBLICKEY_DATA")
			if publicKeyData == "" {
				exception.Throw(ErrBadConfig.FormatError(pathAuthPublicKey))
			}
			saveKeyData(a.GetPublicKey(), publicKeyData)
		}
		// Validate that the file is readable
		if _, err := os.Open(a.GetPublicKey()); err != nil {
			exception.Throw(ErrReadingKeyFile.FormatError("public key", a.GetPublicKey()))
		}
	}
}

// GetTokenURL - Returns the token URL
func (a *AuthConfiguration) GetTokenURL() string {
	if a.URL == "" || a.Realm == "" {
		return ""
	}
	return a.URL + "/realms/" + a.Realm + tokenEndpoint
}

// GetRealm - Returns the token audience URL
func (a *AuthConfiguration) GetRealm() string {
	return a.Realm
}

// GetAudience - Returns the token audience URL
func (a *AuthConfiguration) GetAudience() string {
	if a.URL == "" || a.Realm == "" {
		return ""
	}
	return a.URL + "/realms/" + a.Realm
}

// GetClientID - Returns the token audience URL
func (a *AuthConfiguration) GetClientID() string {
	return a.ClientID
}

// GetPrivateKey - Returns the private key file path
func (a *AuthConfiguration) GetPrivateKey() string {
	return a.PrivateKey
}

// GetPublicKey - Returns the public key file path
func (a *AuthConfiguration) GetPublicKey() string {
	return a.PublicKey
}

// GetKeyPassword - Returns the token audience URL
func (a *AuthConfiguration) GetKeyPassword() string {
	return a.KeyPwd
}

// GetTimeout - Returns the token audience URL
func (a *AuthConfiguration) GetTimeout() time.Duration {
	return a.Timeout
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func saveKeyData(filename string, data string) {
	dataBytes := []byte(data)
	os.WriteFile(filename, dataBytes, 0600)
}
