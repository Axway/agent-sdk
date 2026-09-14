package handler

import (
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
	h := newMockSubResourceCarrier()
	assert.Equal(t, "", webhookDispatchedOperation(h))

	markWebhookDispatched(h, webhookOperationProvision)
	assert.Equal(t, webhookOperationProvision, webhookDispatchedOperation(h))

	markWebhookDispatched(h, webhookOperationDeprovision)
	assert.Equal(t, webhookOperationDeprovision, webhookDispatchedOperation(h))
}

func TestMarkWebhookDispatched(t *testing.T) {
	h := newMockSubResourceCarrier()
	markWebhookDispatched(h, webhookOperationProvision)

	details, ok := h.GetSubResource(defs.XAgentDetails).(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, webhookOperationProvision, details[webhookDispatchDetailKey])
}

func TestWebhookDetailsValue(t *testing.T) {
	h := newMockSubResourceCarrier()
	assert.Equal(t, "", webhookDetailsValue(h, webhookStatusKey))

	h.SetSubResource(defs.XWebhookDetails, map[string]interface{}{
		webhookStatusKey:  webhookStatusSuccess,
		webhookMessageKey: "all good",
	})
	assert.Equal(t, webhookStatusSuccess, webhookDetailsValue(h, webhookStatusKey))
	assert.Equal(t, "all good", webhookDetailsValue(h, webhookMessageKey))
	assert.Equal(t, "", webhookDetailsValue(h, "missing-key"))

	// non-string values are ignored
	h.SetSubResource(defs.XWebhookDetails, map[string]interface{}{webhookStatusKey: 1})
	assert.Equal(t, "", webhookDetailsValue(h, webhookStatusKey))
}

func TestMirrorWebhookDetails(t *testing.T) {
	t.Run("no webhook details to mirror", func(t *testing.T) {
		h := newMockSubResourceCarrier()
		assert.False(t, mirrorWebhookDetails(h))
	})

	t.Run("empty webhook details map", func(t *testing.T) {
		h := newMockSubResourceCarrier()
		h.SetSubResource(defs.XWebhookDetails, map[string]interface{}{})
		assert.False(t, mirrorWebhookDetails(h))
	})

	t.Run("mirrors into new agent details", func(t *testing.T) {
		h := newMockSubResourceCarrier()
		h.SetSubResource(defs.XWebhookDetails, map[string]interface{}{
			webhookStatusKey: webhookStatusSuccess,
		})

		assert.True(t, mirrorWebhookDetails(h))

		agentDetails, ok := h.GetSubResource(defs.XAgentDetails).(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, webhookStatusSuccess, agentDetails[webhookStatusKey])
	})

	t.Run("merges into existing agent details without dropping other keys", func(t *testing.T) {
		h := newMockSubResourceCarrier()
		h.SetSubResource(defs.XAgentDetails, map[string]interface{}{"existing": "value"})
		h.SetSubResource(defs.XWebhookDetails, map[string]interface{}{
			webhookStatusKey: webhookStatusFailed,
		})

		assert.True(t, mirrorWebhookDetails(h))

		agentDetails, ok := h.GetSubResource(defs.XAgentDetails).(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "value", agentDetails["existing"])
		assert.Equal(t, webhookStatusFailed, agentDetails[webhookStatusKey])
	})
}

func TestNewWebhookApplicationRequest(t *testing.T) {
	app := provManagedApp{
		id:             "app-id",
		managedAppName: "my-app",
		teamName:       "my-team",
		consumerOrgID:  "org-id",
		data:           map[string]interface{}{"foo": "bar"},
	}

	req := newWebhookApplicationRequest(webhookOperationProvision, app)

	assert.Equal(t, webhookOperationProvision, req.Operation)
	assert.Equal(t, "app-id", req.ID)
	assert.Equal(t, "my-app", req.ManagedApplicationName)
	assert.Equal(t, "my-team", req.TeamName)
	assert.Equal(t, "org-id", req.ConsumerOrgID)
	assert.Equal(t, app.data, req.AgentDetails)
}

