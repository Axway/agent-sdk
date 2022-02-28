package provisioning

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationRequest interface {
	GetApplicationName() string // returns the name of the managed application for this credential
}
