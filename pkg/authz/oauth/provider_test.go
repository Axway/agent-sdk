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

	scopeOpenID   = "openid"
	scopeProfile  = "profile"
	scopeEmail    = "email"
	scopeWriteAPI = "write:api"

	blacklistOpenIDProfile = "openid,profile"
)

type providerTestCase struct {
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

	cases := map[string]providerTestCase{
		"IDP metadata bad request": {
			idpType:              "generic",
			metadataResponseCode: http.StatusBadRequest,
			expectMetadataErr:    true,
		},
		"registration bad request": {
			idpType: "generic",
			clientRequest: &clientMetadata{
				ClientName: "test",
			},
			metadataResponseCode:     http.StatusOK,
			registrationResponseCode: http.StatusBadRequest,
			expectRegistrationErr:    true,
		},
		"unregistration bad request": {
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
				extraProperties: map[string]any{
					"key":               "value",
					oktaApplicationType: oktaAppTypeWeb,
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusBadRequest,
			expectUnRegistrationErr:    true,
		},
		"successful create and delete client": {
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
				extraProperties: map[string]any{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		"successful client_credential": {
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
				extraProperties: map[string]any{
					"key": "value",
				},
			},
			metadataResponseCode:       http.StatusOK,
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
		"provider with existing auth server metadata": {
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
				extraProperties: map[string]any{
					"key": "value",
				},
			},
			clientID:                   "test-client-id",
			authServerMetadata:         &AuthorizationServerMetadata{},
			registrationResponseCode:   http.StatusCreated,
			unRegistrationResponseCode: http.StatusNoContent,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
	err = p.UnregisterClient(cr.GetClientID(), cr.GetRegistrationAccessToken(), s.GetUnregisterEndpoint(), cr.GetScopes(), "")
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

	tests := map[string]struct {
		idpType         string
		extraProperties map[string]any
		expectError     bool
		errorContains   string
	}{
		"Valid Okta provider with PKCE and browser type": {
			idpType: TypeOkta,
			extraProperties: map[string]any{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeBrowser,
			},
			expectError: false,
		},
		"Invalid Okta provider with PKCE and service type": {
			idpType: TypeOkta,
			extraProperties: map[string]any{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeService,
			},
			expectError:   true,
			errorContains: "pkce_required",
		},
		"Valid generic provider": {
			idpType:         "generic",
			extraProperties: map[string]any{},
			expectError:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

			provider, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)

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

func (f *failingHookIDP) validateExtraProperties(extraProps map[string]any) error {
	return nil
}

func (f *failingHookIDP) postProcessClientRegistration(clientRes ClientMetadata, idp config.IDPConfig, apiClient coreapi.Client) error {
	return f.regErr
}

func (f *failingHookIDP) postProcessClientUnregister(clientID string, idp config.IDPConfig, apiClient coreapi.Client, scopes []string, grantType string) error {
	return f.unregErr
}

func TestRegisterClientRollBack(t *testing.T) {
	tests := map[string]struct {
		deleteResponseCode int
		deleteResponseBody string
		errorContains      string
	}{
		"rollback succeeds when hook fails": {
			deleteResponseCode: http.StatusNoContent,
			errorContains:      "failed to complete Okta client setup",
		},
		"rollback failure is surfaced with manual cleanup guidance": {
			deleteResponseCode: http.StatusInternalServerError,
			deleteResponseBody: "delete failed",
			errorContains:      "Manual cleanup in Okta may be required",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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

	err = p.UnregisterClient("cid-1", testToken, srv.URL+testRegisterURL, nil, "")

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

	err = p.UnregisterClient("cid-1", testToken, srv.URL+testRegisterURL, nil, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fully remove the Okta client")
	assert.Contains(t, err.Error(), "OAuth client deletion failed")
	assert.Equal(t, int32(1), deleteCalls.Load())
}

func TestRegisterClientOktaScopesRequest(t *testing.T) {
	cases := map[string]struct {
		regResponseBody string
		requestScopes   Scopes
		wantPolicyCalls int32
	}{
		"response omits scopes — policy creation runs from request scopes": {
			regResponseBody: `{"client_id":"` + testClientID + `","client_secret":"sec-1"}`,
			requestScopes:   Scopes{testScope},
			wantPolicyCalls: 1,
		},
		"response includes scopes — policy creation runs normally": {
			regResponseBody: `{"client_id":"` + testClientID + `","client_secret":"sec-1","scope":"` + testScope + `","grant_types":["client_credentials"]}`,
			requestScopes:   Scopes{testScope},
			wantPolicyCalls: 1,
		},
		"no scopes in request — policy creation skipped": {
			regResponseBody: `{"client_id":"` + testClientID + `","client_secret":"sec-1"}`,
			requestScopes:   nil,
			wantPolicyCalls: 0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var policyCalls atomic.Int32
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, ".well-known"):
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"issuer":"` + srv.URL + `","token_endpoint":"` + srv.URL + `/token","registration_endpoint":"` + srv.URL + `/register","authorization_endpoint":"` + srv.URL + `/auth"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/register":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(tc.regResponseBody))
				case r.Method == http.MethodGet && r.URL.Path == oktaPoliciesEndpointByID:
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				case r.Method == http.MethodPost && r.URL.Path == oktaPoliciesEndpointByID:
					policyCalls.Add(1)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"` + oktaPolicyID + `"}`))
				case r.Method == http.MethodPost && r.URL.Path == oktaPolicyRulesEndpoint:
					w.WriteHeader(http.StatusCreated)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()

			idpCfg := &config.IDPConfiguration{
				Name:        "test",
				Type:        TypeOkta,
				MetadataURL: srv.URL + testAuthServerMetadataURL,
				AuthConfig: &config.IDPAuthConfiguration{
					Type:        config.AccessToken,
					AccessToken: testToken,
				},
			}
			p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
			assert.NoError(t, err)

			clientReq := &clientMetadata{
				ClientName: "test-app",
				GrantTypes: []string{GrantTypeClientCredentials},
				Scope:      tc.requestScopes,
			}
			cr, err := p.RegisterClient(clientReq)
			assert.NoError(t, err)
			assert.NotNil(t, cr)
			assert.Equal(t, tc.wantPolicyCalls, policyCalls.Load())
		})
	}
}

func TestNewProviderValidatesOktaGroup(t *testing.T) {
	cases := map[string]struct {
		groupsCode  int
		groupsBody  string
		wantErr     bool
		errContains string
	}{
		"group found — NewProvider succeeds": {
			groupsCode: http.StatusOK,
			groupsBody: `[{"id":"grp-1","profile":{"name":"` + testGroupName + `"}}]`,
		},
		"group not found — NewProvider fails": {
			groupsCode:  http.StatusOK,
			groupsBody:  `[]`,
			wantErr:     true,
			errContains: "configured Okta group",
		},
		"groups API error — NewProvider fails": {
			groupsCode:  http.StatusForbidden,
			groupsBody:  `forbidden`,
			wantErr:     true,
			errContains: "Okta group",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var ts *httptest.Server
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, ".well-known"):
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"issuer":"` + ts.URL + `","token_endpoint":"` + ts.URL + `/token","registration_endpoint":"` + ts.URL + `/register","authorization_endpoint":"` + ts.URL + `/auth"}`))
				case r.URL.Path == oktaGroupsEndpoint:
					w.WriteHeader(tc.groupsCode)
					_, _ = w.Write([]byte(tc.groupsBody))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer ts.Close()

			idpCfg := &config.IDPConfiguration{
				Name:        "test",
				Type:        TypeOkta,
				MetadataURL: ts.URL + testAuthServerMetadataURL,
				Okta:        &config.OktaIDPConfiguration{Group: testGroupName},
				AuthConfig: &config.IDPAuthConfiguration{
					Type:        config.AccessToken,
					AccessToken: testToken,
				},
			}
			provider, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 10*time.Second)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestFilterScopeBlacklist(t *testing.T) {
	cases := map[string]struct {
		scopes    []string
		blacklist string
		want      []string
	}{
		"removes blacklisted scopes": {
			scopes:    []string{scopeOpenID, scopeProfile, testScope, scopeWriteAPI},
			blacklist: blacklistOpenIDProfile,
			want:      []string{testScope, scopeWriteAPI},
		},
		"empty blacklist returns all scopes": {
			scopes:    []string{scopeOpenID, testScope},
			blacklist: "",
			want:      []string{scopeOpenID, testScope},
		},
		"blacklist with whitespace is trimmed": {
			scopes:    []string{scopeOpenID, testScope},
			blacklist: " openid , profile ",
			want:      []string{testScope},
		},
		"no scopes match blacklist returns all": {
			scopes:    []string{testScope, scopeWriteAPI},
			blacklist: blacklistOpenIDProfile,
			want:      []string{testScope, scopeWriteAPI},
		},
		"all scopes blacklisted returns empty": {
			scopes:    []string{scopeOpenID, scopeProfile},
			blacklist: blacklistOpenIDProfile,
			want:      []string{},
		},
		"nil scopes returns nil": {
			scopes:    nil,
			blacklist: scopeOpenID,
			want:      nil,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := filterScopeBlacklist(tc.scopes, tc.blacklist)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetSupportedScopesAppliesBlacklist(t *testing.T) {
	cases := map[string]struct {
		cfg        config.IDPConfig
		rawScopes  []string
		wantScopes []string
	}{
		"Okta config with blacklist filters scopes": {
			cfg: &config.IDPConfiguration{
				Type: TypeOkta,
				Okta: &config.OktaIDPConfiguration{ScopeBlacklist: "openid,profile"},
			},
			rawScopes:  []string{"openid", "profile", "read:api"},
			wantScopes: []string{"read:api"},
		},
		"Okta config with default blacklist filters defaults": {
			cfg:        &config.IDPConfiguration{Type: TypeOkta, Okta: &config.OktaIDPConfiguration{}},
			rawScopes:  []string{"openid", "profile", "email", "read:api"},
			wantScopes: []string{"read:api"},
		},
		"non-Okta config returns scopes unfiltered": {
			cfg:        &config.IDPConfiguration{Type: "generic"},
			rawScopes:  []string{"openid", "profile", "read:api"},
			wantScopes: []string{"openid", "profile", "read:api"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := &provider{
				cfg: tc.cfg,
				authServerMetadata: &AuthorizationServerMetadata{
					ScopesSupported: tc.rawScopes,
				},
			}
			assert.Equal(t, tc.wantScopes, p.GetSupportedScopes())
		})
	}
}
