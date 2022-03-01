package provisioning

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationRequest interface {
	GetManagedApplicationName() string      // returns the name of the managed application for this credential
	GetApplicationName() string             // returns the name of the application on the dataplane
	GetProperty(key string) (string, error) // return the value based on teh key
}
