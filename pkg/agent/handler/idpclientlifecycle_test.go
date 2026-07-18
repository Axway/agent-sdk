package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleClientMock struct {
	createSubCalled bool
	createSubErr    error
	lastSubs        map[string]interface{}
}

func (m *lifecycleClientMock) GetResource(_ string) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m *lifecycleClientMock) UpdateResourceFinalizer(_ *apiv1.ResourceInstance, _, _ string, _ bool) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m *lifecycleClientMock) CreateSubResource(_ apiv1.ResourceMeta, subs map[string]interface{}) error {
	m.createSubCalled = true
	m.lastSubs = subs
	return m.createSubErr
}

func (m *lifecycleClientMock) UpdateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m *lifecycleClientMock) DeleteResourceInstance(_ apiv1.Interface) error {
	return nil
}

type idpProviderMock struct {
	unregisterCalls []string
	unregisterErr   error
}

func (m *idpProviderMock) GetName() string { return "mock" }

func (m *idpProviderMock) GetTitle() string { return "mock" }

func (m *idpProviderMock) GetIssuer() string { return "issuer" }

func (m *idpProviderMock) GetTokenEndpoint() string { return "https://idp.example.com/token" }

func (m *idpProviderMock) GetMTLSTokenEndpoint() string { return "" }

func (m *idpProviderMock) GetAuthorizationEndpoint() string { return "https://idp.example.com/auth" }

func (m *idpProviderMock) GetSupportedScopes() []string { return nil }

func (m *idpProviderMock) GetSupportedGrantTypes() []string { return nil }

func (m *idpProviderMock) GetSupportedTokenAuthMethods() []string { return nil }

func (m *idpProviderMock) GetSupportedResponseMethod() []string { return nil }

func (m *idpProviderMock) RegisterClient(_ oauth.ClientMetadata) (oauth.ClientMetadata, error) {
	return nil, nil
}

func (m *idpProviderMock) UnregisterClient(clientID, _, _ string, _ []string, _ string) error {
	m.unregisterCalls = append(m.unregisterCalls, clientID)
	return m.unregisterErr
}

func (m *idpProviderMock) Validate() error { return nil }

func (m *idpProviderMock) GetConfig() corecfg.IDPConfig { return nil }

func (m *idpProviderMock) GetMetadata() *oauth.AuthorizationServerMetadata { return nil }

func (m *idpProviderMock) GetIDPResourceName() string { return "" }

type idpRegistryMock struct {
	provider    oauth.Provider
	err         error
	called      bool
	gotTokenURL string
}

func newManagedAppForLifecycle(name string) *management.ManagedApplication {
	return management.NewManagedApplication(name, "env")
}

func (m *idpRegistryMock) RegisterProvider(_ context.Context, _ corecfg.IDPConfig, _ corecfg.TLSConfig, _ string, _ time.Duration) error {
	return nil
}

func (m *idpRegistryMock) RegisterProviderWithMetadata(_ context.Context, _ corecfg.IDPConfig, _ *oauth.AuthorizationServerMetadata, _ corecfg.TLSConfig, _ string, _ time.Duration) error {
	return nil
}

func (m *idpRegistryMock) UnregisterProvider(_ context.Context, _ oauth.Provider) error { return nil }

func (m *idpRegistryMock) GetProviderByName(_ context.Context, _ string, _ ...oauth.ConfigOption) (oauth.Provider, error) {
	return nil, nil
}

func (m *idpRegistryMock) GetProviderByIssuer(_ context.Context, _ string, _ ...oauth.ConfigOption) (oauth.Provider, error) {
	return nil, nil
}

func (m *idpRegistryMock) GetProviderByTokenEndpoint(_ context.Context, tokenEndpoint string, _ ...oauth.ConfigOption) (oauth.Provider, error) {
	m.called = true
	m.gotTokenURL = tokenEndpoint
	return m.provider, m.err
}

func (m *idpRegistryMock) GetProviderByAuthorizationEndpoint(_ context.Context, _ string, _ ...oauth.ConfigOption) (oauth.Provider, error) {
	return nil, nil
}

func (m *idpRegistryMock) GetProviderByMetadataURL(_ context.Context, _ string, _ ...oauth.ConfigOption) (oauth.Provider, error) {
	return nil, nil
}

func (m *idpRegistryMock) GetIDPResourceName(_ string) (string, bool) { return "", false }

