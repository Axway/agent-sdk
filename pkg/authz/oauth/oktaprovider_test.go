package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	oauthMetadataEndpoint    = "/oauth2/authorizationID/.well-known/oauth-authorization-server"
	oktaPoliciesEndpointByID = " /api/v1/authorizationServers/authorizationID/policies"
	accessToken              = "access-token"
	oktaPolicyID             = "pol-123"
	oktaGroupID              = "00g-123"
	oktaPolicyEndpointByID   = " /api/v1/authorizationServers/authorizationID/policies/pol-123"
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

type oktaGroupItem struct {
	ID   string
	Name string
}

type oktaPolicyItem struct {
	ID   string
	Name string
}

func oktaGroupsSearchHandler(wantQ string, items []oktaGroupItem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if wantQ != "" {
			if q := r.URL.Query().Get("q"); q != wantQ {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("unexpected q"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		buf := make([]byte, 0, 256)
		buf = append(buf, '[')
		for i, item := range items {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, []byte(fmt.Sprintf(`{"id":%q,"profile":{"name":%q}}`, item.ID, item.Name))...)
		}
		buf = append(buf, ']')
		_, _ = w.Write(buf)
	}
}

func oktaAssignGroupHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}
}

func oktaPoliciesListHandler(items []oktaPolicyItem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		buf := make([]byte, 0, 256)
		buf = append(buf, '[')
		for i, item := range items {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, []byte(fmt.Sprintf(`{"id":%q,"name":%q}`, item.ID, item.Name))...)
		}
		buf = append(buf, ']')
		_, _ = w.Write(buf)
	}
}

func oktaPolicyGetHandler(policyID, policyName string, include []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		includeJSON, _ := json.Marshal(include)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"id":%q,"name":%q,"conditions":{"clients":{"include":%s}}}`, policyID, policyName, string(includeJSON))))
	}
}

func oktaPolicyPutMustIncludeClientHandler(clientID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		conds, _ := body["conditions"].(map[string]interface{})
		clients, _ := conds["clients"].(map[string]interface{})
		include, _ := clients["include"].([]interface{})
		for _, v := range include {
			if s, ok := v.(string); ok && s == clientID {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("client id not included"))
	}
}

func newIDPCredential(tsURL, group, policy string) *corecfg.IDPConfiguration {
	credentialObj := &corecfg.IDPConfiguration{
		MetadataURL: tsURL + oauthMetadataEndpoint,
		AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
	}
	if strings.TrimSpace(group) != "" || strings.TrimSpace(policy) != "" {
		credentialObj.Okta = &corecfg.OktaIDPConfiguration{Group: group, Policy: policy}
	}
	return credentialObj
}

func assertMinCalls(t *testing.T, calls map[string]int, wantMinCalls map[string]int) {
	t.Helper()
	for key, minCount := range wantMinCalls {
		assert.GreaterOrEqual(t, calls[key], minCount, "expected at least %d calls to %s", minCount, key)
	}
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
		wantErr      bool
	}

	cases := []testCase{
		{
			name:       "assigns existing group + records existing policy",
			oktaGroup:  "MyAppUsers",
			oktaPolicy: "MarketplacePolicy",
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint:                              oktaGroupsSearchHandler("MyAppUsers", []oktaGroupItem{{ID: "00g-other", Name: "Other"}, {ID: oktaGroupIDExisting, Name: "MyAppUsers"}}),
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: oktaAssignGroupHandler(),
				http.MethodGet + oktaPoliciesEndpointByID:                    oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: "MarketplacePolicy"}}),
				http.MethodGet + oktaPolicyEndpointByID:                      oktaPolicyGetHandler(oktaPolicyID, "MarketplacePolicy", []string{}),
				http.MethodPut + oktaPolicyEndpointByID:                      oktaPolicyPutMustIncludeClientHandler("app123"),
			},
			wantCreated: map[string]string{
				"oktaGroupId":  oktaGroupIDExisting,
				"oktaPolicyId": oktaPolicyID,
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                              1,
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: 1,
				http.MethodGet + oktaPoliciesEndpointByID:                    1,
				http.MethodGet + oktaPolicyEndpointByID:                      1,
				http.MethodPut + oktaPolicyEndpointByID:                      1,
			},
		},
		{
			name:       "infers auth server id for policy",
			oktaGroup:  "MyAppUsers",
			oktaPolicy: "MarketplacePolicy",
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint:                              oktaGroupsSearchHandler("", []oktaGroupItem{{ID: oktaGroupIDExisting, Name: "MyAppUsers"}}),
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: oktaAssignGroupHandler(),
				http.MethodGet + oktaPoliciesEndpointByID:                    oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: "MarketplacePolicy"}}),
				http.MethodGet + oktaPolicyEndpointByID:                      oktaPolicyGetHandler(oktaPolicyID, "MarketplacePolicy", []string{}),
				http.MethodPut + oktaPolicyEndpointByID:                      oktaPolicyPutMustIncludeClientHandler("app123"),
			},
			wantCreated: map[string]string{
				"oktaGroupId":  oktaGroupIDExisting,
				"oktaPolicyId": oktaPolicyID,
			},
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint:                              1,
				http.MethodPut + appGroupsEndpointBase + oktaGroupIDExisting: 1,
				http.MethodGet + oktaPoliciesEndpointByID:                    1,
				http.MethodGet + oktaPolicyEndpointByID:                      1,
				http.MethodPut + oktaPolicyEndpointByID:                      1,
			},
		},
		{
			name:         "no actions when group/policy disabled",
			routes:       map[string]http.HandlerFunc{},
			wantCreated:  map[string]string{},
			wantMinCalls: map[string]int{},
		},
		{
			name:      "error when group not found",
			oktaGroup: "NewGroup",
			routes: map[string]http.HandlerFunc{
				http.MethodGet + groupsEndpoint: oktaGroupsSearchHandler("NewGroup", nil),
			},
			wantErr: true,
			wantMinCalls: map[string]int{
				http.MethodGet + groupsEndpoint: 1,
			},
		},
		{
			name:       "error when policy not found",
			oktaPolicy: "MissingPolicy",
			routes: map[string]http.HandlerFunc{
				http.MethodGet + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: "pol-other", Name: "OtherPolicy"}}),
			},
			wantErr: true,
			wantMinCalls: map[string]int{
				http.MethodGet + oktaPoliciesEndpointByID: 1,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			expectedAuth := "SSWS access-token"
			ts, scripted := newOktaScriptedServer(t, expectedAuth, tc.routes)
			defer ts.Close()
			credentialObj := newIDPCredential(ts.URL, tc.oktaGroup, tc.oktaPolicy)
			clientRes := &clientMetadata{ClientID: "app123"}

			created, err := oktaProvider.postProcessClientRegistration(clientRes, credentialObj, apiClient)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantCreated, created)
			}

			assertMinCalls(t, scripted.getCalls(), tc.wantMinCalls)
		})
	}
}

func TestOktaPostProcessClientUnreg(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{}

	type testCase struct {
		name         string
		agentDetails map[string]string
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
	}

	cases := []testCase{
		{
			name: "unassigns group when agent details include group id",
			agentDetails: map[string]string{
				"oktaGroupId": oktaGroupID,
			},
			routes: map[string]http.HandlerFunc{
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodDelete + appGroupsEndpointBase + oktaGroupID: 1,
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
			metadataURL: "https://integrator-1663282.okta.com/oauth2/authorizationid/.well-known/oauth-authorization-server",
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
