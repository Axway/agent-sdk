package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

const (
	groupsEndpoint           = " /api/v1/groups"
	appGroupsEndpointBase    = " /api/v1/apps/app123/groups/"
	oktaGroupIDExisting      = "00g-123"
	oktaGroupIDNew           = "00g-new"
	oauthMetadataEndpoint    = "/oauth2/ausxna8tgvHrw8UrN697/.well-known/oauth-authorization-server"
	oktaPoliciesEndpoint     = " /api/v1/authorizationServers/default/policies/"
	oktaPoliciesEndpointByID = " /api/v1/authorizationServers/ausxna8tgvHrw8UrN697/policies/"
	accessToken              = "access-token"
	oktaPolicyID             = "pol-123"
	oktaRuleID               = "rule-456"
	oktaGroupID              = "00g-123"
	rules                    = "/rules/"
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
		oktaGroup    string
		oktaPolicy   string
		routes       map[string]http.HandlerFunc
		wantCreated  map[string]string
		wantMinCalls map[string]int
	}

	cases := []testCase{
		{
			name:       "provisions group + policy/rule",
			oktaGroup:  "MyAppUsers",
			oktaPolicy: `{"authServerId":"default","policyTemplate":{"name":"AutoPolicy-Test","description":"Auto-created","rule":{"name":"AutoRule-Test","conditions":{"grantTypes":{"include":["authorization_code"]}},"actions":{"token":{"accessTokenLifetime":3600}}}}}`,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					if q := r.URL.Query().Get("q"); q != "MyAppUsers" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte("unexpected q"))
						return
					}
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":"00g-other","profile":{"name":"Other"}},{"id":"%s","profile":{"name":"MyAppUsers"}}]`, oktaGroupIDExisting)))
				},
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: func(w http.ResponseWriter, r *http.Request) {
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
				"oktaGroupId":  oktaGroupIDExisting,
				"oktaPolicyId": oktaPolicyID,
				"oktaRuleId":   oktaRuleID,
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                                                  1,
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting:                     1,
				http.MethodPost + " /api/v1/authorizationServers/default/policies":               1,
				http.MethodPost + " /api/v1/authorizationServers/default/policies/pol-123/rules": 1,
			},
		},
		{
			name:       "infers auth server id for policy/rule",
			oktaGroup:  "MyAppUsers",
			oktaPolicy: `{"policyTemplate":{"name":"AutoPolicy-Test","description":"Auto-created","rule":{"name":"AutoRule-Test","conditions":{"grantTypes":{"include":["authorization_code"]}},"actions":{"token":{"accessTokenLifetime":3600}}}}}`,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(fmt.Sprintf(`[{"id":"%s","profile":{"name":"MyAppUsers"}}]`, oktaGroupIDExisting)))
				},
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				},
				http.MethodPost + " /api/v1/authorizationServers/ausxna8tgvHrw8UrN697/policies": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, oktaPolicyID)))
				},
				http.MethodPost + " /api/v1/authorizationServers/ausxna8tgvHrw8UrN697/policies/pol-123/rules": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, oktaRuleID)))
				},
			},
			wantCreated: map[string]string{
				"oktaGroupId":  oktaGroupIDExisting,
				"oktaPolicyId": oktaPolicyID,
				"oktaRuleId":   oktaRuleID,
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                                                               1,
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting:                                  1,
				http.MethodPost + " /api/v1/authorizationServers/ausxna8tgvHrw8UrN697/policies":               1,
				http.MethodPost + " /api/v1/authorizationServers/ausxna8tgvHrw8UrN697/policies/pol-123/rules": 1,
			},
		},
		{
			name:         "no actions when group/policy disabled",
			routes:       map[string]http.HandlerFunc{},
			wantCreated:  map[string]string{},
			wantMinCalls: map[string]int{},
		},
		{
			name:      "creates group when not found",
			oktaGroup: "NewGroup",
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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"%s"}`, oktaGroupIDNew)))
				},
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDNew: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				},
			},
			wantCreated: map[string]string{
				"oktaGroupId": oktaGroupIDNew,
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                         1,
				http.MethodPost + groupsEndpoint:                        1,
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDNew: 1,
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
			if tc.oktaGroup != "" || tc.oktaPolicy != "" {
				credentialObj.Okta = &corecfg.OktaIDPConfiguration{Group: tc.oktaGroup, Policy: tc.oktaPolicy}
			}
			clientRes := &clientMetadata{ClientID: "app123"}

			created, err := oktaProvider.postProcessClientRegistration(clientRes, credentialObj, apiClient)
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
		oktaPolicy   string
		agentDetails map[string]string
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
	}

	cases := []testCase{
		{
			name:       "cleans up policy/rule and group assignment",
			oktaPolicy: `{"authServerId":"default","policyTemplate":{"name":"AutoPolicy-Test"}}`,
			agentDetails: map[string]string{
				"oktaPolicyId": oktaPolicyID,
				"oktaRuleId":   oktaRuleID,
				"oktaGroupId":  oktaGroupID,
			},
			routes: map[string]http.HandlerFunc{
				http.MethodDelete + oktaPoliciesEndpoint + oktaPolicyID + rules + oktaRuleID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + oktaPoliciesEndpoint + oktaPolicyID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodDelete + oktaPoliciesEndpoint + oktaPolicyID + rules + oktaRuleID: 1,
				http.MethodDelete + oktaPoliciesEndpoint + oktaPolicyID:                      1,
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID:                      1,
			},
		},
		{
			name:       "infers auth server id for cleanup",
			oktaPolicy: `{"policyTemplate":{"name":"AutoPolicy-Test"}}`,
			agentDetails: map[string]string{
				"oktaPolicyId": oktaPolicyID,
				"oktaRuleId":   oktaRuleID,
				"oktaGroupId":  oktaGroupID,
			},
			routes: map[string]http.HandlerFunc{
				http.MethodDelete + oktaPoliciesEndpointByID + oktaPolicyID + rules + oktaRuleID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + oktaPoliciesEndpointByID + oktaPolicyID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodDelete + oktaPoliciesEndpointByID + oktaPolicyID + rules + oktaRuleID: 1,
				http.MethodDelete + oktaPoliciesEndpointByID + oktaPolicyID:                      1,
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID:                          1,
			},
		},
		{
			name:         "no cleanup when agent details are empty",
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
			if tc.oktaPolicy != "" {
				credentialObj.Okta = &corecfg.OktaIDPConfiguration{Policy: tc.oktaPolicy}
			}

			err := oktaProvider.postProcessClientUnregister("app123", tc.agentDetails, credentialObj, apiClient)
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

	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{}
	clientRes := &clientMetadata{ClientID: "app123"}
	created, err := oktaProvider.postProcessClientRegistration(clientRes, credentialObj, apiClient)
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
