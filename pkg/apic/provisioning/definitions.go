package provisioning

// enums

// credentialType - the type fo credential
type credentialType int

const (
	// credentialTypeOAuth - OAuth credentials
	credentialTypeOAuth credentialType = iota + 1
	// credentialTypeAPIKey - APIKey credentials
	credentialTypeAPIKey
)

func (c credentialType) String() string {
	return map[credentialType]string{
		credentialTypeOAuth:  "OAuth",
		credentialTypeAPIKey: "API Key",
	}[c]
}

// RequestType - the type of credential request being sent
type RequestType int

const (
	// RequestTypeProvision - provision new credentials
	RequestTypeProvision RequestType = iota + 1
	// RequestTypeRenew - renew existing credentials
	RequestTypeRenew
)

// status - the status of the request
type status int

const (
	// Success - request was successful
	Success status = iota + 1
	// Failed - request failed
	Failed
)

// interfaces

// Provisioning - interface to be implemented by agents for access provisioning
type Provisioning interface {
	ApplicationRequestProvision(applicationRequest ApplicationRequest) (status RequestStatus)
	ApplicationRequestDeprovision(applicationRequest ApplicationRequest) (status RequestStatus)
	AccessRequestProvision(accessRequest AccessRequest) (status RequestStatus)
	AccessRequestDeprovision(accessRequest AccessRequest) (status RequestStatus)
	CredentialProvision(credentialRequest CredentialRequest) (status RequestStatus, credentails Credential)
	CredentialDeprovision(credentialRequest CredentialRequest) (status RequestStatus)
}