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

// mockBuilder satisfies IDPResourceBuilder for supplier tests.
type mockBuilder struct {
	idpResult      *management.IdentityProvider
	idpErr         error
	metadataResult *management.IdentityProviderMetadata
	metadataErr    error
}

func (b *mockBuilder) GetIdentityProvider(_ config.IDPConfig) (*management.IdentityProvider, error) {
	return b.idpResult, b.idpErr
}
func (b *mockBuilder) GetIdentityProviderMetadata(_ config.IDPConfig, _ *AuthorizationServerMetadata) (*management.IdentityProviderMetadata, error) {
	return b.metadataResult, b.metadataErr
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
			_, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
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
					return []*apiv1.ResourceInstance{idpRI(tc.existingName)}, nil
				},
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			resultName, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
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
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					return nil, errors.New("create error")
				},
				createSubRes: noOpCreateSubRes,
			}

			metadata, idpCfg := makeTestMetadata(t)
			_, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
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
			_, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", policies)
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
			_, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})
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
			resultName, err := NewIDPEngageLifecycle(client).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", policies)
			assert.NoError(t, err)
			assert.NotEmpty(t, resultName)
			assert.Equal(t, tc.wantCreateCount, created)
			assert.Equal(t, tc.wantSubRes, subResCalled)
		})
	}
}

const builderIDPName = "builder-idp"

func TestCreateEngageResourcesWithBuilder(t *testing.T) {
	tests := map[string]struct {
		idpErr          error
		metadataErr     error
		wantCreateCount int
		wantNameStored  bool
	}{
		"builder provides both IdP and metadata": {
			wantCreateCount: 2,
			wantNameStored:  true,
		},
		"builder IdP error returns error": {
			idpErr:          errors.New("builder idp error"),
			wantCreateCount: 0,
			wantNameStored:  false,
		},
		// supplier metadata build failure is non-fatal — IdP name stored, no metadata written
		"builder metadata error skips metadata but stores name": {
			metadataErr:     errors.New("builder metadata error"),
			wantCreateCount: 1,
			wantNameStored:  true,
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
				createOrUpdate: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					created++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				createSubRes: noOpCreateSubRes,
			}

			builder := &mockBuilder{
				idpResult:      management.NewIdentityProvider(builderIDPName),
				idpErr:         tc.idpErr,
				metadataResult: management.NewIdentityProviderMetadata(builderIDPName, builderIDPName),
				metadataErr:    tc.metadataErr,
			}

			metadata, idpCfg := makeTestMetadata(t)
			resultName, err := NewIDPEngageLifecycle(client, WithResourceBuilder(builder)).CreateEngageResourcesFromMetadata(newTestLogger(), idpCfg, idpCfg.GetIDPName(), metadata, "/env", management.EnvironmentPoliciesCredentials{})

			assert.Equal(t, tc.wantCreateCount, created)
			assert.Equal(t, tc.wantNameStored, resultName != "")
			assert.Equal(t, tc.wantNameStored, err == nil)
		})
	}
}

func newTestLogger() log.FieldLogger {
	return log.NewFieldLogger().WithComponent("idplifecycle-test").WithPackage("oauth")
}
