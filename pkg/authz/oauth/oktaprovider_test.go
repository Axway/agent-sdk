package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/authz/oauth/clients"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/stretchr/testify/assert"
)

const (
	testScope                   = "read:api"
	testPolicyName              = "read:api-clientcredentials"
	testClientID                = "app123"
	accessToken                 = "access-token"
	testAuthHeader              = "SSWS " + accessToken
	defaultAppTemplate          = corecfg.OktaPlaceholderMPApplicationName + "-" + corecfg.OktaPlaceholderOwningTeam + "-" + corecfg.OktaPlaceholderCredentialName
	defaultPolicyTemplate       = corecfg.OktaPlaceholderScope + "-" + corecfg.OktaPlaceholderOAuthFlow
	normalizedClientCredentials = "clientcredentials"
	normalizedAuthorizationCode = "authorizationcode"
	normalizedImplicitGrant     = "implicitgrant"
	testAuthMethodNone          = "none"
	oauthMetadataEndpoint       = "/oauth2/authorizationID/.well-known/oauth-authorization-server"
	oktaPoliciesEndpointByID    = "/api/v1/authorizationServers/authorizationID/policies"
	oktaPolicyID                = "pol-123"
	oktaPolicyEndpointByID      = "/api/v1/authorizationServers/authorizationID/policies/pol-123"
	oktaPolicyRulesEndpoint     = "/api/v1/authorizationServers/authorizationID/policies/pol-123/rules"
	oktaDeactivateEndpoint           = "/api/v1/apps/app123/lifecycle/deactivate"
	oktaDeleteEndpoint               = "/api/v1/apps/app123"
	oktaPolicyDeactivateEndpoint = "/api/v1/authorizationServers/authorizationID/policies/pol-123/lifecycle/deactivate"
	oktaPolicyActivateEndpoint   = "/api/v1/authorizationServers/authorizationID/policies/pol-123/lifecycle/activate"
	callCreateRule               = "create-rule"
	callActivate                 = "activate"
	testGroupName               = "Marketplace"
	testGroupID                 = "grp-456"
	oktaGroupsEndpoint          = "/api/v1/groups"
	oktaAppGroupEndpoint        = "/api/v1/apps/app123/groups/grp-456"
)

// oktaScriptedServer drives httptest with per-route handlers and call-count tracking.
type oktaScriptedServer struct {
	mu           sync.Mutex
	expectedAuth string
	calls        map[string]int
	routes       map[string]http.HandlerFunc
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
		if h, ok := routes[r.Method+" "+r.URL.Path]; ok {
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

type oktaPolicyItem struct {
	ID   string
	Name string
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
			buf = fmt.Appendf(buf, `{"id":%q,"name":%q}`, item.ID, item.Name)
		}
		buf = append(buf, ']')
		_, _ = w.Write(buf)
	}
}

func oktaPolicyGetHandler(policyID, name string, include []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		includeJSON, _ := json.Marshal(include)
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q,"conditions":{"clients":{"include":%s}}}`, policyID, name, string(includeJSON))
	}
}

func oktaPolicyGetHandlerWithStatus(policyID, name, status string, include []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		includeJSON, _ := json.Marshal(include)
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q,"status":%q,"conditions":{"clients":{"include":%s}}}`, policyID, name, status, string(includeJSON))
	}
}

func oktaPolicyRulesHandler(scopes ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if len(scopes) == 0 {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		scopeJSON, _ := json.Marshal(scopes)
		_, _ = fmt.Fprintf(w, `[{"conditions":{"scopes":{"include":%s}}}]`, string(scopeJSON))
	}
}