func TestNewWebhookApplicationProfileRequest(t *testing.T) {
	profile := provManagedAppProfile{
		id:                "profile-id",
		managedAppName:    "my-app",
		profileDefinition: "my-profile-def",
		teamName:          "my-team",
		consumerOrgID:     "org-id",
		attributes:        map[string]interface{}{"attr": "val"},
		data:              map[string]interface{}{"foo": "bar"},
	}

	req := newWebhookApplicationProfileRequest(webhookOperationDeprovision, profile)

	assert.Equal(t, webhookOperationDeprovision, req.Operation)
	assert.Equal(t, "profile-id", req.ID)
	assert.Equal(t, "my-app", req.ManagedApplicationName)
	assert.Equal(t, "my-profile-def", req.ApplicationProfileDefinition)
	assert.Equal(t, "my-team", req.TeamName)
	assert.Equal(t, "org-id", req.ConsumerOrgID)
	assert.Equal(t, profile.attributes, req.Attributes)
	assert.Equal(t, profile.data, req.ApplicationDetails)
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
	t.Run("without quota, not transferring", func(t *testing.T) {
		r := provAccReq{
			id:               "ar-id",
			managedApp:       "my-app",
			requestData:      map[string]interface{}{"req": "data"},
			provData:         "prov-data",
			accessDetails:    map[string]interface{}{"access": "details"},
			refAccessDetails: map[string]interface{}{"ref": "details"},
			appDetails:       map[string]interface{}{"app": "details"},
			instanceDetails:  map[string]interface{}{"instance": "details"},
		}

		req := newWebhookAccessRequest(webhookOperationProvision, r)

		assert.Equal(t, webhookOperationProvision, req.Operation)
		assert.Equal(t, "ar-id", req.ID)
		assert.Equal(t, "", req.ReferencedID)
		assert.Equal(t, "my-app", req.ManagedApplicationName)
		assert.False(t, req.IsTransferring)
		assert.Equal(t, r.requestData, req.RequestData)
		assert.Equal(t, r.provData, req.ProvisioningData)
		assert.Equal(t, r.accessDetails, req.AccessDetails)
		assert.Equal(t, r.refAccessDetails, req.ReferencedAccessDetails)
		assert.Equal(t, r.appDetails, req.ApplicationDetails)
		assert.Equal(t, r.instanceDetails, req.InstanceDetails)
		assert.Nil(t, req.Quota)
	})

	t.Run("transferring when refID set", func(t *testing.T) {
		r := provAccReq{id: "ar-id", refID: "ref-id"}
		req := newWebhookAccessRequest(webhookOperationProvision, r)
		assert.Equal(t, "ref-id", req.ReferencedID)
		assert.True(t, req.IsTransferring)
	})

	t.Run("with quota", func(t *testing.T) {
		r := provAccReq{
			id:    "ar-id",
			quota: &mockQuota{limit: 100, interval: "daily"},
		}
		req := newWebhookAccessRequest(webhookOperationProvision, r)
		if assert.NotNil(t, req.Quota) {
			assert.Equal(t, int64(100), req.Quota.Limit)
			assert.Equal(t, "daily", req.Quota.Interval)
		}
	})
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
	t.Run("non-IDP credential", func(t *testing.T) {
		c := &provCreds{
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
		}

		req := newWebhookCredentialRequest(webhookOperationProvision, c)

		assert.Equal(t, webhookOperationProvision, req.Operation)
		assert.Equal(t, "cred-id", req.ID)
		assert.Equal(t, "cred-name", req.Name)
		assert.Equal(t, "my-app", req.ManagedApplicationName)
		assert.Equal(t, "cred-type", req.CredentialType)
		assert.Equal(t, 1, req.CredentialAction)
		assert.Equal(t, c.credData, req.CredentialData)
		assert.Equal(t, c.credDetails, req.CredentialDetails)
		assert.Equal(t, c.appDetails, req.ApplicationDetails)
		assert.Equal(t, c.credSchema, req.CredentialSchema)
		assert.Equal(t, c.credProvSchema, req.CredentialProvisionSchema)
		assert.Equal(t, c.credSchemaDetails, req.CredentialSchemaDetails)
		assert.Equal(t, "mode", req.ProvisionMode)
		assert.Equal(t, 30, req.ExpirationDays)
		assert.Equal(t, "", req.IDPClientID)
		assert.Equal(t, "", req.IDPTokenEndpoint)
	})

	t.Run("IDP credential", func(t *testing.T) {
		c := &provCreds{
			id: "cred-id",
			idpProvisioner: &mockIDPProvisioner{
				isIDPCredential: true,
				provider:        &mockOauthProvider{tokenEndpoint: "https://idp/token"},
				credentialData:  &mockIDPCredentialData{clientID: "client-id"},
			},
		}

		req := newWebhookCredentialRequest(webhookOperationDeprovision, c)

		assert.Equal(t, "client-id", req.IDPClientID)
		assert.Equal(t, "https://idp/token", req.IDPTokenEndpoint)
	})
}
