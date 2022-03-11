package provisioning

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationRequest interface {
	// GetAgentDetailsValue return the value based on the key
	GetAgentDetailsValue(key string) interface{}
	// GetManagedApplicationName returns the name of the managed application for this credential
	GetManagedApplicationName() string
	// GetTeamName gets the owning team name for the managed application
	GetTeamName() string
}
