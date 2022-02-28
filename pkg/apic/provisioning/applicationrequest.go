package provisioning

type ApplicationRequest interface {
	GetApplicationName() string // returns the name of the managed application for this credential
}
