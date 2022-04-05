package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
	GetAccessRequestDetailsValue(key string) string
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplications.
	GetApplicationDetailsValue(key string) string
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetAccessRequestData returns the map[string]interface{} of data from the request
	GetAccessRequestData() map[string]interface{}
	// GetInstanceDetails returns the 'x-agent-details' sub resource of the API Service Instance
	GetInstanceDetails() map[string]interface{}
}
