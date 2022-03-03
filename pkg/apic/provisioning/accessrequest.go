package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetAPIID returns the reference of the API on the data plane to be used
	GetAPIID() string
	// GetApplicationDetails returns a value found on the 'x-agent-details' sub resource of the ManagedApplications.
	GetApplicationDetails(key string) interface{}
	// GetAccessRequestDetails returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
	GetAccessRequestDetails(key string) interface{}
	// GetStage returns the api stage
	GetStage() string
}
