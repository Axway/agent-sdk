package provisioning

// IDPCredentialData - interface for the IDP credential request
type IDPCredentialData interface {
	// GetClientID - returns client ID
	GetClientID() string
	// GetClientSecret - returns client secret
	GetClientSecret() string
	// GetScopes - returns client scopes
	GetScopes() []string
	// GetGrantTypes - returns grant types
	GetGrantTypes() []string
	// GetTokenEndpointAuthMethod - returns token auth method
	GetTokenEndpointAuthMethod() string
	// GetResponseTypes - returns token response type
	GetResponseTypes() []string
	// GetRedirectURIs - Returns redirect urls
	GetRedirectURIs() []string
	// GetJwksURI - returns JWKS uri
	GetJwksURI() string
	// GetPublicKey - returns the public key
	GetPublicKey() string
}
