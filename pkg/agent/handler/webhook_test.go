package handler

import (
	"slices"
	"testing"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

// mockSubResourceCarrier is a minimal subResourceCarrier for testing the webhook helpers without
// depending on a concrete apiserver resource type.
type mockSubResourceCarrier struct {
	subResources map[string]interface{}
}

func newMockSubResourceCarrier() *mockSubResourceCarrier {
	return &mockSubResourceCarrier{subResources: map[string]interface{}{}}
}

func (m *mockSubResourceCarrier) GetSubResource(key string) interface{} {
	return m.subResources[key]
}

func (m *mockSubResourceCarrier) SetSubResource(key string, resource interface{}) {
	m.subResources[key] = resource
}

func TestWebhookDispatchedFor(t *testing.T) {
	tests := []struct {
		name      string
		dispatch  string
		operation string
		expected  bool
	}{
		{name: "matches dispatched operation", dispatch: webhookOperationProvision, operation: webhookOperationProvision, expected: true},
		{name: "does not match different operation", dispatch: webhookOperationProvision, operation: webhookOperationDeprovision, expected: false},
		{name: "nothing dispatched yet", dispatch: "", operation: webhookOperationProvision, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMockSubResourceCarrier()
			if tc.dispatch != "" {
				markWebhookDispatched(h, tc.dispatch)
			}
			assert.Equal(t, tc.expected, webhookDispatchedFor(h, tc.operation))
		})
	}
}

func TestWebhookDispatchedOperation(t *testing.T) {
	tests := []struct {
		name     string
		marks    []string
		expected string
	}{
		{name: "nothing dispatched yet", marks: nil, expected: ""},
		{name: "single dispatch recorded", marks: []string{webhookOperationProvision}, expected: webhookOperationProvision},
		{name: "later dispatch overwrites earlier one", marks: []string{webhookOperationProvision, webhookOperationDeprovision}, expected: webhookOperationDeprovision},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMockSubResourceCarrier()
			for _, op := range tc.marks {
				markWebhookDispatched(h, op)
			}

			assert.Equal(t, tc.expected, webhookDispatchedOperation(h))

			details, ok := h.GetSubResource(defs.XAgentDetails).(map[string]interface{})
			if tc.expected == "" {
				assert.False(t, ok)
			} else {
				assert.True(t, ok)
				assert.Equal(t, tc.expected, details[webhookDispatchDetailKey])
			}
		})
	}
}

func TestWebhookDetailsValue(t *testing.T) {
	tests := []struct {
		name           string
		webhookDetails map[string]interface{}
		key            string
		expected       string
	}{
		{name: "no webhook details subresource", key: webhookStatusKey, expected: ""},
		{
			name:           "reads status",
			webhookDetails: map[string]interface{}{webhookStatusKey: webhookStatusSuccess, webhookMessageKey: "all good"},
			key:            webhookStatusKey,
			expected:       webhookStatusSuccess,
		},
		{
			name:           "reads message",
			webhookDetails: map[string]interface{}{webhookStatusKey: webhookStatusSuccess, webhookMessageKey: "all good"},
			key:            webhookMessageKey,
			expected:       "all good",
		},
		{
			name:           "missing key",
			webhookDetails: map[string]interface{}{webhookStatusKey: webhookStatusSuccess},
			key:            "missing-key",
			expected:       "",
		},
		{
			name:           "non-string value is ignored",
			webhookDetails: map[string]interface{}{webhookStatusKey: 1},
			key:            webhookStatusKey,
			expected:       "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMockSubResourceCarrier()
			if tc.webhookDetails != nil {
				h.SetSubResource(defs.XWebhookDetails, tc.webhookDetails)
			}
			assert.Equal(t, tc.expected, webhookDetailsValue(h, tc.key))
		})
	}
}

