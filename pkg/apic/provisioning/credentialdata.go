package provisioning

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetCredentialType returns the type of credential related to this request
	GetCredentialType() CredentialType
	// GetRequestType returns the type of request being made Provision/Deprovision/Renew
	GetRequestType() string
	// GetCredentialDetails returns a value found on the 'x-agent-details' sub resource of the Credential
	GetCredentialDetails(key string) interface{}
	// GetApplicationDetails returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
	GetApplicationDetails(key string) interface{}
}
