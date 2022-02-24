package provisioning

import "fmt"

const credentialTypeSetError = "can not set credential as %s as its already set as another %s"

type Credential struct {
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

type CredentialBuilder interface {
	SetOAuth(id, secret string) CredentialBuilder
	SetAPIKey(key string) CredentialBuilder
	Process() (*Credential, error)
}

type credentialBuilder struct {
	err        error
	credential *Credential
}

func NewCredentialBuilder() CredentialBuilder {
	return &credentialBuilder{
		credential: &Credential{},
	}
}

func (c *credentialBuilder) Process() (*Credential, error) {
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
