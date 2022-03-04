package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetAPIID returns the reference of the API on the data plane to be used
	GetAPIID() string
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplications.
	GetApplicationDetailsValue(key string) interface{}
	// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
	GetAccessRequestDetailsValue(key string) interface{}
	// GetStage returns the api stage
	GetStage() string
}
