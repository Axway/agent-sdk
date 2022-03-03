package provisioning

// enums

// CredentialType - the type fo credential
type CredentialType int

const (
	// OAuthCredential - OAuth credentials
	OAuthCredential CredentialType = iota + 1
	// APIKeyCredential - APIKey credentials
	APIKeyCredential
)

func (c CredentialType) String() string {
	return map[CredentialType]string{
		OAuthCredential:  "OAuth",
		APIKeyCredential: "API Key",
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

func (c RequestType) String() string {
	return map[RequestType]string{
		RequestTypeProvision: "provision",
		RequestTypeRenew:     "renew",
	}[c]
}

// Status - the Status of the request
type Status int

type State int

const (
	// Success - request was successful
	Success Status = iota + 1
	// Error - request failed
	Error
	// Pending - request is pending
	Pending
)

// String returns the string value of the Status
func (c Status) String() string {
	return map[Status]string{
		Success: "Success",
		Error:   "Error",
		Pending: "Pending",
	}[c]
}

const (
	// Provision - state is waiting to provision
	Provision = iota + 1
	// Deprovision - state is waiting to deprovision
	Deprovision
)

// String returns the string value of the State
func (c State) String() string {
	return map[State]string{
		Provision:   "Provision",
		Deprovision: "Deprovision",
	}[c]
}

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
