package oauth

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// mockIDPClient satisfies IDPClient for lifecycle tests.
type mockIDPClient struct {
	getInstances   func(map[string]string, string) ([]*apiv1.ResourceInstance, error)
	createOrUpdate func(apiv1.Interface) (*apiv1.ResourceInstance, error)
	createSubRes   func(apiv1.ResourceMeta, map[string]interface{}) error
	getResource    func(string) (*apiv1.ResourceInstance, error)
}

func (m *mockIDPClient) GetAPIV1ResourceInstances(q map[string]string, url string) ([]*apiv1.ResourceInstance, error) {
	return m.getInstances(q, url)
}
func (m *mockIDPClient) CreateOrUpdateResource(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.createOrUpdate(ri)
}
func (m *mockIDPClient) CreateSubResource(rm apiv1.ResourceMeta, subs map[string]interface{}) error {
	return m.createSubRes(rm, subs)
}
func (m *mockIDPClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	if m.getResource == nil {
		return nil, nil
	}
	return m.getResource(url)
}

// mockIdpCache satisfies idpCache for lifecycle tests.
type mockIdpCache struct {
	byTokenURL map[string]*apiv1.ResourceInstance
}

func newMockIdpCache() *mockIdpCache {
	return &mockIdpCache{
		byTokenURL: map[string]*apiv1.ResourceInstance{},
	}
}

func (c *mockIdpCache) GetIdentityProviderMetadataByTokenUrl(tokenURL string) *apiv1.ResourceInstance {
	return c.byTokenURL[tokenURL]
}
func (c *mockIdpCache) AddIdentityProviderMetadata(ri *apiv1.ResourceInstance) {
	c.byTokenURL[ri.Name] = ri
}

func makeTestProvider(t *testing.T) Provider {
	t.Helper()
	idpServer := NewMockIDPServer()
	t.Cleanup(idpServer.Close)
	idpCfg := createIDPConfig("test-idp", idpServer.GetMetadataURL())
	p, err := NewProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second)
	assert.NoError(t, err)
	return p
}

func makeTestMetadata(t *testing.T) (*AuthorizationServerMetadata, config.IDPConfig) {
	t.Helper()
	p := makeTestProvider(t)
	return p.GetMetadata(), p.GetConfig()
}

func idpRI(name string) *apiv1.ResourceInstance {
	ri, _ := management.NewIdentityProvider(name).AsInstance()
	return ri
}

func idpMetadataRI(idpScopeName string) *apiv1.ResourceInstance {
	meta := management.NewIdentityProviderMetadata("test-meta", idpScopeName)
	ri, _ := meta.AsInstance()
	return ri
}

func noOpCreateSubRes(_ apiv1.ResourceMeta, _ map[string]interface{}) error { return nil }

func TestCreateEngageResourcesQueryError(t *testing.T) {
	tests := map[string]struct {
		wantErr     bool
		wantCreated int
	}{
		"query error returns error and creates nothing": {
			wantErr:     true,
			wantCreated: 0,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return nil, errors.New("query error")
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			_, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantCreated, created)
		})
	}
}

func TestCreateEngageResourcesExistingFound(t *testing.T) {
	tests := map[string]struct {
		existingName string
		wantCreated  int
	}{
		"existing resource reused without creating": {
			existingName: "existing-idp",
			wantCreated:  0,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{idpMetadataRI(tc.existingName)}, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			resultName, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
			assert.NoError(t, err)
			assert.Equal(t, tc.existingName, resultName)
			assert.Equal(t, tc.wantCreated, created)
		})
	}
}

func TestCreateEngageResourcesIDPCreateError(t *testing.T) {
	tests := map[string]struct {
		wantErr     bool
		wantCreated int
	}{
		"IdP create error returns error and creates no metadata": {
			wantErr:     true,
			wantCreated: 1,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				getResource: func(_ string) (*apiv1.ResourceInstance, error) {
					return nil, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					return nil, errors.New("create error")
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			_, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantCreated, created)
		})
	}
}

