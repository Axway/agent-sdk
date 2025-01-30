package provisioning

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationRequest interface {
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication
	GetApplicationDetailsValue(key string) string
	// GetManagedApplicationName returns the name of the managed application for this credential
	GetManagedApplicationName() string
	// GetTeamName gets the owning team name for the managed application
	GetTeamName() string
	// GetConsumerOrgID gets the ID of the owning consumer org for the managed application
	GetConsumerOrgID() string
	// GetID returns the ID of the resource for the request
	GetID() string
}

// ApplicationRequest - interface for agents to use to get necessary application request details
type ApplicationProfileRequest interface {
	// GetApplicationProfileVales returns the map[string]interface{} of data from the request
	GetApplicationProfileData() map[string]interface{}
	// GetApplicationDetailsValue returns a value found on the 'x-agent-details' sub resource of the ManagedApplication
	GetApplicationDetailsValue(key string) string
	// GetManagedApplicationName returns the name of the managed application for this credential
	GetManagedApplicationName() string
	// GetTeamName gets the owning team name for the managed application
	GetTeamName() string
	// GetConsumerOrgID gets the ID of the owning consumer org for the managed application
	GetConsumerOrgID() string
	// GetID returns the ID of the resource for the request
	GetID() string
}
