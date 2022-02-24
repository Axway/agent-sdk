package provisioning

type CredentialRequest interface {
	GetApplicationName() string
	GetCredentialType() credentialType
	GetProperty(string) interface{}
	GetRequestType() string
}