func TestCreateEngageResourcesPolicyError(t *testing.T) {
	tests := map[string]struct {
		wantErr     bool
		wantCreated int
	}{
		"policy write error returns error and stores no name": {
			wantErr:     true,
			wantCreated: 1,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				getResource: func(_ string) (*apiv1.ResourceInstance, error) {
					return nil, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					return errors.New("policy write error")
				},
			}

			policies := management.EnvironmentPoliciesCredentials{
				Expiry: management.EnvironmentPoliciesCredentialsExpiry{Period: 90},
			}
			metadata, idpCfg := makeTestMetadata(t)
			_, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", policies)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantCreated, created)
		})
	}
}

func TestCreateEngageResourcesMetadataWriteError(t *testing.T) {
	tests := map[string]struct {
		wantErr     bool
		wantCreated int
	}{
		"metadata write error returns error and stores no name": {
			wantErr:     true,
			wantCreated: 2,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				getResource: func(_ string) (*apiv1.ResourceInstance, error) {
					return nil, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					if created == 1 {
						inst, _ := ri.AsInstance()
						return inst, nil
					}
					return nil, errors.New("metadata write error")
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			_, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantCreated, created)
		})
	}
}

func TestCreateEngageResourcesSuccess(t *testing.T) {
	tests := map[string]struct {
		withPolicies    bool
		wantCreateCount int
		wantSubRes      bool
	}{
		"creates IdP and metadata, no policies": {
			withPolicies:    false,
			wantCreateCount: 2,
			wantSubRes:      false,
		},
		"creates IdP, policies, and metadata": {
			withPolicies:    true,
			wantCreateCount: 2,
			wantSubRes:      true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			created := 0
			subResCalled := false
			client := &mockIDPClient{
				getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				getResource: func(_ string) (*apiv1.ResourceInstance, error) {
					return nil, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResCalled = true
					return nil
				},
			}

			var policies management.EnvironmentPoliciesCredentials
			if tc.withPolicies {
				policies = management.EnvironmentPoliciesCredentials{
					Expiry: management.EnvironmentPoliciesCredentialsExpiry{Period: 90},
				}
			}

			metadata, idpCfg := makeTestMetadata(t)
			resultName, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", policies)
			assert.NoError(t, err)
			assert.NotEmpty(t, resultName)
			assert.Equal(t, tc.wantCreateCount, created)
			assert.Equal(t, tc.wantSubRes, subResCalled)
		})
	}
}

func TestCreateEngageResourcesGetResourceError(t *testing.T) {
	created := 0
	client := &mockIDPClient{
		getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
			return []*apiv1.ResourceInstance{}, nil
		},
		getResource: func(_ string) (*apiv1.ResourceInstance, error) {
			return nil, errors.New("get resource error")
		},
		createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
			created++
			inst, _ := ri.AsInstance()
			return inst, nil
		},
		createSubRes: noOpCreateSubRes,
	}

	metadata, idpCfg := makeTestMetadata(t)
	_, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
	assert.Nil(t, err)
	assert.Equal(t, 2, created)
}

func TestCreateEngageResourcesIDPFoundViaGetResource(t *testing.T) {
	// When GetResource finds an existing IDP, creation is skipped but metadata is still created.
	created := 0
	existingIDP := idpRI("existing-idp")
	client := &mockIDPClient{
		getInstances: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
			return []*apiv1.ResourceInstance{}, nil
		},
		getResource: func(_ string) (*apiv1.ResourceInstance, error) {
			return existingIDP, nil
		},
		createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
			created++
			inst, _ := ri.AsInstance()
			return inst, nil
		},
		createSubRes: noOpCreateSubRes,
	}

	metadata, idpCfg := makeTestMetadata(t)
	resultName, err := NewIDPEngageLifecycle(client, newMockIdpCache()).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPType(), idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
	assert.NoError(t, err)
	assert.Equal(t, existingIDP.GetName(), resultName)
	assert.Equal(t, 1, created, "only metadata should be created when IDP already exists")
}

func newTestLogger() log.FieldLogger {
	return log.NewFieldLogger().WithComponent("idplifecycle-test").WithPackage("oauth")
}
