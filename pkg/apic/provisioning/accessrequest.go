package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplications.
	GetApplicationDetailsValue(key string) string
	// GetApplicationName returns the name of the managed application for this credential
	GetApplicationName() string
	// GetID returns the ID of the resource for the request
	GetID() string
	// IsTransferring returns flag indicating the AccessRequest is for migrating referenced AccessRequest
	IsTransferring() bool
	// GetReferencedID returns the ID of the referenced AccessRequest resource for the request
	GetReferencedID() string
	// GetAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the AccessRequest.
	GetAccessRequestDetailsValue(key string) string
	// GetReferencedAccessRequestDetailsValue returns a value found on the 'x-agent-details' sub resource of the referenced AccessRequest.
	GetReferencedAccessRequestDetailsValue(key string) string
	// GetAccessRequestData returns the map[string]interface{} of data from the request
	GetAccessRequestData() map[string]interface{}
	// GetAccessRequestProvisioningData returns the interface{} of data from the provisioning response
	GetAccessRequestProvisioningData() interface{}
	// GetInstanceDetails returns the 'x-agent-details' sub resource of the API Service Instance
	GetInstanceDetails() map[string]interface{}
	// GetQuota returns the quota from within the access request
	GetQuota() Quota
}
