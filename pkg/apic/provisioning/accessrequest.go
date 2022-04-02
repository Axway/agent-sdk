package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
	GetAccessRequestDetailsValue(key string) string
	// GetAPIID returns the reference of the API on the data plane to be used
	GetAPIID() string
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplications.
	GetApplicationDetailsValue(key string) string
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetStage returns the api stage
	GetStage() string
	// GetAccessRequestData returns the map[string]interface{} of data from the request
	GetAccessRequestData() map[string]interface{}
	// GetInstanceDetails returns the 'x-agent-details' sub resource of the API Service Instance
	GetInstanceDetails() map[string]interface{}
}
