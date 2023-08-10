package oauth

type MTLSEndPointAlias struct {
	TokenEndpoint         string `json:"token_endpoint,omitempty"`
	RegistrationEndpoint  string `json:"registration_endpoint,omitempty"`
	IntrospectionEndpoint string `json:"introspection_endpoint,omitempty"`
	RevocationEndpoint    string `json:"revocation_endpoint,omitempty"`
}

// AuthorizationServerMetadata - OAuth metadata from IdP
type AuthorizationServerMetadata struct {
	Issuer string `json:"issuer,omitempty"`

	AuthorizationEndpoint              string `json:"authorization_endpoint,omitempty"`
	TokenEndpoint                      string `json:"token_endpoint,omitempty"`
	RegistrationEndpoint               string `json:"registration_endpoint,omitempty"`
	JwksURI                            string `json:"jwks_uri,omitempty"`
	IntrospectionEndpoint              string `json:"introspection_endpoint,omitempty"`
	RevocationEndpoint                 string `json:"revocation_endpoint,omitempty"`
	EndSessionEndpoint                 string `json:"end_session_endpoint,omitempty"`
	DeviceAuthorizationEndpoint        string `json:"device_authorization_endpoint,omitempty"`
	PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint,omitempty"`

	ResponseTypesSupported                    []string `json:"response_types_supported,omitempty"`
	ResponseModesSupported                    []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported                       []string `json:"grant_types_supported,omitempty"`
	SubjectTypeSupported                      []string `json:"subject_types_supported,omitempty"`
	ScopesSupported                           []string `json:"scopes_supported,omitempty"`
	TokenEndpointAuthMethodSupported          []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	ClaimsSupported                           []string `json:"claims_supported,omitempty"`
	CodeChallengeMethodsSupported             []string `json:"code_challenge_methods_supported,omitempty"`
	IntrospectionEndpointAuthMethodsSupported []string `json:"introspection_endpoint_auth_methods_supported,omitempty"`
	RevocationEndpointAuthMethodsSupported    []string `json:"revocation_endpoint_auth_methods_supported,omitempty"`

	RequestParameterSupported              bool     `json:"request_parameter_supported,omitempty"`
	RequestObjectSigningAlgValuesSupported []string `json:"request_object_signing_alg_values_supported,omitempty"`

	MTLSEndPointAlias *MTLSEndPointAlias `json:"mtls_endpoint_aliases,omitempty"`
}
