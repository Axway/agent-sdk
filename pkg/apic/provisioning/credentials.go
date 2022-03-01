package provisioning

import "fmt"

const credentialTypeSetError = "can not set credential as %s as its already set as another %s"

// Credential - holds the details about the credential to send to encrypt and send to platform
type Credential interface{}

type credential struct {
	Credential
	credentialType credentialType
	data           interface{}
}

type oAuthCredential struct {
	id     string
	secret string
}

type apiKeyCredential struct {
	key string
}

// CredentialBuilder - builder to create new credentials to send to Central
type CredentialBuilder interface {
	SetOAuth(id, secret string) CredentialBuilder
	SetAPIKey(key string) CredentialBuilder
	Process() (Credential, error)
}

type credentialBuilder struct {
	err        error
	credential *credential
}

// NewCredentialBuilder - create a credential builder
func NewCredentialBuilder() CredentialBuilder {
	return &credentialBuilder{
		credential: &credential{},
	}
}

// Process - process the builder, returning errors
func (c *credentialBuilder) Process() (Credential, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.credential, nil
}

func (c *credentialBuilder) hasError(credType credentialType) bool {
	if c.err != nil {
		return true
	}

	if c.credential.credentialType != 0 {
		c.err = fmt.Errorf(credentialTypeSetError, credType, c.credential.credentialType)
		return true
	}
	return false
}

// SetOauth - set the credential as an Oauth type
func (c *credentialBuilder) SetOAuth(id, secret string) CredentialBuilder {
	if c.hasError(credentialTypeOAuth) {
		return c
	}

	c.credential.credentialType = credentialTypeOAuth
	c.credential.data = &oAuthCredential{
		id:     id,
		secret: secret,
	}
	return c
}

// SetAPIKey - set the credential as an API Key type
func (c *credentialBuilder) SetAPIKey(key string) CredentialBuilder {
	if c.hasError(credentialTypeAPIKey) {
		return c
	}

	c.credential.credentialType = credentialTypeAPIKey
	c.credential.data = &apiKeyCredential{
		key: key,
	}
	return c
}
