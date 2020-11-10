package config

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/exception"
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
	Timeout        time.Duration `config:"timeout"`
}

func newAuthConfig() AuthConfig {
	return &AuthConfiguration{
		Timeout: 30 * time.Second,
	}
}

func (a *AuthConfiguration) validate() {
	if a.URL == "" {
		exception.Throw(errors.New("Error auth.url not set in config"))
	}

	if a.GetRealm() == "" {
		exception.Throw(errors.New("Error auth.realm not set in config"))
	}

	if a.GetClientID() == "" {
		exception.Throw(errors.New("Error auth.clientid not set in config"))
	}

	if a.GetPrivateKey() == "" {
		exception.Throw(errors.New("Error auth.privatekey not set in config"))
	} else {
		if !fileExists(a.GetPrivateKey()) {
			privateKeyData := os.Getenv("CENTRAL_AUTH_PRIVATEKEY_DATA")
			if privateKeyData == "" {
				exception.Throw(errors.New("Error auth.privatekey not set in config"))
			}
			saveKeyData(a.GetPrivateKey(), privateKeyData)
		}
	}

	if a.GetPublicKey() == "" {
		exception.Throw(errors.New("Error auth.publickey not set in config"))
	} else {
		if !fileExists(a.GetPublicKey()) {
			publicKeyData := os.Getenv("CENTRAL_AUTH_PUBLICKEY_DATA")
			if publicKeyData == "" {
				exception.Throw(errors.New("Error auth.publickey not set in config"))
			}
			saveKeyData(a.GetPublicKey(), publicKeyData)
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
	ioutil.WriteFile(filename, dataBytes, 0600)
}