func oktaPolicyPutMustIncludeClientHandler(clientID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		conds, _ := body["conditions"].(map[string]any)
		clients, _ := conds["clients"].(map[string]any)
		include, _ := clients["include"].([]any)
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

func newIDPCredential(tsURL string) *corecfg.IDPConfiguration {
	return &corecfg.IDPConfiguration{
		MetadataURL: tsURL + oauthMetadataEndpoint,
		AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
	}
}

func assertMinCalls(t *testing.T, calls map[string]int, wantMinCalls map[string]int) {
	t.Helper()
	for key, minCount := range wantMinCalls {
		assert.GreaterOrEqual(t, calls[key], minCount, "expected at least %d calls to %s", minCount, key)
	}
}

func TestOktaPostProcessClientRegistration(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		scopes       []string
		grantType    string
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
		wantErr      bool
	}{
		"creates new per-scope policy when none exists": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q}`, oktaPolicyID, testPolicyName)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"rule-1"}`))
				},
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaPoliciesEndpointByID:  1,
				http.MethodPost + " " + oktaPoliciesEndpointByID: 1,
				http.MethodPost + " " + oktaPolicyRulesEndpoint:  1,
			},
		},
		"assigns client to existing per-scope policy": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:  oktaPolicyGetHandler(oktaPolicyID, testPolicyName, []string{}),
				http.MethodGet + " " + oktaPolicyRulesEndpoint:  oktaPolicyRulesHandler(testScope),
				http.MethodPut + " " + oktaPolicyEndpointByID:  oktaPolicyPutMustIncludeClientHandler(testClientID),
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaPoliciesEndpointByID: 1,
				http.MethodGet + " " + oktaPolicyEndpointByID:   1,
				http.MethodGet + " " + oktaPolicyRulesEndpoint:  1,
				http.MethodPut + " " + oktaPolicyEndpointByID:   1,
			},
		},
		"repairs active policy missing rule before assigning client": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:  oktaPolicyGetHandler(oktaPolicyID, testPolicyName, []string{}),
				http.MethodGet + " " + oktaPolicyRulesEndpoint:  oktaPolicyRulesHandler(),
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"rule-1"}`))
				},
				http.MethodPut + " " + oktaPolicyEndpointByID: oktaPolicyPutMustIncludeClientHandler(testClientID),
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaPoliciesEndpointByID: 1,
				http.MethodGet + " " + oktaPolicyEndpointByID:   1,
				http.MethodGet + " " + oktaPolicyRulesEndpoint:  1,
				http.MethodPost + " " + oktaPolicyRulesEndpoint: 1,
				http.MethodPut + " " + oktaPolicyEndpointByID:   1,
			},
		},
		"reactivates and repairs deactivated policy missing rule before assigning client": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID:  oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:    oktaPolicyGetHandlerWithStatus(oktaPolicyID, testPolicyName, "INACTIVE", []string{}),
				http.MethodGet + " " + oktaPolicyRulesEndpoint:   oktaPolicyRulesHandler(),
				http.MethodPost + " " + oktaPolicyActivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"rule-1"}`))
				},
				http.MethodPut + " " + oktaPolicyEndpointByID: oktaPolicyPutMustIncludeClientHandler(testClientID),
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaPoliciesEndpointByID:    1,
				http.MethodGet + " " + oktaPolicyEndpointByID:      1,
				http.MethodGet + " " + oktaPolicyRulesEndpoint:     1,
				http.MethodPost + " " + oktaPolicyActivateEndpoint: 1,
				http.MethodPost + " " + oktaPolicyRulesEndpoint:    1,
				http.MethodPut + " " + oktaPolicyEndpointByID:      1,
			},
		},
		"no scopes results in no policy calls": {
			scopes:       []string{},
			grantType:    GrantTypeClientCredentials,
			routes:       map[string]http.HandlerFunc{},
			wantMinCalls: map[string]int{},
		},
		"policy name exactly 100 chars succeeds": {
			scopes:    []string{strings.Repeat("a", 82)},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q}`, oktaPolicyID)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				},
			},
			wantErr: false,
		},
		"policy name length validation error": {
			scopes:    []string{"a-very-long-scope-name-that-when-combined-with-the-flow-name-exceeds-the-okta-100-char-limit-for-policies"},
			grantType: GrantTypeClientCredentials,
			routes:    map[string]http.HandlerFunc{},
			wantErr:   true,
		},
		"CreatePolicy error aborts registration": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				},
			},
			wantErr: true,
		},
		"CreatePolicyRule error aborts registration": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q}`, oktaPolicyID, testPolicyName)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				},
			},
			wantErr: true,
		},
		"orphaned policy cleaned up when CreatePolicyRule fails": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q}`, oktaPolicyID, testPolicyName)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
				},
				http.MethodPost + " " + oktaPolicyDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				http.MethodDelete + " " + oktaPolicyEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodPost + " " + oktaPolicyDeactivateEndpoint: 1,
				http.MethodDelete + " " + oktaPolicyEndpointByID:     1,
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ts, scripted := newOktaScriptedServer(t, testAuthHeader, tc.routes)
			defer ts.Close()
			credentialObj := newIDPCredential(ts.URL)
			clientRes := &clientMetadata{ClientID: testClientID, Scope: tc.scopes, GrantTypes: []string{tc.grantType}}

			err := oktaProvider.postProcessClientRegistration(clientRes, credentialObj, apiClient)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assertMinCalls(t, scripted.getCalls(), tc.wantMinCalls)
		})
	}
}

func TestOktaPostProcessClientUnregister(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		scopes       []string
		grantType    string
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
		wantErr      bool
	}{
		"deactivates app, deletes app, removes client from shared policy": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:   oktaPolicyGetHandler(oktaPolicyID, testPolicyName, []string{testClientID, "other-client"}),
				http.MethodPut + " " + oktaPolicyEndpointByID:   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			},
			wantMinCalls: map[string]int{
				http.MethodPost + " " + oktaDeactivateEndpoint:  1,
				http.MethodDelete + " " + oktaDeleteEndpoint:    1,
				http.MethodGet + " " + oktaPoliciesEndpointByID: 1,
				http.MethodGet + " " + oktaPolicyEndpointByID:   1,
				http.MethodPut + " " + oktaPolicyEndpointByID:   1,
			},
		},
		"policy deactivated and deleted when last client removed": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodGet + " " + oktaPoliciesEndpointByID:      oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:         oktaPolicyGetHandler(oktaPolicyID, testPolicyName, []string{testClientID}),
				http.MethodPut + " " + oktaPolicyEndpointByID:         func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				http.MethodPost + " " + oktaPolicyDeactivateEndpoint:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				http.MethodDelete + " " + oktaPolicyEndpointByID:      func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaPoliciesEndpointByID:     1,
				http.MethodGet + " " + oktaPolicyEndpointByID:       1,
				http.MethodPut + " " + oktaPolicyEndpointByID:       1,
				http.MethodPost + " " + oktaPolicyDeactivateEndpoint: 1,
				http.MethodDelete + " " + oktaPolicyEndpointByID:    1,
			},
		},
		"missing policy during unassign is skipped": {
			scopes:    []string{testScope},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
			},
			wantMinCalls: map[string]int{
				http.MethodPost + " " + oktaDeactivateEndpoint:  1,
				http.MethodDelete + " " + oktaDeleteEndpoint:    1,
				http.MethodGet + " " + oktaPoliciesEndpointByID: 1,
			},
		},
		"deactivate 404 is treated as success": {
			scopes:    []string{},
			grantType: GrantTypeClientCredentials,
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
				http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				},
			},
			wantMinCalls: map[string]int{
				http.MethodPost + " " + oktaDeactivateEndpoint: 1,
				http.MethodDelete + " " + oktaDeleteEndpoint:   1,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ts, scripted := newOktaScriptedServer(t, testAuthHeader, tc.routes)
			defer ts.Close()
			credentialObj := newIDPCredential(ts.URL)

			err := oktaProvider.postProcessClientUnregister(testClientID, credentialObj, apiClient, tc.scopes, tc.grantType)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assertMinCalls(t, scripted.getCalls(), tc.wantMinCalls)
		})
	}
}

func TestNormalizeGrantType(t *testing.T) {
	cases := map[string]struct {
		input string
		want  string
	}{
		"client_credentials":    {input: GrantTypeClientCredentials, want: normalizedClientCredentials},
		"authorization_code":    {input: GrantTypeAuthorizationCode, want: normalizedAuthorizationCode},
		"implicit":              {input: GrantTypeImplicit, want: normalizedImplicitGrant},
		"unknown strips underscores": {input: "custom_grant_type", want: "customgranttype"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeGrantType(tc.input))
		})
	}
}

func TestOktaPostProcessClientRegUsesIDPAccessToken(t *testing.T) {
	ts := httptest.NewServer(nil)
	defer ts.Close()

	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}
	clientRes := &clientMetadata{ClientID: testClientID}
	err := oktaProvider.postProcessClientRegistration(clientRes, newIDPCredential(ts.URL), apiClient)
	assert.NoError(t, err)
}

func TestOktaBaseURLFromMetadataURL(t *testing.T) {
	cases := map[string]struct {
		metadataURL string
		want        string
		wantErr     bool
	}{
		"empty metadata url returns error": {
			metadataURL: "",
			wantErr:     true,
		},
		"okta issuer url returns scheme+host": {
			metadataURL: "https://integrator-1663282.okta.com/oauth2/authorizationid/.well-known/oauth-authorization-server",
			want:        "https://integrator-1663282.okta.com",
		},
		"invalid url returns error": {
			metadataURL: "://bad-url",
			wantErr:     true,
		},
		"missing scheme/host returns error": {
			metadataURL: "/relative/path",
			wantErr:     true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
	cases := map[string]struct {
		pkceRequired bool
		want         bool
	}{
		"PKCE required true":  {pkceRequired: true, want: true},
		"PKCE required false": {pkceRequired: false, want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			props := map[string]any{oktaPKCERequired: tc.pkceRequired}
			c, err := NewClientMetadataBuilder().
				SetClientName(oktaSpa).
				SetExtraProperties(props).
				Build()
			assert.NoError(t, err)
			cm := c.(*clientMetadata)

			buf, err := json.Marshal(cm)
			assert.NoError(t, err)

			var out map[string]any
			assert.NoError(t, json.Unmarshal(buf, &out))

			val, ok := out[oktaPKCERequired]
			assert.True(t, ok)
			assert.Equal(t, tc.want, val)
		})
	}
}

func TestValidateOktaExtraProperties(t *testing.T) {
	cases := map[string]struct {
		extraProps map[string]any
		wantErr    bool
	}{
		"Valid: PKCE with browser type": {
			extraProps: map[string]any{oktaPKCERequired: true, oktaApplicationType: oktaAppTypeBrowser},
		},
		"Valid: PKCE without app type": {
			extraProps: map[string]any{oktaPKCERequired: true},
		},
		"Invalid: PKCE with service type": {
			extraProps: map[string]any{oktaPKCERequired: true, oktaApplicationType: oktaAppTypeService},
			wantErr:    true,
		},
		"Invalid: PKCE with web type": {
			extraProps: map[string]any{oktaPKCERequired: true, oktaApplicationType: oktaAppTypeWeb},
			wantErr:    true,
		},
		"Valid: No PKCE with any type": {
			extraProps: map[string]any{oktaApplicationType: oktaAppTypeService},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := (&okta{logger: log.NewFieldLogger()}).validateExtraProperties(tc.extraProps)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestValidateOktaTemplates(t *testing.T) {
	cases := map[string]struct {
		appTemplate     string
		policyTemplate     string
		wantErr     bool
		errContains string
	}{
		"default templates pass": {
			appTemplate: defaultAppTemplate,
			policyTemplate: defaultPolicyTemplate,
		},
		"app template with only one recognized placeholder passes": {
			appTemplate:    corecfg.OktaPlaceholderMPApplicationName,
			policyTemplate: defaultPolicyTemplate,
		},
		"app template with two of three recognized placeholders passes": {
			appTemplate:    corecfg.OktaPlaceholderMPApplicationName + "-" + corecfg.OktaPlaceholderCredentialName,
			policyTemplate: defaultPolicyTemplate,
		},
		"unrecognized placeholder in app template fails": {
			appTemplate:    corecfg.OktaPlaceholderMPApplicationName + "-%TYPO%-" + corecfg.OktaPlaceholderCredentialName,
			policyTemplate: defaultPolicyTemplate,
			wantErr:        true,
			errContains:    corecfg.OktaAppNameTemplateKey,
		},
		"policy template missing %SCOPE% fails": {
			appTemplate:    defaultAppTemplate,
			policyTemplate: corecfg.OktaPlaceholderOAuthFlow,
			wantErr:        true,
			errContains:    corecfg.OktaPlaceholderScope,
		},
		"policy template missing %OAUTH_FLOW% fails": {
			appTemplate:    defaultAppTemplate,
			policyTemplate: corecfg.OktaPlaceholderScope,
			wantErr:        true,
			errContains:    corecfg.OktaPlaceholderOAuthFlow,
		},
		"unrecognized placeholder in policy template fails": {
			appTemplate:    defaultAppTemplate,
			policyTemplate: corecfg.OktaPlaceholderScope + "-%CUSTOM_FLOW%",
			wantErr:        true,
			errContains:    corecfg.OktaPolicyNameTemplateKey,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := &corecfg.IDPConfiguration{
				Okta: &corecfg.OktaIDPConfiguration{
					AppNameTemplate:    tc.appTemplate,
					PolicyNameTemplate: tc.policyTemplate,
				},
			}
			err := validateOktaTemplates(cfg)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestOktaPreProcessClientRequest(t *testing.T) {
	cases := map[string]struct {
		grantTypes        []string
		responseTypes     []string
		extraProperties   map[string]any
		wantAppType       string
		wantResponseTypes []string
		wantAuthMethod    string
	}{
		"Authorization code with PKCE should use browser type": {
			grantTypes:      []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]any{oktaPKCERequired: true},
			wantAppType:     oktaAppTypeBrowser,
			wantAuthMethod:  testAuthMethodNone,
		},
		"Authorization code without PKCE should use web type": {
			grantTypes:      []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]any{oktaPKCERequired: false},
			wantAppType:     oktaAppTypeWeb,
		},
		"Client credentials should remain service type": {
			grantTypes:        []string{GrantTypeClientCredentials},
			responseTypes:     []string{},
			extraProperties:   map[string]any{},
			wantAppType:       oktaAppTypeService,
			wantResponseTypes: []string{AuthResponseToken},
		},
		"Explicit browser type should be preserved with PKCE": {
			grantTypes:      []string{GrantTypeAuthorizationCode},
			extraProperties: map[string]any{oktaApplicationType: oktaAppTypeBrowser, oktaPKCERequired: true},
			wantAppType:     oktaAppTypeBrowser,
			wantAuthMethod:  testAuthMethodNone,
		},
		"Implicit flow without PKCE should use web type": {
			grantTypes:      []string{GrantTypeImplicit},
			extraProperties: map[string]any{oktaPKCERequired: false},
			wantAppType:     oktaAppTypeWeb,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			oktaProvider := &okta{logger: log.NewFieldLogger()}
			clientReq := &clientMetadata{
				GrantTypes:      tc.grantTypes,
				ResponseTypes:   tc.responseTypes,
				extraProperties: tc.extraProperties,
			}
			_ = oktaProvider.validateExtraProperties(clientReq.extraProperties)
			oktaProvider.preProcessClientRequest(clientReq)

			appType, ok := clientReq.extraProperties[oktaApplicationType].(string)
			assert.True(t, ok)
			assert.Equal(t, tc.wantAppType, appType)

			if tc.wantResponseTypes != nil {
				assert.Equal(t, tc.wantResponseTypes, clientReq.ResponseTypes)
			}
			if tc.wantAuthMethod != "" {
				assert.Equal(t, tc.wantAuthMethod, clientReq.TokenEndpointAuthMethod)
			}
		})
	}
}

func TestOktaCreatePolicyUsesDefaultPriority(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}
	var captured []byte
	routes := map[string]http.HandlerFunc{
		http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
		http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
			captured, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"id":%q}`, oktaPolicyID)
		},
		http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		},
	}
	ts, _ := newOktaScriptedServer(t, testAuthHeader, routes)
	defer ts.Close()

	idpCfg := &corecfg.IDPConfiguration{
		MetadataURL: ts.URL + oauthMetadataEndpoint,
		AuthConfig:  &corecfg.IDPAuthConfiguration{AccessToken: accessToken},
	}
	clientRes := &clientMetadata{ClientID: testClientID, Scope: []string{testScope}, GrantTypes: []string{GrantTypeClientCredentials}}
	assert.NoError(t, oktaProvider.postProcessClientRegistration(clientRes, idpCfg, apiClient))

	var body map[string]any
	assert.NoError(t, json.Unmarshal(captured, &body))
	assert.EqualValues(t, 1, body["priority"])
}

