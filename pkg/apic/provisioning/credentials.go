package provisioning

const (
	apiKey = "api-key"
	oauth  = "oauth"
	other  = "other"
)

// Credential - holds the details about the credential to send to encrypt and send to platform
type Credential interface {
	GetData() map[string]interface{}
}

type credential struct {
	Credential
	credentialType string
	data           map[string]interface{}
}

func (c credential) GetData() map[string]interface{} {
	return c.data
}

// CredentialBuilder - builder to create new credentials to send to Central
type CredentialBuilder interface {
	SetOAuth(id, secret string) Credential
	SetAPIKey(key string) Credential
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

// SetOAuth - set the credential as an Oauth type
func (c *credentialBuilder) SetOAuth(id, secret string) Credential {
	c.credential.credentialType = oauth
	c.credential.data = map[string]interface{}{
		"id":     id,
		"secret": secret,
	}
	return c.credential
}

// SetAPIKey - set the credential as an API Key type
func (c *credentialBuilder) SetAPIKey(key string) Credential {
	c.credential.credentialType = apiKey
	c.credential.data = map[string]interface{}{
		"key": key,
	}
	return c.credential
}

// SetCredential - set the credential
func (c *credentialBuilder) SetCredential(data map[string]interface{}) Credential {
	c.credential.credentialType = other
	c.credential.data = data
	return c.credential
}
