package oauth

// Provider types
const (
	Generic ProviderType = iota + 1
	Okta
	KeyCloak
)

// Provider type string const
const (
	TypeGeneric  = "generic"
	TypeOkta     = "okta"
	TypeKeycloak = "keycloak"
)

// IDP Auth type string const
const (
	IDPAuthTypeAccessToken = "accessToken"
	IDPAuthTypeClient      = "client"
)

const (
	defaultServerName = "OAuth server"

	hdrAuthorization = "Authorization"
	hdrContentType   = "Content-Type"

	mimeApplicationFormURLEncoded = "application/x-www-form-urlencoded"
	mimeApplicationJSON           = "application/json"

	grantAuthorizationCode = "authorization_code"
	grantImplicit          = "implicit"
	grantClientCredentials = "client_credentials"

	authResponseToken = "token"
	authResponseCode  = "code"

	assertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	metaGrantType           = "grant_type"
	metaClientID            = "client_id"
	metaClientSecret        = "client_secret"
	metaScope               = "scope"
	metaClientAssertionType = "client_assertion_type"
	metaClientAssertion     = "client_assertion"
)