func TestMirrorWebhookDetails(t *testing.T) {
	tests := []struct {
		name                 string
		existingAgentDetails map[string]interface{}
		hasWebhookDetails    bool
		webhookDetails       map[string]interface{}
		expectedReturn       bool
		expectedAgentDetails map[string]interface{}
	}{
		{
			name:              "no webhook details to mirror",
			hasWebhookDetails: false,
			expectedReturn:    false,
		},
		{
			name:              "empty webhook details map",
			hasWebhookDetails: true,
			webhookDetails:    map[string]interface{}{},
			expectedReturn:    false,
		},
		{
			name:                 "mirrors into new agent details",
			hasWebhookDetails:    true,
			webhookDetails:       map[string]interface{}{"clientId": "abc123"},
			expectedReturn:       true,
			expectedAgentDetails: map[string]interface{}{"clientId": "abc123"},
		},
		{
			name:                 "merges into existing agent details without dropping other keys",
			existingAgentDetails: map[string]interface{}{"existing": "value"},
			hasWebhookDetails:    true,
			webhookDetails:       map[string]interface{}{"clientId": "abc123"},
			expectedReturn:       true,
			expectedAgentDetails: map[string]interface{}{"existing": "value", "clientId": "abc123"},
		},
		{
			name:                 "does not let a webhook payload clobber the agent's reserved keys",
			existingAgentDetails: map[string]interface{}{webhookDispatchDetailKey: "update"},
			hasWebhookDetails:    true,
			webhookDetails: map[string]interface{}{
				webhookStatusKey:         webhookStatusSuccess,
				webhookMessageKey:        "all good",
				webhookDispatchDetailKey: "provision",
				"clientId":               "abc123",
			},
			expectedReturn: true,
			expectedAgentDetails: map[string]interface{}{
				webhookDispatchDetailKey: "update",
				"clientId":               "abc123",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newMockSubResourceCarrier()
			if tc.existingAgentDetails != nil {
				h.SetSubResource(defs.XAgentDetails, tc.existingAgentDetails)
			}
			if tc.hasWebhookDetails {
				h.SetSubResource(defs.XWebhookDetails, tc.webhookDetails)
			}

			assert.Equal(t, tc.expectedReturn, mirrorWebhookDetails(h))

			if tc.expectedAgentDetails != nil {
				agentDetails, ok := h.GetSubResource(defs.XAgentDetails).(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tc.expectedAgentDetails, agentDetails)
			}
		})
	}
}

func TestWebhookReservedDetailKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		reserved bool
	}{
		{name: "dispatch detail key is reserved", key: webhookDispatchDetailKey, reserved: true},
		{name: "status key is reserved", key: webhookStatusKey, reserved: true},
		{name: "message key is reserved", key: webhookMessageKey, reserved: true},
		{name: "business data key is not reserved", key: "clientId", reserved: false},
		{name: "similarly named key is not reserved", key: "status_details", reserved: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.reserved, slices.Contains(webhookReservedDetailKey, tc.key))
		})
	}
}

func TestNewWebhookApplicationRequest(t *testing.T) {
	tests := []struct {
		name     string
		app      provManagedApp
		expected webhookApplicationRequest
	}{
		{
			name: "builds request from managed app",
			app: provManagedApp{
				id:             "app-id",
				managedAppName: "my-app",
				teamName:       "my-team",
				consumerOrgID:  "org-id",
				data:           map[string]interface{}{"foo": "bar"},
			},
			expected: webhookApplicationRequest{
				Operation:              webhookOperationProvision,
				ID:                     "app-id",
				ManagedApplicationName: "my-app",
				TeamName:               "my-team",
				ConsumerOrgID:          "org-id",
				AgentDetails:           map[string]interface{}{"foo": "bar"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newWebhookApplicationRequest(webhookOperationProvision, tc.app))
		})
	}
}

func TestNewWebhookApplicationProfileRequest(t *testing.T) {
	tests := []struct {
		name     string
		profile  provManagedAppProfile
		expected webhookApplicationProfileRequest
	}{
		{
			name: "builds request from managed app profile",
			profile: provManagedAppProfile{
				id:                "profile-id",
				managedAppName:    "my-app",
				profileDefinition: "my-profile-def",
				teamName:          "my-team",
				consumerOrgID:     "org-id",
				attributes:        map[string]interface{}{"attr": "val"},
				data:              map[string]interface{}{"foo": "bar"},
			},
			expected: webhookApplicationProfileRequest{
				Operation:                    webhookOperationDeprovision,
				ID:                           "profile-id",
				ManagedApplicationName:       "my-app",
				ApplicationProfileDefinition: "my-profile-def",
				TeamName:                     "my-team",
				ConsumerOrgID:                "org-id",
				Attributes:                   map[string]interface{}{"attr": "val"},
				ApplicationDetails:           map[string]interface{}{"foo": "bar"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newWebhookApplicationProfileRequest(webhookOperationDeprovision, tc.profile))
		})
	}
}

