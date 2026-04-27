package agent

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
)

const (
	testEnvName     = "test-env"
	testIDPName     = "test-idp"
	existingIDPName = "existing-idp"
	supplierIDPName = "supplier-idp"
)

// setupIDPLifecycleAgent initialises the global agent with a minimal discovery-agent config
// pointing at a real httptest server (for auth + env endpoint responses) and wires the
// supplied apicClient mock.
func setupIDPLifecycleAgent(t *testing.T, apicClient *mock.Client, flagEnabled bool) (oauth.MockIDPServer, func()) {
	t.Helper()
	idpServer := oauth.NewMockIDPServer()

	centralServer := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			resp.Write([]byte(`{"access_token":"tok","expires_in":9999}`))
			return
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/"+testEnvName) {
			env := management.NewEnvironment(testEnvName)
			buf, _ := json.Marshal(env)
			resp.Write(buf)
			return
		}
	}))

	cfg := createCentralCfg(centralServer.URL, testEnvName)
	resetResources()
	features := &config.AgentFeaturesConfiguration{IDPResourceMgmt: flagEnabled}
	InitializeWithAgentFeatures(cfg, features, nil)
	agent.apicClient = apicClient
	agent.authProviderRegistry = nil // force fresh registry

	return idpServer, func() {
		idpServer.Close()
		centralServer.Close()
	}
}

func makeIDPConfig(metadataURL string) *config.IDPConfiguration {
	return &config.IDPConfiguration{
		Name:        testIDPName,
		Type:        "okta",
		MetadataURL: metadataURL,
		AuthConfig: &config.IDPAuthConfiguration{
			Type:         "client",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
		GrantType:        oauth.GrantTypeClientCredentials,
		ClientScopes:     "read",
		AuthMethod:       config.ClientSecretBasic,
		AuthResponseType: oauth.AuthResponseToken,
	}
}

// idpInstanceRI builds a *ResourceInstance that looks like a fetched IdentityProvider.
func idpInstanceRI(name string) *apiv1.ResourceInstance {
	idp := management.NewIdentityProvider(name)
	ri, _ := idp.AsInstance()
	return ri
}

func TestManageIDPResourceFlagDisabled(t *testing.T) {
	tests := map[string]struct {
		flagEnabled        bool
		expectedQueryCount int
	}{
		"no IdP query when flag disabled": {flagEnabled: false, expectedQueryCount: 0},
		"IdP query made when flag enabled": {flagEnabled: true, expectedQueryCount: 1},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			queryCallCount := 0
			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					queryCallCount++
					return nil, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				UpdateResourceInstanceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, tc.flagEnabled)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			err := registerCredentialProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedQueryCount, queryCallCount)
		})
	}
}

func TestManageIDPResourceExistingFound(t *testing.T) {
	tests := map[string]struct {
		existingName    string
		wantCreateCalled bool
		wantSubResource  bool
	}{
		"reuses existing resource name without creating": {
			existingName:    existingIDPName,
			wantCreateCalled: false,
			wantSubResource:  false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			existingRI := idpInstanceRI(tc.existingName)
			createCalled := false
			subResourceCalled := false

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{existingRI}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCalled = true
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResourceCalled = true
					return nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.Equal(t, tc.existingName, resultName, "should reuse the existing resource name")
			assert.Equal(t, tc.wantCreateCalled, createCalled, "should not create a new IdentityProvider resource")
			assert.Equal(t, tc.wantSubResource, subResourceCalled, "should not write policies when reusing existing resource")

			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.True(t, ok)
			assert.Equal(t, tc.existingName, storedName)
		})
	}
}

func TestManageIDPResourceQueryError(t *testing.T) {
	tests := map[string]struct {
		wantNameStored bool
	}{
		"query error falls through to creation and name is stored": {
			wantNameStored: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return nil, errors.New("query error")
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)

			assert.NotEmpty(t, resultName, "should attempt creation and return a name even when query errors")
			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.Equal(t, tc.wantNameStored, ok)
			if tc.wantNameStored {
				assert.Equal(t, resultName, storedName)
			}
		})
	}
}