func TestOktaDeactivateThenDelete(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	var mu sync.Mutex
	callOrder := make([]string, 0, 2)

	routes := map[string]http.HandlerFunc{
		http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callOrder = append(callOrder, "deactivate")
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		},
		http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callOrder = append(callOrder, "delete")
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		},
	}
	ts, _ := newOktaScriptedServer(t, testAuthHeader, routes)
	defer ts.Close()

	err := oktaProvider.postProcessClientUnregister(testClientID, newIDPCredential(ts.URL), apiClient, nil, GrantTypeClientCredentials)
	assert.NoError(t, err)
	assert.Equal(t, []string{"deactivate", "delete"}, callOrder)
}

func TestOktaPolicyRepairOrder(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		policyStatus  string
		wantCallOrder []string
	}{
		"active policy: rule created, no activate": {
			policyStatus:  "ACTIVE",
			wantCallOrder: []string{callCreateRule},
		},
		"inactive policy: rule created then activated": {
			policyStatus:  "INACTIVE",
			wantCallOrder: []string{callCreateRule, callActivate},
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			callOrder := make([]string, 0, 2)

			routes := map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
				http.MethodGet + " " + oktaPolicyEndpointByID:  oktaPolicyGetHandlerWithStatus(oktaPolicyID, testPolicyName, tc.policyStatus, []string{}),
				http.MethodGet + " " + oktaPolicyRulesEndpoint: oktaPolicyRulesHandler(),
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					callOrder = append(callOrder, callCreateRule)
					mu.Unlock()
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"rule-1"}`))
				},
				http.MethodPost + " " + oktaPolicyActivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					callOrder = append(callOrder, callActivate)
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
				},
				http.MethodPut + " " + oktaPolicyEndpointByID: oktaPolicyPutMustIncludeClientHandler(testClientID),
			}
			ts, _ := newOktaScriptedServer(t, testAuthHeader, routes)
			defer ts.Close()

			clientRes := &clientMetadata{ClientID: testClientID, Scope: []string{testScope}, GrantTypes: []string{GrantTypeClientCredentials}}
			assert.NoError(t, oktaProvider.postProcessClientRegistration(clientRes, newIDPCredential(ts.URL), apiClient))
			assert.Equal(t, tc.wantCallOrder, callOrder)
		})
	}
}

func TestOktaCleanupOrphanedPolicy(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		deactivateCode   int
		deleteCode       int
		wantDeleteCalled bool
	}{
		"deactivate and delete both succeed": {
			deactivateCode:   http.StatusOK,
			deleteCode:       http.StatusNoContent,
			wantDeleteCalled: true,
		},
		"deactivate fails — delete is skipped": {
			deactivateCode:   http.StatusForbidden,
			deleteCode:       http.StatusNoContent,
			wantDeleteCalled: false,
		},
		"deactivate succeeds delete fails": {
			deactivateCode:   http.StatusOK,
			deleteCode:       http.StatusForbidden,
			wantDeleteCalled: true,
		},
	}
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			deleteCalled := false
			routes := map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaPolicyDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.deactivateCode)
				},
				http.MethodDelete + " " + oktaPolicyEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					deleteCalled = true
					w.WriteHeader(tc.deleteCode)
				},
			}
			ts, _ := newOktaScriptedServer(t, testAuthHeader, routes)
			defer ts.Close()

			oktaClientObj := clients.New(apiClient, ts.URL, accessToken)
			oktaProvider.cleanupOrphanedPolicy(oktaClientObj, "authorizationID", oktaPolicyID)
			assert.Equal(t, tc.wantDeleteCalled, deleteCalled)
		})
	}
}

func TestOktaPostProcessClientUnregisterAbortOnRemoveError(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	routes := map[string]http.HandlerFunc{
		http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
		http.MethodDelete + " " + oktaDeleteEndpoint: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler([]oktaPolicyItem{{ID: oktaPolicyID, Name: testPolicyName}}),
		http.MethodGet + " " + oktaPolicyEndpointByID:  oktaPolicyGetHandler(oktaPolicyID, testPolicyName, []string{testClientID}),
		http.MethodPut + " " + oktaPolicyEndpointByID: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		},
	}
	ts, scripted := newOktaScriptedServer(t, testAuthHeader, routes)
	defer ts.Close()

	err := oktaProvider.postProcessClientUnregister(testClientID, newIDPCredential(ts.URL), apiClient, []string{testScope, "write:api"}, GrantTypeClientCredentials)
	assert.Error(t, err)

	calls := scripted.getCalls()
	assert.Equal(t, 1, calls[http.MethodGet+" "+oktaPoliciesEndpointByID], "policy list GET must stop after first scope fails")
}

func groupListHandler(id, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `[{"id":%q,"profile":{"name":%q}}]`, id, name)
	}
}

func newIDPCredentialWithGroup(tsURL, group string) *corecfg.IDPConfiguration {
	cfg := newIDPCredential(tsURL)
	cfg.Okta = &corecfg.OktaIDPConfiguration{Group: group}
	return cfg
}

func TestValidateOktaGroupExists(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")

	cases := map[string]struct {
		group   string
		routes  map[string]http.HandlerFunc
		wantErr bool
	}{
		"no group configured is a no-op": {
			group:   "",
			routes:  map[string]http.HandlerFunc{},
			wantErr: false,
		},
		"configured group found returns nil": {
			group: testGroupName,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaGroupsEndpoint: groupListHandler(testGroupID, testGroupName),
			},
			wantErr: false,
		},
		"configured group not found returns error": {
			group: testGroupName,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				},
			},
			wantErr: true,
		},
		"API error returns error": {
			group: testGroupName,
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusForbidden)
				},
			},
			wantErr: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ts, _ := newOktaScriptedServer(t, testAuthHeader, tc.routes)
			defer ts.Close()
			cfg := newIDPCredentialWithGroup(ts.URL, tc.group)
			err := validateOktaGroupExists(cfg, apiClient)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestOktaPostProcessClientRegistrationWithGroup(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
		wantErr      bool
	}{
		"group assigned after per-scope policy": {
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q}`, oktaPolicyID)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				},
				http.MethodGet + " " + oktaGroupsEndpoint:    groupListHandler(testGroupID, testGroupName),
				http.MethodPut + " " + oktaAppGroupEndpoint:  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaGroupsEndpoint:   1,
				http.MethodPut + " " + oktaAppGroupEndpoint: 1,
			},
		},
		"group not found returns error": {
			routes: map[string]http.HandlerFunc{
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
				http.MethodPost + " " + oktaPoliciesEndpointByID: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
					_, _ = fmt.Fprintf(w, `{"id":%q}`, oktaPolicyID)
				},
				http.MethodPost + " " + oktaPolicyRulesEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusCreated)
				},
				http.MethodGet + " " + oktaGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				},
			},
			wantErr: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ts, scripted := newOktaScriptedServer(t, testAuthHeader, tc.routes)
			defer ts.Close()
			cfg := newIDPCredentialWithGroup(ts.URL, testGroupName)
			clientRes := &clientMetadata{ClientID: testClientID, Scope: []string{testScope}, GrantTypes: []string{GrantTypeClientCredentials}}
			err := oktaProvider.postProcessClientRegistration(clientRes, cfg, apiClient)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assertMinCalls(t, scripted.getCalls(), tc.wantMinCalls)
		})
	}
}