// mockQuota is a minimal provisioning.Quota implementation for testing.
type mockQuota struct {
	limit    int64
	interval string
}

func (q *mockQuota) GetInterval() provisioning.QuotaInterval { return 0 }
func (q *mockQuota) GetIntervalString() string               { return q.interval }
func (q *mockQuota) GetLimit() int64                         { return q.limit }
func (q *mockQuota) GetPlanName() string                     { return "" }

func TestNewWebhookAccessRequest(t *testing.T) {
	tests := []struct {
		name     string
		request  provAccReq
		expected webhookAccessRequest
	}{
		{
			name: "without quota, not transferring",
			request: provAccReq{
				id:               "ar-id",
				managedApp:       "my-app",
				requestData:      map[string]interface{}{"req": "data"},
				provData:         "prov-data",
				accessDetails:    map[string]interface{}{"access": "details"},
				refAccessDetails: map[string]interface{}{"ref": "details"},
				appDetails:       map[string]interface{}{"app": "details"},
				instanceDetails:  map[string]interface{}{"instance": "details"},
			},
			expected: webhookAccessRequest{
				Operation:               webhookOperationProvision,
				ID:                      "ar-id",
				ManagedApplicationName:  "my-app",
				RequestData:             map[string]interface{}{"req": "data"},
				ProvisioningData:        "prov-data",
				AccessDetails:           map[string]interface{}{"access": "details"},
				ReferencedAccessDetails: map[string]interface{}{"ref": "details"},
				ApplicationDetails:      map[string]interface{}{"app": "details"},
				InstanceDetails:         map[string]interface{}{"instance": "details"},
			},
		},
		{
			name:    "transferring when refID set",
			request: provAccReq{id: "ar-id", refID: "ref-id"},
			expected: webhookAccessRequest{
				Operation:      webhookOperationProvision,
				ID:             "ar-id",
				ReferencedID:   "ref-id",
				IsTransferring: true,
			},
		},
		{
			name:    "with quota",
			request: provAccReq{id: "ar-id", quota: &mockQuota{limit: 100, interval: "daily"}},
			expected: webhookAccessRequest{
				Operation: webhookOperationProvision,
				ID:        "ar-id",
				Quota:     &webhookQuota{Limit: 100, Interval: "daily"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newWebhookAccessRequest(webhookOperationProvision, tc.request))
		})
	}
}

// mockIDPCredentialData is a minimal provisioning.IDPCredentialData implementation for testing.
type mockIDPCredentialData struct {
	clientID string
}

func (m *mockIDPCredentialData) GetClientID() string                { return m.clientID }
func (m *mockIDPCredentialData) GetClientSecret() string            { return "" }
func (m *mockIDPCredentialData) GetScopes() []string                { return nil }
func (m *mockIDPCredentialData) GetGrantTypes() []string            { return nil }
func (m *mockIDPCredentialData) GetTokenEndpointAuthMethod() string { return "" }
func (m *mockIDPCredentialData) GetResponseTypes() []string         { return nil }
func (m *mockIDPCredentialData) GetRedirectURIs() []string          { return nil }
func (m *mockIDPCredentialData) GetJwksURI() string                 { return "" }
func (m *mockIDPCredentialData) GetPublicKey() string               { return "" }
func (m *mockIDPCredentialData) GetCertificate() string             { return "" }
func (m *mockIDPCredentialData) GetCertificateMetadata() string     { return "" }
func (m *mockIDPCredentialData) GetTLSClientAuthSanDNS() string     { return "" }
func (m *mockIDPCredentialData) GetTLSClientAuthSanEmail() string   { return "" }
func (m *mockIDPCredentialData) GetTLSClientAuthSanIP() string      { return "" }
func (m *mockIDPCredentialData) GetTLSClientAuthSanURI() string     { return "" }

// mockOauthProvider is a minimal oauth.Provider implementation for testing.
type mockOauthProvider struct {
	tokenEndpoint string
}

