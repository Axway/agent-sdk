package provisioning

// AccessRequest - interface for agents to use to get necessary access request details
type AccessRequest interface {
	GetApplicationName() string         // returns the name of the managed application for this credential
	GetAPIID() string                   // returns the reference of the API on the data plane to be used
	GetProperty(key string) interface{} // returns the property value for the key sent in
}