func TestOktaPostProcessClientUnregisterWithGroup(t *testing.T) {
	apiClient := coreapi.NewClient(nil, "")
	oktaProvider := &okta{logger: log.NewFieldLogger()}

	cases := map[string]struct {
		routes       map[string]http.HandlerFunc
		wantMinCalls map[string]int
		wantErr      bool
	}{
		"group removed after app delete": {
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				http.MethodDelete + " " + oktaDeleteEndpoint:   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
				http.MethodGet + " " + oktaGroupsEndpoint:         groupListHandler(testGroupID, testGroupName),
				http.MethodDelete + " " + oktaAppGroupEndpoint:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
				http.MethodGet + " " + oktaPoliciesEndpointByID:   oktaPoliciesListHandler(nil),
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaGroupsEndpoint:        1,
				http.MethodDelete + " " + oktaAppGroupEndpoint:   1,
			},
		},
		"group not in Okta during unregister is skipped": {
			routes: map[string]http.HandlerFunc{
				http.MethodPost + " " + oktaDeactivateEndpoint: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
				http.MethodDelete + " " + oktaDeleteEndpoint:   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
				http.MethodGet + " " + oktaGroupsEndpoint: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`[]`))
				},
				http.MethodGet + " " + oktaPoliciesEndpointByID: oktaPoliciesListHandler(nil),
			},
			wantMinCalls: map[string]int{
				http.MethodGet + " " + oktaGroupsEndpoint: 1,
			},
			wantErr: false,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ts, scripted := newOktaScriptedServer(t, testAuthHeader, tc.routes)
			defer ts.Close()
			cfg := newIDPCredentialWithGroup(ts.URL, testGroupName)
			err := oktaProvider.postProcessClientUnregister(testClientID, cfg, apiClient, nil, GrantTypeClientCredentials)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assertMinCalls(t, scripted.getCalls(), tc.wantMinCalls)
		})
	}
}

