package provisioning

const credentialTypeSetError = "can not set credential as %s as its already set as another %s"

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

type genericCredential struct {
	data map[string]interface{}
}

// CredentialBuilder - builder to create new credentials to send to Central
type CredentialBuilder interface {
	SetOAuth(id, secret string) CredentialBuilder
	SetAPIKey(key string) CredentialBuilder
	SetCredential(data map[string]interface{}) CredentialBuilder
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

// SetOauth - set the credential as an Oauth type
func (c *credentialBuilder) SetOAuth(id, secret string) CredentialBuilder {
	c.credential.credentialType = oauth
	c.credential.data = map[string]interface{}{
		id:     id,
		secret: secret,
	}
	return c
}

// SetAPIKey - set the credential as an API Key type
func (c *credentialBuilder) SetAPIKey(key string) CredentialBuilder {
	c.credential.credentialType = apiKey
	c.credential.data = map[string]interface{}{
		key: key,
	}
	return c
}

// SetCredential - set the credential
func (c *credentialBuilder) SetCredential(data map[string]interface{}) CredentialBuilder {
	c.credential.credentialType = other
	c.credential.data = data
	return c
}
