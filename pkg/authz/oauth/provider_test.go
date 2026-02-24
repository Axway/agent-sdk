package oauth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	testLocalHost = "http://localhost"
	testJwksURL   = "http://jwks"
)

type providerTestCase struct {
	name                       string
	idpType                    string
	authHeader                 map[string]string
	authQueryParams            map[string]string
	headers                    map[string]string
	queryParams                map[string]string
	clientRequest              *clientMetadata
	expectedClient             *clientMetadata
	metadataResponseCode       int
	registrationResponseCode   int
	unRegistrationResponseCode int
	expectMetadataErr          bool
	expectRegistrationErr      bool
	expectUnRegistrationErr    bool
	authServerMetadata         *AuthorizationServerMetadata
	clientID                   string
}

func TestProvider(t *testing.T) {

	cases := []providerTestCase{
		{
			name:                 "IDP metadata bad request",
			idpType:              "generic",
			metadataResponseCode: http.StatusBadRequest,
			expectMetadataErr:    true,
		},
		{
			name:    "registration bad request",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName: "test",
			},
			metadataResponseCode:     http.StatusOK,
			registrationResponseCode: http.StatusBadRequest,
			expectRegistrationErr:    true,
		},
		{
			name:    "unregistration bad request",
			idpType: "okta",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{testLocalHost},
				JwksURI:      testJwksURL,
				GrantTypes:   []string{GrantTypeAuthorizationCode},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{testLocalHost},
				JwksURI:                 testJwksURL,
				GrantTypes:              []string{GrantTypeAuthorizationCode},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{AuthResponseCode},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]interface{}{
					"key":               "value",
					oktaApplicationType: oktaAppTypeWeb,
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusBadRequest,
			expectUnRegistrationErr:    true,
		},
		{
			name:    "successful create and delete client",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{testLocalHost},
				JwksURI:      testJwksURL,
				GrantTypes:   []string{GrantTypeImplicit},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{testLocalHost},
				JwksURI:                 testJwksURL,
				GrantTypes:              []string{GrantTypeImplicit},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{AuthResponseToken},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]interface{}{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		{
			name:            "successful client_credential",
			idpType:         "generic",
			authHeader:      map[string]string{"authHdr": "authHrdVal"},
			authQueryParams: map[string]string{"authParam": "authParamVal"},
			headers:         map[string]string{"hdr": "hrdVal"},
			queryParams:     map[string]string{"param": "paramVal"},
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{testLocalHost},
				JwksURI:      testJwksURL,
				GrantTypes:   []string{GrantTypeClientCredentials},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{testLocalHost},
				JwksURI:                 testJwksURL,
				GrantTypes:              []string{GrantTypeClientCredentials},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]interface{}{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		{
			name:    "provider with existing auth server metadata",
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName:   "test",
				RedirectURIs: []string{testLocalHost},
				JwksURI:      testJwksURL,
				GrantTypes:   []string{GrantTypeClientCredentials},
			},
			expectedClient: &clientMetadata{
				ClientName:              "test",
				RedirectURIs:            []string{testLocalHost},
				JwksURI:                 testJwksURL,
				GrantTypes:              []string{GrantTypeClientCredentials},
				TokenEndpointAuthMethod: config.ClientSecretBasic,
				ResponseTypes:           []string{},
				Scope:                   []string{"read", "write"},
				extraProperties: map[string]interface{}{
					"key": "value",
				},
			},
			clientID:                   "test-client-id",
			authServerMetadata:         &AuthorizationServerMetadata{},
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runProviderTestCase(t, tc)
		})
	}
}

