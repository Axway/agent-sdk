package provisioning

// default names of ARD and CRDs
const (
	APIKeyARD         = "api-key"
	APIKeyCRD         = "api-key"
	OAuthSecretCRD    = "oauth-secret"
	OAuthPublicKeyCRD = "oauth-public-key"
	OAuthIDPCRD       = "oauth-idp"

	OauthClientID        = "clientId"
	OauthClientSecret    = "clientSecret"
	OauthPublicKey       = "publicKey"
	OauthGrantType       = "grantType"
	OauthTokenAuthMethod = "tokenAuthMethod"
	OauthScopes          = "scopes"
	OauthRedirectURIs    = "redirectURLs"
	OauthJwksURI         = "jwksURI"
	OauthJwks            = "jwks"

	IDPTokenURL = "idpTokenURL"

	APIKey = "apiKey"

	CredExpDetail = "Agent: CredentialExpired"
)

// RequestType - the type of credential request being sent
type RequestType int

const (
	// RequestTypeProvision - provision new credentials
	RequestTypeProvision RequestType = iota + 1
	// RequestTypeRenew - renew existing credentials
	RequestTypeRenew
)

// String returns the string value of the RequestType enum
func (c RequestType) String() string {
	return map[RequestType]string{
		RequestTypeProvision: "provision",
		RequestTypeRenew:     "renew",
	}[c]
}

// Status - the Status of the request
type Status int

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

// State is the provisioning state
type State int

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

// CredentialAction - the Action the agent needs to take for this CredentialUpdate request
type CredentialAction int

const (
	// Enable - enable a credential
	Enable CredentialAction = iota + 1
	// Suspend - disable a credential
	Suspend
	// Rotate - create a new secret for a credential
	Rotate
	// Expire - mark the credential as expired
	Expire
)

// String returns the string value of the CredentialAction
func (c CredentialAction) String() string {
	return map[CredentialAction]string{
		Enable:  "Enable",
		Suspend: "Suspend",
		Rotate:  "Rotate",
		Expire:  "Expire",
	}[c]
}

// Provisioning - interface to be implemented by agents for access provisioning
type Provisioning interface {
	AccessRequestDeprovision(AccessRequest) RequestStatus
	AccessRequestProvision(AccessRequest) (RequestStatus, AccessData)
	ApplicationRequestDeprovision(ApplicationRequest) RequestStatus
	ApplicationRequestProvision(ApplicationRequest) RequestStatus
	CredentialDeprovision(CredentialRequest) RequestStatus
	CredentialProvision(CredentialRequest) (RequestStatus, Credential)
	CredentialUpdate(CredentialRequest) (RequestStatus, Credential)
}