func (m *mockOauthProvider) GetName() string                        { return "" }
func (m *mockOauthProvider) GetTitle() string                       { return "" }
func (m *mockOauthProvider) GetIssuer() string                      { return "" }
func (m *mockOauthProvider) GetTokenEndpoint() string               { return m.tokenEndpoint }
func (m *mockOauthProvider) GetMTLSTokenEndpoint() string           { return "" }
func (m *mockOauthProvider) GetAuthorizationEndpoint() string       { return "" }
func (m *mockOauthProvider) GetSupportedScopes() []string           { return nil }
func (m *mockOauthProvider) GetSupportedGrantTypes() []string       { return nil }
func (m *mockOauthProvider) GetSupportedTokenAuthMethods() []string { return nil }
func (m *mockOauthProvider) GetSupportedResponseMethod() []string   { return nil }
func (m *mockOauthProvider) RegisterClient(cm oauth.ClientMetadata) (oauth.ClientMetadata, error) {
	return nil, nil
}
func (m *mockOauthProvider) UnregisterClient(clientID, accessToken, registrationClientURI string, scopes []string, grantType string) error {
	return nil
}
func (m *mockOauthProvider) Validate() error                                 { return nil }
func (m *mockOauthProvider) GetConfig() corecfg.IDPConfig                    { return nil }
func (m *mockOauthProvider) GetMetadata() *oauth.AuthorizationServerMetadata { return nil }
func (m *mockOauthProvider) GetIDPResourceName() string                      { return "" }

// mockIDPProvisioner is a minimal idp.Provisioner implementation for testing.
type mockIDPProvisioner struct {
	isIDPCredential bool
	provider        oauth.Provider
	credentialData  provisioning.IDPCredentialData
}

func (m *mockIDPProvisioner) IsIDPCredential() bool          { return m.isIDPCredential }
func (m *mockIDPProvisioner) GetIDPProvider() oauth.Provider { return m.provider }
func (m *mockIDPProvisioner) GetIDPCredentialData() provisioning.IDPCredentialData {
	return m.credentialData
}
func (m *mockIDPProvisioner) RegisterClient() error                       { return nil }
func (m *mockIDPProvisioner) UnregisterClient() error                     { return nil }
func (m *mockIDPProvisioner) GetAgentDetails() (map[string]string, error) { return nil, nil }
func (m *mockIDPProvisioner) Validate() error                             { return nil }

func TestNewWebhookCredentialRequest(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		creds     *provCreds
		expected  webhookCredentialRequest
	}{
		{
			name:      "non-IDP credential",
			operation: webhookOperationProvision,
			creds: &provCreds{
				id:                "cred-id",
				name:              "cred-name",
				managedApp:        "my-app",
				credType:          "cred-type",
				credAction:        1,
				credData:          map[string]interface{}{"a": "b"},
				credDetails:       map[string]interface{}{"c": "d"},
				appDetails:        map[string]interface{}{"e": "f"},
				credSchema:        map[string]interface{}{"g": "h"},
				credProvSchema:    map[string]interface{}{"i": "j"},
				credSchemaDetails: map[string]interface{}{"k": "l"},
				provisionMode:     "mode",
				days:              30,
				idpProvisioner:    &mockIDPProvisioner{isIDPCredential: false},
			},
			expected: webhookCredentialRequest{
				Operation:                 webhookOperationProvision,
				ID:                        "cred-id",
				Name:                      "cred-name",
				ManagedApplicationName:    "my-app",
				CredentialType:            "cred-type",
				CredentialAction:          1,
				CredentialData:            map[string]interface{}{"a": "b"},
				CredentialDetails:         map[string]interface{}{"c": "d"},
				ApplicationDetails:        map[string]interface{}{"e": "f"},
				CredentialSchema:          map[string]interface{}{"g": "h"},
				CredentialProvisionSchema: map[string]interface{}{"i": "j"},
				CredentialSchemaDetails:   map[string]interface{}{"k": "l"},
				ProvisionMode:             "mode",
				ExpirationDays:            30,
			},
		},
		{
			name:      "IDP credential",
			operation: webhookOperationDeprovision,
			creds: &provCreds{
				id: "cred-id",
				idpProvisioner: &mockIDPProvisioner{
					isIDPCredential: true,
					provider:        &mockOauthProvider{tokenEndpoint: "https://idp/token"},
					credentialData:  &mockIDPCredentialData{clientID: "client-id"},
				},
			},
			expected: webhookCredentialRequest{
				Operation:        webhookOperationDeprovision,
				ID:               "cred-id",
				IDPClientID:      "client-id",
				IDPTokenEndpoint: "https://idp/token",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, newWebhookCredentialRequest(tc.operation, tc.creds))
		})
	}
}
