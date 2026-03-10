package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	groupsEndpoint        = " /api/v1/groups"
	appGroupsEndpoint     = " /api/v1/apps/app123/groups"
	oauthMetadataEndpoint = "/oauth2/ausxna8tgvHrw8UrN697/.well-known/oauth-authorization-server"
	accessToken           = "access-token"
)

type oktaScriptedServer struct {
	mu            sync.Mutex
	expectedAuth  string
	calls         map[string]int
	routes        map[string]http.HandlerFunc
	strictRouting bool
}

func newOktaScriptedServer(t *testing.T, expectedAuth string, routes map[string]http.HandlerFunc) (*httptest.Server, *oktaScriptedServer) {
	t.Helper()

	s := &oktaScriptedServer{
		expectedAuth: expectedAuth,
		calls:        make(map[string]int),
		routes:       routes,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.calls[r.Method+" "+r.URL.Path]++
		s.mu.Unlock()

		if expectedAuth != "" {
			if got := r.Header.Get("Authorization"); got != expectedAuth {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("bad auth"))
				return
			}
		}

		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))

	return ts, s
}

func (s *oktaScriptedServer) getCalls() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.calls))
	for k, v := range s.calls {
		out[k] = v
	}
	return out
}

func TestOktaPostProcessClientRegistration(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{}

	type testCase struct {
		name         string
		extraProps   map[string]interface{}
		routes       map[string]http.HandlerFunc
		wantCreated  map[string]string
		wantMinCalls map[string]int
	}

	cases := []testCase{
		{
			name: "provisions group + policy/rule",
			extraProps: map[string]interface{}{
				"group":        "MyAppUsers",
				"createPolicy": true,
				"authServerId": "default",
				"policyTemplate": map[string]interface{}{
					"name":        "AutoPolicy-Test",
					"description": "Auto-created",
					"rule": map[string]interface{}{
						"name":       "AutoRule-Test",
						"conditions": map[string]interface{}{"grantTypes": map[string]interface{}{"include": []string{"authorization_code"}}},
						"actions":    map[string]interface{}{"token": map[string]interface{}{"accessTokenLifetime": 3600}},
					},
				},
			},
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					if q := r.URL.Query().Get("q"); q != "MyAppUsers" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte("unexpected q"))
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[{"id":"00g-other","profile":{"name":"Other"}},{"id":"00g-123","profile":{"name":"MyAppUsers"}}]`))
				},
				http.MethodPost + appGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				},
				http.MethodPost + " /api/v1/authorizationServers/default/policies": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"pol-123"}`))
				},
				http.MethodPost + " /api/v1/authorizationServers/default/policies/pol-123/rules": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"rule-456"}`))
				},
			},
			wantCreated: map[string]string{
				"oktaGroupId":  "00g-123",
				"oktaPolicyId": "pol-123",
				"oktaRuleId":   "rule-456",
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                                                  1,
				http.MethodPost + appGroupsEndpoint:                                              1,
				http.MethodPost + " /api/v1/authorizationServers/default/policies":               1,
				http.MethodPost + " /api/v1/authorizationServers/default/policies/pol-123/rules": 1,
			},
		},
		{
			name:         "no actions when group/policy disabled",
			extraProps:   map[string]interface{}{"createPolicy": false},
			routes:       map[string]http.HandlerFunc{},
			wantCreated:  map[string]string{},
			wantMinCalls: map[string]int{},
		},
		{
			name: "creates group when not found",
			extraProps: map[string]interface{}{
				"group":        "NewGroup",
				"createPolicy": false,
			},
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					if q := r.URL.Query().Get("q"); q != "NewGroup" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte("unexpected q"))
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				},
				http.MethodPost + groupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"00g-new"}`))
				},
				http.MethodPost + appGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				},
			},
			wantCreated: map[string]string{
				"oktaGroupId": "00g-new",
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:     1,
				http.MethodPost + groupsEndpoint:    1,
				http.MethodPost + appGroupsEndpoint: 1,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			expectedAuth := "SSWS access-token"
			ts, scripted := newOktaScriptedServer(t, expectedAuth, tc.routes)
			defer ts.Close()

			credentialObj := &corecfg.IDPConfiguration{
				MetadataURL: ts.URL + oauthMetadataEndpoint,
				AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
			}
			clientRes := &clientMetadata{ClientID: "app123"}

			created, err := oktaProvider.postProcessClientRegistration(clientRes, tc.extraProps, credentialObj, apiClient)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantCreated, created)

			calls := scripted.getCalls()
			for key, minCount := range tc.wantMinCalls {
				assert.GreaterOrEqual(t, calls[key], minCount, "expected at least %d calls to %s", minCount, key)
			}
		})
	}
}

