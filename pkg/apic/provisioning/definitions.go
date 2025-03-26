package provisioning

// default names of ARD and CRDs
const (
	APIKeyARD         = "api-key"
	BasicAuthARD      = "http-basic"
	APIKeyCRD         = "api-key"
	BasicAuthCRD      = "http-basic"
	OAuthSecretCRD    = "oauth-secret"
	OAuthPublicKeyCRD = "oauth-public-key"
	OAuthIDPCRD       = "oauth-idp"
	ExternalCRD       = "external-crd"
	MtlsCRD           = "mtls"
	MtlsARD           = "mtls"

	OauthClientID            = "clientId"
	OauthClientSecret        = "clientSecret"
	OauthPublicKey           = "publicKey"
	OauthGrantType           = "grantType"
	OauthTokenAuthMethod     = "tokenAuthMethod"
	OauthScopes              = "scopes"
	OauthRedirectURIs        = "redirectURLs"
	OauthJwksURI             = "jwksURI"
	OauthJwks                = "jwks"
	OauthCertificate         = "certificate"
	OauthCertificateMetadata = "certificateMetadata"
	OauthTLSAuthSANDNS       = "tlsClientAuthSanDNS"
	OauthTLSAuthSANEmail     = "tlsClientAuthSanEmail"
	OauthTLSAuthSANIP        = "tlsClientAuthSanIP"
	OauthTLSAuthSANURI       = "tlsClientAuthSanURI"
	OauthRegistrationToken   = "registration"

	IDPTokenURL = "idpTokenURL"

	APIKey = "apiKey"

	// MTLS
	Mtls       = "mtls"
	XAxwayMTLS = "x-axway-mtls"

	BasicAuthUsername = "username"
	BasicAuthPassword = "password"

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

type ApplicationProvisioner interface {
	ApplicationRequestDeprovision(ApplicationRequest) RequestStatus
	ApplicationRequestProvision(ApplicationRequest) RequestStatus
}

type ApplicationProfileProvisioner interface {
	ApplicationProfileRequestProvision(ApplicationProfileRequest) RequestStatus
}

type AccessProvisioner interface {
	AccessRequestDeprovision(AccessRequest) RequestStatus
	AccessRequestProvision(AccessRequest) (RequestStatus, AccessData)
}

type CredentialProvisioner interface {
	CredentialDeprovision(CredentialRequest) RequestStatus
	CredentialProvision(CredentialRequest) (RequestStatus, Credential)
	CredentialUpdate(CredentialRequest) (RequestStatus, Credential)
}

type CustomCredential interface {
	GetIgnoredCredentialTypes() []string
}

// ExpiredCredentialAction - the action to take on an expired credential
type ExpiredCredentialAction int

const (
	// DeprovisionExpiredCredential - deprovision expired credentials
	DeprovisionExpiredCredential ExpiredCredentialAction = iota + 1
)

// String returns the string value of the RequestType enum
func (c ExpiredCredentialAction) String() string {
	return map[ExpiredCredentialAction]string{
		DeprovisionExpiredCredential: "deprovision",
	}[c]
}

// String returns the string value of the RequestType enum
func ExpiredCredentialActionFromString(action string) ExpiredCredentialAction {
	if val, ok := map[string]ExpiredCredentialAction{
		"deprovision": DeprovisionExpiredCredential,
	}[action]; ok {
		return val
	}
	return 0
}
