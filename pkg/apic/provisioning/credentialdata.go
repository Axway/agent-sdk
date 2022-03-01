package provisioning

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	GetApplicationName() string        // returns the name of the managed application for this credential
	GetCredentialType() credentialType // returns the type of credential related to this request
	GetRequestType() string            // returns the type of request being made Provision/Deprovision/Renew
	GetProperty(key string) string     // returns the value for the key
}

type Creds struct {
}

func (c Creds) GetApplicationName() string {
	return "app name"
}

func (c Creds) GetCredentialType() credentialType {
	return credentialTypeAPIKey
}

func (c Creds) GetRequestType() string {
	return "request type"
}

func (c Creds) GetProperty(key string) string {
	return "prop"
}