func TestOktaPostProcessClientUnreg(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{}

	type testCase struct {
		name         string
		extraProps   map[string]interface{}
		agentDetails map[string]string
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
	}

	cases := []testCase{
		{
			name: "cleans up policy/rule and group assignment",
			extraProps: map[string]interface{}{
				"authServerId": "default",
			},
			agentDetails: map[string]string{
				"oktaPolicyId": "pol-123",
				"oktaRuleId":   "rule-456",
				"oktaGroupId":  "00g-123",
			},
			routes: map[string]http.HandlerFunc{
				http.MethodDelete + " /api/v1/authorizationServers/default/policies/pol-123/rules/rule-456": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + " /api/v1/authorizationServers/default/policies/pol-123": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + " /api/v1/apps/app123/groups/00g-123": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodDelete + " /api/v1/authorizationServers/default/policies/pol-123/rules/rule-456": 1,
				http.MethodDelete + " /api/v1/authorizationServers/default/policies/pol-123":                1,
				http.MethodDelete + " /api/v1/apps/app123/groups/00g-123":                                   1,
			},
		},
		{
			name:         "no cleanup when agent details are empty",
			extraProps:   map[string]interface{}{"authServerId": "default"},
			agentDetails: map[string]string{},
			routes:       map[string]http.HandlerFunc{},
			wantMinCalls: map[string]int{},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			expectedAuth := "SSWS access-token"
			ts, scripted := newOktaScriptedServer(t, expectedAuth, tc.routes)
			defer ts.Close()

			credentialObj := &corecfg.IDPConfiguration{
				MetadataURL: ts.URL + oauthMetadataEndpoint,
				AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
			}

			err := oktaProvider.postProcessClientUnregister("app123", tc.agentDetails, tc.extraProps, credentialObj, apiClient)
			assert.NoError(t, err)

			calls := scripted.getCalls()
			for key, minCount := range tc.wantMinCalls {
				assert.GreaterOrEqual(t, calls[key], minCount, "expected at least %d calls to %s", minCount, key)
			}
		})
	}
}

func TestOktaPostProcessClientRegUsesIDPAccessToken(t *testing.T) {
	// We only need to prove that auth.accessToken from the IDP config is used to
	// enable Okta post-processing (i.e. the hook does not early-return).
	// Avoid making any Okta API calls by disabling all optional actions.
	ts := httptest.NewServer(nil)
	defer ts.Close()

	credentialObj := &corecfg.IDPConfiguration{
		MetadataURL: ts.URL + oauthMetadataEndpoint,
		AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
	}
	extraProps := map[string]interface{}{
		"createPolicy": false,
	}

	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{}
	clientRes := &clientMetadata{ClientID: "app123"}
	created, err := oktaProvider.postProcessClientRegistration(clientRes, extraProps, credentialObj, apiClient)
	assert.NoError(t, err)
	assert.NotNil(t, created)
}