func TestPolicyNameTemplate(t *testing.T) {
	cases := map[string]struct {
		cfg       corecfg.IDPConfig
		grantType string
		scope     string
		wantName  string
	}{
		"default template with client_credentials": {
			cfg:       &corecfg.IDPConfiguration{},
			grantType: GrantTypeClientCredentials,
			scope:     testScope,
			wantName:  testScope + "-" + normalizedClientCredentials,
		},
		"default template with authorization_code": {
			cfg:       &corecfg.IDPConfiguration{},
			grantType: GrantTypeAuthorizationCode,
			scope:     testScope,
			wantName:  testScope + "-" + normalizedAuthorizationCode,
		},
		"custom template overrides default": {
			cfg:       &corecfg.IDPConfiguration{Okta: &corecfg.OktaIDPConfiguration{PolicyNameTemplate: corecfg.OktaPlaceholderOAuthFlow + "-" + corecfg.OktaPlaceholderScope}},
			grantType: GrantTypeClientCredentials,
			scope:     testScope,
			wantName:  normalizedClientCredentials + "-" + testScope,
		},
		"implicit flow normalizes correctly": {
			cfg:       &corecfg.IDPConfiguration{},
			grantType: GrantTypeImplicit,
			scope:     testScope,
			wantName:  testScope + "-" + normalizedImplicitGrant,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			template := policyNameTemplate(tc.cfg)
			normalizedFlow := normalizeGrantType(tc.grantType)
			got := strings.NewReplacer(corecfg.OktaPlaceholderScope, tc.scope, corecfg.OktaPlaceholderOAuthFlow, normalizedFlow).Replace(template)
			assert.Equal(t, tc.wantName, got)
		})
	}
}
