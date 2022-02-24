package provisioning

type AccessRequest interface {
	GetApplicationName() string
	GetAPIID() credentialType
	GetProperty(string) interface{}
}