func TestOktaBaseURLFromMetadataURL(t *testing.T) {
	cases := []struct {
		name        string
		metadataURL string
		want        string
		wantErr     bool
	}{
		{
			name:        "empty metadata url returns empty",
			metadataURL: "",
			want:        "",
			wantErr:     false,
		},
		{
			name:        "okta issuer url returns scheme+host",
			metadataURL: "https://integrator-1663282.okta.com/oauth2/ausxna8tgvHrw8UrN697/.well-known/oauth-authorization-server",
			want:        "https://integrator-1663282.okta.com",
			wantErr:     false,
		},
		{
			name:        "invalid url returns error",
			metadataURL: "://bad-url",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "missing scheme/host returns error",
			metadataURL: "/relative/path",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := oktaBaseURLFromMetadataURL(tc.metadataURL)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestOktaPKCERequired(t *testing.T) {
	cases := []struct {
		name          string
		pkceRequired  bool
		expectedValue bool
	}{
		{
			name:          "PKCE required true",
			pkceRequired:  true,
			expectedValue: true,
		},
		{
			name:          "PKCE required false",
			pkceRequired:  false,
			expectedValue: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := map[string]interface{}{
				oktaPKCERequired: tc.pkceRequired,
			}
			c, err := NewClientMetadataBuilder().
				SetClientName(oktaSpa).
				SetExtraProperties(props).
				Build()
			assert.Nil(t, err)
			cm := c.(*clientMetadata)

			buf, err := json.Marshal(cm)
			assert.Nil(t, err)
			assert.NotNil(t, buf)

			var out map[string]interface{}
			err = json.Unmarshal(buf, &out)
			assert.Nil(t, err)

			// Should be a boolean, not a string
			val, ok := out[oktaPKCERequired]
			assert.True(t, ok)
			assert.IsType(t, tc.expectedValue, val)
			assert.Equal(t, tc.expectedValue, val)
		})
	}
}

func TestValidateOktaExtraProperties(t *testing.T) {
	cases := []struct {
		name        string
		extraProps  map[string]interface{}
		expectError bool
	}{
		{
			name: "Valid: PKCE with browser type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeBrowser,
			},
			expectError: false,
		},
		{
			name: "Valid: PKCE without app type",
			extraProps: map[string]interface{}{
				oktaPKCERequired: true,
			},
			expectError: false,
		},
		{
			name: "Invalid: PKCE with service type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeService,
			},
			expectError: true,
		},
		{
			name: "Invalid: PKCE with web type",
			extraProps: map[string]interface{}{
				oktaPKCERequired:    true,
				oktaApplicationType: oktaAppTypeWeb,
			},
			expectError: true,
		},
		{
			name: "Valid: No PKCE with any type",
			extraProps: map[string]interface{}{
				oktaApplicationType: oktaAppTypeService,
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oktaProvider := &okta{}
			err := oktaProvider.validateExtraProperties(tc.extraProps)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOktaPreProcessClientRequest(t *testing.T) {
	cases := []struct {
		name                  string
		grantTypes            []string
		responseTypes         []string
		extraProperties       map[string]interface{}
		expectedAppType       string
		expectedResponseTypes []string
		expectedAuthMethod    string
	}{
		{
			name:       "Authorization code with PKCE should use browser type",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: true,
			},
			expectedAppType:    oktaAppTypeBrowser,
			expectedAuthMethod: "none",
		},
		{
			name:       "Authorization code without PKCE should use web type",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: false,
			},
			expectedAppType: oktaAppTypeWeb,
		},
		{
			name:                  "Client credentials should remain service type",
			grantTypes:            []string{GrantTypeClientCredentials},
			responseTypes:         []string{},
			extraProperties:       map[string]interface{}{},
			expectedAppType:       oktaAppTypeService,
			expectedResponseTypes: []string{AuthResponseToken},
		},
		{
			name:       "Explicit browser type should be preserved with PKCE",
			grantTypes: []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]interface{}{
				oktaApplicationType: oktaAppTypeBrowser,
				oktaPKCERequired:    true,
			},
			expectedAppType:    oktaAppTypeBrowser,
			expectedAuthMethod: "none",
		},
		{
			name:       "Implicit flow without PKCE should use web type",
			grantTypes: []string{GrantTypeImplicit},
			extraProperties: map[string]interface{}{
				oktaPKCERequired: false,
			},
			expectedAppType: oktaAppTypeWeb,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oktaProvider := &okta{}
			clientReq := &clientMetadata{
				GrantTypes:      tc.grantTypes,
				ResponseTypes:   tc.responseTypes,
				extraProperties: tc.extraProperties,
			}

			// Simulate validation step which sets defaults (as happens in NewProvider)
			_ = oktaProvider.validateExtraProperties(clientReq.extraProperties)

			oktaProvider.preProcessClientRequest(clientReq)

			appType, ok := clientReq.extraProperties[oktaApplicationType].(string)
			assert.True(t, ok)
			assert.Equal(t, tc.expectedAppType, appType)

			if tc.expectedResponseTypes != nil {
				assert.Equal(t, tc.expectedResponseTypes, clientReq.ResponseTypes)
			}

			if tc.expectedAuthMethod != "" {
				assert.Equal(t, tc.expectedAuthMethod, clientReq.TokenEndpointAuthMethod)
			}
		})
	}
}
