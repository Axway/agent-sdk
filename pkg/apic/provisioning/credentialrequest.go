package provisioning

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetCredentialType returns the type of credential related to this request
	GetCredentialType() string
	// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credential
	GetCredentialDetailsValue(key string) interface{}
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
	GetApplicationDetailsValue(key string) interface{}
}