func TestManageIDPResourceCreatedSuccessfully(t *testing.T) {
	tests := map[string]struct {
		wantCreateCount     int
		wantSubResource     bool
		envExpiryPeriod     int32
		envVisibilityPeriod int32
	}{
		"creates IdP, policies sub-resource, and metadata": {
			wantCreateCount:     2,
			wantSubResource:     true,
			envExpiryPeriod:     90,
			envVisibilityPeriod: 30,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0
			subResourceCalled := false
			envPolicies := management.EnvironmentPoliciesCredentials{
				Expiry:     management.EnvironmentPoliciesCredentialsExpiry{Period: tc.envExpiryPeriod},
				Visibility: management.EnvironmentPoliciesCredentialsVisibility{Period: tc.envVisibilityPeriod},
			}

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCount++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResourceCalled = true
					return nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					env := management.NewEnvironment(testEnvName)
					env.Policies = management.EnvironmentPolicies{Credentials: envPolicies}
					return env, nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.NotEmpty(t, resultName, "should return the created resource name")
			assert.Equal(t, tc.wantCreateCount, createCount, "should call CreateOrUpdateResource for IdP and IdP Metadata")
			assert.Equal(t, tc.wantSubResource, subResourceCalled, "should call CreateSubResource for policies")

			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.True(t, ok)
			assert.Equal(t, resultName, storedName)
		})
	}
}

func TestManageIDPResourceCreateIDPError(t *testing.T) {
	tests := map[string]struct {
		wantCreateCount int
		wantSubResource bool
	}{
		"no policies written after IdP create error": {
			wantCreateCount: 1,
			wantSubResource: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0
			subResourceCalled := false

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCount++
					return nil, errors.New("server error")
				},
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResourceCalled = true
					return nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.Empty(t, resultName, "should return empty string on IdP create error")
			assert.Equal(t, tc.wantCreateCount, createCount, "should only attempt to create the IdentityProvider")
			assert.Equal(t, tc.wantSubResource, subResourceCalled, "should not write policies after IdP create error")
		})
	}
}

func TestManageIDPResourceMetadataFetchError(t *testing.T) {
	tests := map[string]struct {
		wantCreateCount int
		metadataStatus  int
	}{
		"IdP created and name stored even when metadata fetch fails": {
			wantCreateCount: 1,
			metadataStatus:  http.StatusInternalServerError,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCount++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()
			idpServer.SetMetadataResponseCode(tc.metadataStatus)

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.NotEmpty(t, resultName, "should return the created IdP name even when metadata fetch fails")
			assert.Equal(t, tc.wantCreateCount, createCount, "should only create the IdentityProvider — not IdP Metadata")

			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.True(t, ok)
			assert.Equal(t, resultName, storedName)
		})
	}
}

func TestManageIDPResourceMetadataWriteError(t *testing.T) {
	tests := map[string]struct {
		wantCreateCount int
	}{
		"IdP name still stored when metadata write fails": {
			wantCreateCount: 2,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCount++
					if createCount == 1 {
						inst, _ := ri.AsInstance()
						return inst, nil
					}
					return nil, errors.New("metadata write error")
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.NotEmpty(t, resultName, "should return the created IdP name even when metadata write fails")
			assert.Equal(t, tc.wantCreateCount, createCount, "should attempt both IdP and IdP Metadata create")

			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.True(t, ok)
			assert.Equal(t, resultName, storedName)
		})
	}
}

func TestManageIDPResourceWithSupplier(t *testing.T) {
	tests := map[string]struct {
		supplierIDPName      string
		supplierMetaName     string
		wantIDPKind          string
		wantMetadataKind     string
	}{
		"supplier results used for both IdP and Metadata resources": {
			supplierIDPName:  supplierIDPName,
			supplierMetaName: supplierIDPName + "-meta",
			wantIDPKind:      management.IdentityProviderGVK().Kind,
			wantMetadataKind: management.IdentityProviderMetadataGVK().Kind,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			customIDP := management.NewIdentityProvider(tc.supplierIDPName)
			customMetadata := management.NewIdentityProviderMetadata(tc.supplierMetaName, tc.supplierIDPName)
			var createdResources []string

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					inst, _ := ri.AsInstance()
					createdResources = append(createdResources, inst.Kind)
					return inst, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			SetIDPResourceSupplier(&mockIDPResourceSupplier{
				idpResult:      customIDP,
				metadataResult: customMetadata,
			})
			defer SetIDPResourceSupplier(nil)

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)
			assert.Equal(t, tc.supplierIDPName, resultName)
			assert.Contains(t, createdResources, tc.wantIDPKind)
			assert.Contains(t, createdResources, tc.wantMetadataKind)
		})
	}
}

func TestGetEnvCredentialPoliciesError(t *testing.T) {
	tests := map[string]struct {
		envErr          error
		wantExpiryZero  bool
		wantVisibZero   bool
	}{
		"empty policies returned when GetEnvironment errors": {
			envErr:         errors.New("env fetch failed"),
			wantExpiryZero: true,
			wantVisibZero:  true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			apicClient := &mock.Client{
				GetEnvironmentMock: func() (*management.Environment, error) {
					return nil, tc.envErr
				},
			}

			_, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			policies := getEnvCredentialPolicies(logger)
			if tc.wantExpiryZero {
				assert.Zero(t, policies.Expiry.Period)
			}
			if tc.wantVisibZero {
				assert.Zero(t, policies.Visibility.Period)
			}
		})
	}
}

