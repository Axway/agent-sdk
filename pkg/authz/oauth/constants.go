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

const (
	defaultServerName = "OAuth server"

	hdrAuthorization = "Authorization"
	hdrContentType   = "Content-Type"

	mimeApplicationFormURLEncoded = "application/x-www-form-urlencoded"
	mimeApplicationJSON           = "application/json"

	GrantTypeAuthorizationCode     = "authorization_code"
	GrantTypeImplicit              = "implicit"
	GrantTypeClientCredentials     = "client_credentials"
	GrantTypeRefreshToken          = "refresh_token"
	GrantTypeSaml2Bearer           = "urn:ietf:params:oauth:grant-type:saml2-bearer"
	GrantTypePassword              = "password"
	GrantTypeIntegratedWindowsAuth = "iwa:ntlm" // NTLM
	GrantTypeJWTBearer             = "urn:ietf:params:oauth:grant-type:jwt-bearer"

	AuthResponseToken = "token"
	AuthResponseCode  = "code"

	assertionTypeJWT = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

	metaGrantType           = "grant_type"
	metaClientID            = "client_id"
	metaClientSecret        = "client_secret"
	metaScope               = "scope"
	metaClientAssertionType = "client_assertion_type"
	metaClientAssertion     = "client_assertion"

	TLSClientAuthSubjectDN = "tls_client_auth_subject_dn"
	TLSClientAuthSanDNS    = "tls_client_auth_san_dns"
	TLSClientAuthSanEmail  = "tls_client_auth_san_email"
	TLSClientAuthSanIP     = "tls_client_auth_san_ip"
	TLSClientAuthSanURI    = "tls_client_auth_san_uri"
)