// runProviderTestCase contains the subtest logic for TestProvider cases (extracted to reduce complexity)
func runProviderTestCase(t *testing.T, tc providerTestCase) {
	s := NewMockIDPServer()
	defer s.Close()
	idpCfg := &config.IDPConfiguration{
		Name: "test",
		Type: tc.idpType,
		AuthConfig: &config.IDPAuthConfiguration{
			Type:           config.Client,
			ClientID:       "test",
			ClientSecret:   "test",
			RequestHeaders: tc.authHeader,
			QueryParams:    tc.authQueryParams,
		},
		GrantType:       GrantTypeClientCredentials,
		ClientScopes:    "read,write",
		AuthMethod:      config.ClientSecretBasic,
		MetadataURL:     s.GetMetadataURL(),
		ExtraProperties: config.ExtraProperties{"key": "value"},
		RequestHeaders:  tc.headers,
		QueryParams:     tc.queryParams,
	}

	s.SetMetadataResponseCode(tc.metadataResponseCode)
	var opts []func(*providerOptions)
	if tc.authServerMetadata != nil {
		tc.authServerMetadata.TokenEndpoint = s.GetTokenURL()
		tc.authServerMetadata.RegistrationEndpoint = s.GetRegistrationEndpoint()
		opts = []func(*providerOptions){
			WithAuthServerMetadata(tc.authServerMetadata),
		}
	}
	p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second, opts...)
	if tc.expectMetadataErr {
		assert.NotNil(t, err)
		assert.Nil(t, p)
		return
	}

	assert.Nil(t, err)
	assert.NotNil(t, p)
	if tc.authServerMetadata != nil {
		authMetadata := p.GetMetadata()
		assert.Equal(t, tc.authServerMetadata.TokenEndpoint, authMetadata.TokenEndpoint)
		assert.Equal(t, tc.authServerMetadata.RegistrationEndpoint, authMetadata.RegistrationEndpoint)
	}

	s.SetRegistrationResponseCode(tc.registrationResponseCode)

	if tc.clientID != "" {
		s.SetClientID(tc.clientID)
	}

	cr, _, err := p.RegisterClient(tc.clientRequest)
	if tc.expectRegistrationErr {
		assert.NotNil(t, err)
		assert.Nil(t, cr)
		return
	}

	assert.Nil(t, err)
	assert.NotNil(t, cr)

	assert.Equal(t, tc.expectedClient.GetClientName(), cr.GetClientName())
	assert.NotEmpty(t, cr.GetClientID())
	assert.NotEmpty(t, cr.GetClientSecret())
	assert.Equal(t, strings.Join(tc.expectedClient.GetGrantTypes(), ","), strings.Join(cr.GetGrantTypes(), ","))
	assert.Equal(t, tc.expectedClient.GetTokenEndpointAuthMethod(), cr.GetTokenEndpointAuthMethod())
	assert.Equal(t, strings.Join(tc.expectedClient.GetResponseTypes(), ","), strings.Join(cr.GetResponseTypes(), ","))
	assert.Equal(t, strings.Join(tc.expectedClient.GetRedirectURIs(), ","), strings.Join(cr.GetRedirectURIs(), ","))
	assert.Equal(t, strings.Join(tc.expectedClient.GetScopes(), ","), strings.Join(cr.GetScopes(), ","))
	assert.Equal(t, tc.expectedClient.GetJwksURI(), cr.GetJwksURI())
	assert.Equal(t, len(tc.expectedClient.GetExtraProperties()), len(cr.GetExtraProperties()))
	if tc.clientID != "" {
		assert.Equal(t, s.GetUnregisterEndpoint(), cr.GetRegistrationClientURI())
	}
	s.SetRegistrationResponseCode(tc.unRegistrationResponseCode)
	err = p.UnregisterClient(cr.GetClientID(), cr.GetRegistrationAccessToken(), s.GetUnregisterEndpoint(), nil)
	if tc.expectUnRegistrationErr {
		assert.NotNil(t, err)
		return
	}
	assertHeaders(t, tc.authHeader, s.GetTokenRequestHeaders())
	assertQueryParams(t, tc.authQueryParams, s.GetTokenQueryParams())
	assertHeaders(t, tc.headers, s.GetRequestHeaders())
	assertQueryParams(t, tc.queryParams, s.GetQueryParams())

	assert.Nil(t, err)
}

func TestNewProviderValidatesExtraProperties(t *testing.T) {
	// Test that validation happens during provider construction
	s := NewMockIDPServer()
	defer s.Close()

	tests := []struct {
		name            string
		idpType         string
		extraProperties map[string]interface{}
		expectError     bool
		errorContains   string
	}{
		{
			name:    "Valid Okta provider with PKCE and browser type",
			idpType: TypeOkta,
			extraProperties: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeBrowser,
			},
			expectError: false,
		},
		{
			name:    "Invalid Okta provider with PKCE and service type",
			idpType: TypeOkta,
			extraProperties: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeService,
			},
			expectError:   true,
			errorContains: "pkce_required",
		},
		{
			name:            "Valid generic provider",
			idpType:         "generic",
			extraProperties: map[string]interface{}{},
			expectError:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idpCfg := &config.IDPConfiguration{
				Name:            "test",
				Type:            tc.idpType,
				MetadataURL:     s.GetMetadataURL(),
				ExtraProperties: tc.extraProperties,
				AuthConfig: &config.IDPAuthConfiguration{
					Type:        config.AccessToken,
					AccessToken: "test-token",
				},
			}

			provider, err := NewProvider(idpCfg, &config.TLSConfiguration{}, "", 10*time.Second)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}
