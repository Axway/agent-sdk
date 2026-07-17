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

	"github.com/Axway/agent-sdk/pkg/apic"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
)

const (
	testEnvName     = "test-env"
	testIDPName     = "test-idp"
	existingIDPName = "existing-idp"
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
		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/"+testEnvName) {
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

	// Pre-register the provider so GetMetadata() is available when tests call manageIDPResource directly.
	idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
	_ = GetAuthProviderRegistry().RegisterProvider(idpCfg, config.NewTLSConfig(), "", 30*time.Second)

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

// idpMetadataInstanceRI builds a *ResourceInstance that looks like a fetched IdentityProviderMetadata
// with idpScopeName as the scope name — used to test the ManageIDPResource Engage query path.
func idpMetadataInstanceRI(idpScopeName string) *apiv1.ResourceInstance {
	meta := management.NewIdentityProviderMetadata("test-meta", idpScopeName)
	ri, _ := meta.AsInstance()
	return ri
}

func TestManageIDPResourceFlagDisabled(t *testing.T) {
	tests := map[string]struct {
		flagEnabled        bool
		expectedQueryCount int
	}{
		"no IdP query when flag disabled":  {flagEnabled: false, expectedQueryCount: 0},
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
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
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
		existingName     string
		wantCreateCalled bool
		wantSubResource  bool
	}{
		"reuses existing resource name without creating": {
			existingName:     existingIDPName,
			wantCreateCalled: false,
			wantSubResource:  false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			existingRI := idpMetadataInstanceRI(tc.existingName)
			createCalled := false
			subResourceCalled := false

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return []*apiv1.ResourceInstance{existingRI}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
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
		"query error aborts creation and name is not stored": {
			wantNameStored: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					return nil, errors.New("query error")
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			idpServer, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			idpCfg := makeIDPConfig(idpServer.GetMetadataURL())
			resultName := manageIDPResource(logger, idpCfg)

			assert.Empty(t, resultName, "query error should abort creation and return empty name")
			_, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.Equal(t, tc.wantNameStored, ok)
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
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
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
				CreateOrUpdateResourceMock: func(_ apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
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

func TestManageIDPResourceProviderNotFound(t *testing.T) {
	tests := map[string]struct {
		wantCreateCount int
		wantResult      string
	}{
		"empty result when provider not in registry": {
			wantCreateCount: 0,
			wantResult:      "",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			createCount := 0

			// Bare agent with an empty registry — no RegisterProvider call.
			resetResources()
			agent.apicClient = &mock.Client{
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
					createCount++
					inst, _ := ri.AsInstance()
					return inst, nil
				},
			}
			agent.authProviderRegistry = nil

			idpCfg := makeIDPConfig("http://unregistered.example.com/.well-known/openid-configuration")
			result := manageIDPResource(logger, idpCfg)
			assert.Equal(t, tc.wantResult, result)
			assert.Equal(t, tc.wantCreateCount, createCount, "should not call CreateOrUpdateResource when provider is not registered")
		})
	}
}

func TestManageIDPResourceMetadataWriteError(t *testing.T) {
	tests := map[string]struct {
		wantResultEmpty bool
		wantNameStored  bool
		wantCreateCount int
	}{
		"metadata write error returns empty name and stores nothing": {
			wantResultEmpty: true,
			wantNameStored:  false,
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
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
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
			assert.Equal(t, tc.wantResultEmpty, resultName == "")
			assert.Equal(t, tc.wantCreateCount, createCount)
			_, ok := GetAuthProviderRegistry().GetIDPResourceName(idpServer.GetMetadataURL())
			assert.Equal(t, tc.wantNameStored, ok)
		})
	}
}

func TestGetEnvCredentialPoliciesError(t *testing.T) {
	tests := map[string]struct {
		envErr         error
		wantExpiryZero bool
		wantVisibZero  bool
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

func TestManageIDPResource(t *testing.T) {
	testMeta := &oauth.AuthorizationServerMetadata{
		Issuer:                "https://idp.example.com",
		AuthorizationEndpoint: "https://idp.example.com/auth",
		TokenEndpoint:         "https://idp.example.com/token",
		IntrospectionEndpoint: "https://idp.example.com/introspect",
		JwksURI:               "https://idp.example.com/jwks",
	}

	tests := map[string]struct {
		metadata         *oauth.AuthorizationServerMetadata
		createErr        error
		assertResult     func(t *testing.T, result string)
		wantQueryCount   int
		wantCreateCalled bool
	}{
		"nil metadata returns empty without any API call": {
			metadata:       nil,
			assertResult:   func(t *testing.T, result string) { assert.Empty(t, result) },
			wantQueryCount: 0,
		},
		"creates resource when not cached in Engage": {
			metadata:         testMeta,
			assertResult:     func(t *testing.T, result string) { assert.NotEmpty(t, result) },
			wantQueryCount:   1,
			wantCreateCalled: true,
		},
		"create failure returns empty name": {
			metadata:         testMeta,
			createErr:        errors.New("create failed"),
			assertResult:     func(t *testing.T, result string) { assert.Empty(t, result) },
			wantQueryCount:   1,
			wantCreateCalled: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			queryCount := 0
			createCalled := false

			apicClient := &mock.Client{
				GetAPIV1ResourceInstancesMock: func(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
					queryCount++
					return []*apiv1.ResourceInstance{}, nil
				},
				CreateOrUpdateResourceMock: func(ri apiv1.Interface, _ ...apic.UpdateOption) (*apiv1.ResourceInstance, error) {
					createCalled = true
					if tc.createErr != nil {
						return nil, tc.createErr
					}
					inst, _ := ri.AsInstance()
					return inst, nil
				},
				CreateSubResourceMock: func(_ apiv1.ResourceMeta, _ map[string]interface{}) error {
					return nil
				},
				GetEnvironmentMock: func() (*management.Environment, error) {
					return management.NewEnvironment(testEnvName), nil
				},
			}

			_, cleanup := setupIDPLifecycleAgent(t, apicClient, true)
			defer cleanup()

			result := ManageIDPResource(logger, testIDPName, tc.metadata)

			tc.assertResult(t, result)
			assert.Equal(t, tc.wantQueryCount, queryCount)
			assert.Equal(t, tc.wantCreateCalled, createCalled)
		})
	}
}
