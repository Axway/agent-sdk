package provisioning

import "time"

const (
	oauth = "oauth"
	other = "other"
)

// Credential - holds the details about the credential to send to encrypt and send to platform
type Credential interface {
	GetData() map[string]interface{}
	GetExpirationTime() time.Time
}

type credential struct {
	Credential
	credentialType string
	data           map[string]interface{}
	expTime        time.Time
}

func (c credential) GetData() map[string]interface{} {
	return c.data
}

func (c credential) GetExpirationTime() time.Time {
	return c.expTime
}

// CredentialBuilder - builder to create new credentials to send to Central
type CredentialBuilder interface {
	SetExpirationTime(expTime time.Time) CredentialBuilder
	SetOAuthID(id string) Credential
	SetOAuthIDAndSecret(id, secret string) Credential
	SetAPIKey(key string) Credential
	SetHTTPBasic(username, password string) Credential
	SetCredential(data map[string]interface{}) Credential
}

type credentialBuilder struct {
	credential *credential
}

// NewCredentialBuilder - create a credential builder
func NewCredentialBuilder() CredentialBuilder {
	return &credentialBuilder{
		credential: &credential{},
	}
}

// SetOAuthID - set the credential as an Oauth type
func (c *credentialBuilder) SetOAuthID(id string) Credential {
	c.credential.credentialType = oauth
	c.credential.data = map[string]interface{}{
		OauthClientID: id,
	}
	return c.credential
}

// SetOAuthIDAndSecret - set the credential as an Oauth type
func (c *credentialBuilder) SetOAuthIDAndSecret(id, secret string) Credential {
	c.credential.credentialType = oauth
	c.credential.data = map[string]interface{}{
		OauthClientID:     id,
		OauthClientSecret: secret,
	}
	return c.credential
}

// SetAPIKey - set the credential as an API Key type
func (c *credentialBuilder) SetAPIKey(key string) Credential {
	c.credential.credentialType = APIKeyCRD
	c.credential.data = map[string]interface{}{
		APIKey: key,
	}
	return c.credential
}

// SetHTTPBasic - set the credential as an API Key type
func (c *credentialBuilder) SetHTTPBasic(username, password string) Credential {
	c.credential.credentialType = BasicAuthCRD
	c.credential.data = map[string]interface{}{
		BasicAuthUsername: username,
		BasicAuthPassword: password,
	}
	return c.credential
}

// SetExpirationTime - set the credential expiration time
func (c *credentialBuilder) SetExpirationTime(expTime time.Time) CredentialBuilder {
	c.credential.expTime = expTime
	return c
}

// SetCredential - set the credential
func (c *credentialBuilder) SetCredential(data map[string]interface{}) Credential {
	c.credential.credentialType = other
	c.credential.data = data
	return c.credential
}
