package provisioning

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationRequest interface {
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication
	GetApplicationDetailsValue(key string) string
	// GetManagedApplicationName returns the name of the managed application for this credential
	GetManagedApplicationName() string
	// GetTeamName gets the owning team name for the managed application
	GetTeamName() string
	// GetID returns the ID of the resource for the request
	GetID() string
}
