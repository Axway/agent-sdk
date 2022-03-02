package provisioning

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	GetApplicationName() string        // returns the name of the managed application for this credential
	GetCredentialType() CredentialType // returns the type of credential related to this request
	GetRequestType() string            // returns the type of request being made Provision/Deprovision/Renew
	GetProperty(key string) string     // returns the value for the key
}
