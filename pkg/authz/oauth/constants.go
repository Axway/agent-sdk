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
	IDPAuthTypeAccessToken             = "accessToken"
	IDPAuthTypeClient                  = "client"
	IDPAuthTypeClientSecretBasic       = "client_secret_basic"
	IDPAuthTypeClientSecretPost        = "client_secret_post"
	IDPAuthTypeClientSecretJWT         = "client_secret_jwt"
	IDPAuthTypePrivateKeyJWT           = "private_key_jwt"
	IDPAuthTypeTLSClientAuth           = "tls_client_auth"
	IDPAuthTypeSelfSignedTLSClientAuth = "self_signed_tls_client_auth"
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