func TestExtractClientIDs(t *testing.T) {
	tests := map[string]struct {
		raw     interface{}
		expect  []string
		nilResp bool
	}{
		"from []string": {
			raw:    []string{"a", "b"},
			expect: []string{"a", "b"},
		},
		"from []interface{}": {
			raw:    []interface{}{"a", 1, "b"},
			expect: []string{"a", "b"},
		},
		"from JSON string": {
			raw:    `["a","b"]`,
			expect: []string{"a", "b"},
		},
		"invalid JSON string": {
			raw:     `not-json`,
			nilResp: true,
		},
		"nil input": {
			raw:     nil,
			nilResp: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := extractClientIDs(tt.raw)
			if tt.nilResp {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestPersistIDPClientOnManagedApplication(t *testing.T) {
	tests := map[string]struct {
		app          *management.ManagedApplication
		clientID     string
		tokenURL     string
		createSubErr error
		wantErr      bool
		wantCall     bool
		wantIDs      []string
		wantTokenURL string
	}{
		"nil app": {
			clientID: "client-1",
			tokenURL: "https://idp.example.com/token",
		},
		"empty client ID": {
			app:      newManagedAppForLifecycle("app"),
			tokenURL: "https://idp.example.com/token",
		},
		"first client is stored": {
			app:          newManagedAppForLifecycle("app"),
			clientID:     "client-1",
			tokenURL:     "https://idp.example.com/token",
			wantCall:     true,
			wantIDs:      []string{"client-1"},
			wantTokenURL: "https://idp.example.com/token",
		},
		"duplicate client is ignored": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{oktaClientIDsAgentDetail: []interface{}{"client-1"}})
				return a
			}(),
			clientID: "client-1",
			tokenURL: "https://idp.example.com/token",
		},
		"second client is appended": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{oktaClientIDsAgentDetail: []interface{}{"client-1"}})
				return a
			}(),
			clientID:     "client-2",
			tokenURL:     "https://idp.example.com/token",
			wantCall:     true,
			wantIDs:      []string{"client-1", "client-2"},
			wantTokenURL: "https://idp.example.com/token",
		},
		"create subresource error is returned": {
			app:          newManagedAppForLifecycle("app"),
			clientID:     "client-1",
			tokenURL:     "https://idp.example.com/token",
			createSubErr: errors.New("write failed"),
			wantErr:      true,
			wantCall:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &lifecycleClientMock{createSubErr: tt.createSubErr}
			err := persistIDPClientOnManagedApplication(log.NewFieldLogger(), c, tt.app, tt.clientID, tt.tokenURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantCall, c.createSubCalled)

			if tt.wantCall && !tt.wantErr {
				details, ok := c.lastSubs[defs.XAgentDetails].(map[string]interface{})
				require.True(t, ok)
				ids, ok := details[oktaClientIDsAgentDetail].([]string)
				require.True(t, ok)
				assert.Equal(t, tt.wantIDs, ids)
				assert.Equal(t, tt.wantTokenURL, details[tokenURLAgentDetail])
			}
		})
	}
}

func TestCleanupManagedApplicationIDPClients(t *testing.T) {
	tests := map[string]struct {
		app                 *management.ManagedApplication
		registry            *idpRegistryMock
		wantErr             bool
		wantRegistryLookup  bool
		wantUnregisterCalls []string
	}{
		"nil registry": {
			app: newManagedAppForLifecycle("app"),
		},
		"nil app": {
			registry: &idpRegistryMock{},
		},
		"no client IDs": {
			app:      newManagedAppForLifecycle("app"),
			registry: &idpRegistryMock{},
		},
		"missing token URL skips cleanup": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{oktaClientIDsAgentDetail: []interface{}{"client-1"}})
				return a
			}(),
			registry: &idpRegistryMock{},
		},
		"provider lookup error is ignored": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{
					oktaClientIDsAgentDetail: []interface{}{"client-1"},
					tokenURLAgentDetail:      "https://idp.example.com/token",
				})
				return a
			}(),
			registry:           &idpRegistryMock{err: errors.New("not found")},
			wantRegistryLookup: true,
		},
		"provider is nil is ignored": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{
					oktaClientIDsAgentDetail: []interface{}{"client-1"},
					tokenURLAgentDetail:      "https://idp.example.com/token",
				})
				return a
			}(),
			registry:           &idpRegistryMock{},
			wantRegistryLookup: true,
		},
		"unregister error is returned": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{
					oktaClientIDsAgentDetail: []string{"client-1"},
					tokenURLAgentDetail:      "https://idp.example.com/token",
				})
				return a
			}(),
			registry:            &idpRegistryMock{provider: &idpProviderMock{unregisterErr: errors.New("unregister failed")}},
			wantRegistryLookup:  true,
			wantErr:             true,
			wantUnregisterCalls: []string{"client-1"},
		},
		"success unregisters all IDs": {
			app: func() *management.ManagedApplication {
				a := newManagedAppForLifecycle("app")
				util.SetAgentDetails(a, map[string]interface{}{
					oktaClientIDsAgentDetail: `["client-1","client-2"]`,
					tokenURLAgentDetail:      "https://idp.example.com/token",
				})
				return a
			}(),
			registry:            &idpRegistryMock{provider: &idpProviderMock{}},
			wantRegistryLookup:  true,
			wantUnregisterCalls: []string{"client-1", "client-2"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := cleanupManagedApplicationIDPClients(context.Background(), log.NewFieldLogger(), tt.registry, tt.app)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.registry != nil {
				assert.Equal(t, tt.wantRegistryLookup, tt.registry.called)
			}

			if tt.registry != nil && tt.registry.provider != nil {
				if p, ok := tt.registry.provider.(*idpProviderMock); ok {
					assert.Equal(t, tt.wantUnregisterCalls, p.unregisterCalls)
				}
			}
		})
	}
}
