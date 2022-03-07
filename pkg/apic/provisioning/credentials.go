package provisioning

const credentialTypeSetError = "can not set credential as %s as its already set as another %s"

const apiKey = "api-key"
const oauth = "oauth"

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

func (c *credentialBuilder) hasError(credType string) bool {
	// if c.err != nil {
	// 	return true
	// }
	//
	// if c.credential.credentialType != 0 {
	// 	c.err = fmt.Errorf(credentialTypeSetError, credType, c.credential.credentialType)
	// 	return true
	// }
	return false
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
