package oauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	testLocalHost             = "http://localhost"
	testJwksURL               = "http://jwks"
	testToken                 = "test-token"
	testAuthServerMetadataURL = "/oauth2/authorizationID/.well-known/oauth-authorization-server"
	testMetadataURL           = "/metadata"
	testRegisterURL           = "/register/cid-1"
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

	cr, err := p.RegisterClient(tc.clientRequest)
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
	err = p.UnregisterClient(cr.GetClientID(), cr.GetRegistrationAccessToken(), s.GetUnregisterEndpoint())
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
					AccessToken: testToken,
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

func TestNewProviderOktaValidatesConfiguredGroupAndPolicyExist(t *testing.T) {
	token := testToken
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testAuthServerMetadataURL:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups":
			if r.URL.Query().Get("q") == "Marketplace" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[{"id":"00g-123","profile":{"name":"Marketplace"}}]`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/authorizationServers/authorizationID/policies":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"pol-123","name":"shane"}]`))
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/authorizationServers/authorizationID/policies/pol-123":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"pol-123","name":"shane","conditions":{"clients":{"include":[]}}}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	idpCfg := &config.IDPConfiguration{
		Name:        "test",
		Type:        TypeOkta,
		MetadataURL: srv.URL + testAuthServerMetadataURL,
		Okta:        &config.OktaIDPConfiguration{Group: "Marketplace", Policy: "shane"},
		AuthConfig: &config.IDPAuthConfiguration{
			Type:        config.AccessToken,
			AccessToken: token,
		},
	}

	p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestNewProviderOktaFailsFastWhenConfiguredGroupMissing(t *testing.T) {
	token := testToken
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == testAuthServerMetadataURL {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	idpCfg := &config.IDPConfiguration{
		Name:        "test",
		Type:        TypeOkta,
		MetadataURL: srv.URL + testAuthServerMetadataURL,
		Okta:        &config.OktaIDPConfiguration{Group: "Marketplace"},
		AuthConfig: &config.IDPAuthConfiguration{
			Type:        config.AccessToken,
			AccessToken: token,
		},
	}

	p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "configured okta group")
}

type failingHookIDP struct {
	authPrefix string
	regErr     error
	unregErr   error
}

func (f *failingHookIDP) getAuthorizationHeaderPrefix() string {
	if f.authPrefix != "" {
		return f.authPrefix
	}
	return "Bearer"
}

func (f *failingHookIDP) preProcessClientRequest(clientRequest *clientMetadata) {
	// No preprocessing needed for this mock IDP implementation
}

func (f *failingHookIDP) validateExtraProperties(extraProps map[string]interface{}) error {
	return nil
}

func (f *failingHookIDP) postProcessClientRegistration(clientRes ClientMetadata, idp config.IDPConfig, apiClient coreapi.Client) error {
	return f.regErr
}

func (f *failingHookIDP) postProcessClientUnregister(clientID string, idp config.IDPConfig, apiClient coreapi.Client) error {
	return f.unregErr
}

func TestRegisterClientRollBack(t *testing.T) {
	tests := []struct {
		name               string
		deleteResponseCode int
		deleteResponseBody string
		errorContains      string
	}{
		{
			name:               "rollback succeeds when hook fails",
			deleteResponseCode: http.StatusNoContent,
			errorContains:      "failed to complete Okta client setup",
		},
		{
			name:               "rollback failure is surfaced with manual cleanup guidance",
			deleteResponseCode: http.StatusInternalServerError,
			deleteResponseBody: "delete failed",
			errorContains:      "Manual cleanup in Okta may be required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var deleteCalls atomic.Int32
			var registerCalls atomic.Int32

			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == testMetadataURL:
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/register":
					registerCalls.Add(1)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"client_id":"cid-1","client_secret":"sec-1","registration_client_uri":"` + srv.URL + testRegisterURL + `"}`))
				case r.Method == http.MethodDelete && r.URL.Path == testRegisterURL:
					deleteCalls.Add(1)
					w.WriteHeader(tc.deleteResponseCode)
					if tc.deleteResponseBody != "" {
						_, _ = w.Write([]byte(tc.deleteResponseBody))
					}
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()

			idpCfg := &config.IDPConfiguration{
				Name:        "test",
				Type:        "generic",
				MetadataURL: srv.URL + testMetadataURL,
				AuthConfig: &config.IDPAuthConfiguration{
					Type:        config.AccessToken,
					AccessToken: testToken,
				},
			}

			pIntf, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
			assert.NoError(t, err)

			p := pIntf.(*provider)
			idpCfg.Type = TypeOkta
			p.idpType = &failingHookIDP{regErr: errors.New("post-registration hook failed")}

			clientReq := &clientMetadata{ClientName: "test"}
			cr, err := p.RegisterClient(clientReq)

			assert.Error(t, err)
			assert.Nil(t, cr)
			assert.Contains(t, err.Error(), tc.errorContains)
			assert.Equal(t, int32(1), registerCalls.Load())
			assert.Equal(t, int32(1), deleteCalls.Load())
		})
	}
}

func TestUnregisterClientDeleteHookFails(t *testing.T) {
	var deleteCalls atomic.Int32

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testMetadataURL:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
		case r.Method == http.MethodDelete && r.URL.Path == testRegisterURL:
			deleteCalls.Add(1)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	idpCfg := &config.IDPConfiguration{
		Name:        "test",
		Type:        "generic",
		MetadataURL: srv.URL + testMetadataURL,
		AuthConfig: &config.IDPAuthConfiguration{
			Type:        config.AccessToken,
			AccessToken: testToken,
		},
	}

	pIntf, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
	assert.NoError(t, err)

	p := pIntf.(*provider)
	idpCfg.Type = TypeOkta
	p.idpType = &failingHookIDP{unregErr: errors.New("cleanup failed")}

	err = p.UnregisterClient("cid-1", testToken, srv.URL+testRegisterURL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to complete provider cleanup after client unregistration")
	assert.Equal(t, int32(1), deleteCalls.Load())
}

func TestUnregisterClientCleanupAndDeleteFail(t *testing.T) {
	var deleteCalls atomic.Int32

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testMetadataURL:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
		case r.Method == http.MethodDelete && r.URL.Path == testRegisterURL:
			deleteCalls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`delete failed`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	idpCfg := &config.IDPConfiguration{
		Name:        "test",
		Type:        "generic",
		MetadataURL: srv.URL + testMetadataURL,
		AuthConfig: &config.IDPAuthConfiguration{
			Type:        config.AccessToken,
			AccessToken: testToken,
		},
	}

	pIntf, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
	assert.NoError(t, err)

	p := pIntf.(*provider)
	idpCfg.Type = TypeOkta
	p.idpType = &failingHookIDP{unregErr: errors.New("cleanup failed")}

	err = p.UnregisterClient("cid-1", testToken, srv.URL+testRegisterURL)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fully remove the Okta client")
	assert.Contains(t, err.Error(), "OAuth client deletion failed")
	assert.Equal(t, int32(1), deleteCalls.Load())
}
