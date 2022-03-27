package provisioning

// CredentialRequest - interface for agents to use to get necessary credential request details
type CredentialRequest interface {
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication.
	GetApplicationDetailsValue(key string) string
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetCredentialDetailsValue returns a value found on the 'x-agent-details' sub resource of the Credential
	GetCredentialDetailsValue(key string) string
	// GetCredentialType returns the type of credential related to this request
	GetCredentialType() string
	// GetCredentialData returns the map[string]interface{} of data from the request
	GetCredentialData() map[string]interface{}
}