func TestApplyIDPPoliciesSkippedWhenEmpty(t *testing.T) {
	tests := map[string]struct {
		policies            management.EnvironmentPoliciesCredentials
		wantSubResourceCall bool
	}{
		"no sub-resource call when both periods are zero": {
			policies:            management.EnvironmentPoliciesCredentials{},
			wantSubResourceCall: false,
		},
		"sub-resource called when only expiry period is set": {
			policies: management.EnvironmentPoliciesCredentials{
				Expiry: management.EnvironmentPoliciesCredentialsExpiry{Period: 90},
			},
			wantSubResourceCall: true,
		},
		"sub-resource called when only visibility period is set": {
			policies: management.EnvironmentPoliciesCredentials{
				Visibility: management.EnvironmentPoliciesCredentialsVisibility{Period: 30},
			},
			wantSubResourceCall: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			subResourceCalled := false
			apicClient := &mock.Client{
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResourceCalled = true
					return nil
				},
			}

			_, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpResource := management.NewIdentityProvider(testIDPName)
			applyIDPPolicies(logger, idpResource, tc.policies)
			assert.Equal(t, tc.wantSubResourceCall, subResourceCalled, "should not call CreateSubResource when policies are empty")
		})
	}
}

func TestApplyIDPPoliciesWritten(t *testing.T) {
	tests := map[string]struct {
		expiryPeriod     int32
		visibilityPeriod int32
		wantCallCount    int
	}{
		"sub-resource written once when policies are non-zero": {
			expiryPeriod:     90,
			visibilityPeriod: 30,
			wantCallCount:    1,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			subResourceCount := 0
			apicClient := &mock.Client{
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					subResourceCount++
					return nil
				},
			}

			_, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpResource := management.NewIdentityProvider(testIDPName)
			policies := management.EnvironmentPoliciesCredentials{
				Expiry:     management.EnvironmentPoliciesCredentialsExpiry{Period: tc.expiryPeriod},
				Visibility: management.EnvironmentPoliciesCredentialsVisibility{Period: tc.visibilityPeriod},
			}
			applyIDPPolicies(logger, idpResource, policies)
			assert.Equal(t, tc.wantCallCount, subResourceCount)
		})
	}
}

func TestManageIDPResourceSupplierIDPError(t *testing.T) {
	tests := map[string]struct {
		supplierErr     error
		wantResultEmpty bool
		wantNameStored  bool
	}{
		"supplier GetIdentityProvider error returns empty name and stores nothing": {
			supplierErr:     errors.New("supplier idp error"),
			wantResultEmpty: true,
			wantNameStored:  false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			SetIDPResourceSupplier(&mockIDPResourceSupplier{idpErr: tc.supplierErr})
			defer SetIDPResourceSupplier(nil)

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)

			if tc.wantResultEmpty {
				assert.Empty(t, resultName)
			}
			_, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.Equal(t, tc.wantNameStored, ok)
		})
	}
}

func TestManageIDPResourceSupplierMetadataError(t *testing.T) {
	tests := map[string]struct {
		metadataErr      error
		wantCreateCount  int
		wantNameStored   bool
	}{
		"supplier GetIdentityProviderMetadata error — IdP name still stored, no metadata resource created": {
			metadataErr:     errors.New("supplier metadata error"),
			wantCreateCount: 1,
			wantNameStored:  true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0
			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface) (*apiv1.ResourceInstance, error) {
					createCount++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			SetIDPResourceSupplier(&mockIDPResourceSupplier{
				idpResult:   management.NewIdentityProvider(supplierIDPName),
				metadataErr: tc.metadataErr,
			})
			defer SetIDPResourceSupplier(nil)

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)

			assert.NotEmpty(t, resultName)
			assert.Equal(t, tc.wantCreateCount, createCount, "only the IdP resource should be created — not metadata")
			storedName, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.Equal(t, tc.wantNameStored, ok)
			if tc.wantNameStored {
				assert.Equal(t, resultName, storedName)
			}
		})
	}
}

func TestWithCRDIdentityProvider(t *testing.T) {
	tests := map[string]struct {
		idpName  string
		expected string
	}{
		"sets non-empty name": {idpName: "my-idp", expected: "my-idp"},
		"sets empty name":     {idpName: "", expected: ""},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			opts := &crdBuilderOptions{}
			WithCRDIdentityProvider(tc.idpName)(opts)
			assert.Equal(t, tc.expected, opts.identityProvider)
		})
	}
}
