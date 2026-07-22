package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	testDataplaneV7          = "v7-dataplane"
	testAuthTokenResponse    = `{"access_token":"somevalue","expires_in": 12235677}`
	testEnvironmentsV7URL    = "/apis/management/v1/environments/v7"
	testDiscoveryAgentsV7URL = testEnvironmentsV7URL + "/discoveryagents/"
	testPlatformTeamsURL     = "/api/v1/platformTeams"
)

func resetResources() {
	agent.agentResourceManager = nil
	if agent.cacheManager != nil {
		agent.cacheManager.ApplyResourceReadLock()
		defer agent.cacheManager.ReleaseResourceReadLock()
		agent.cacheManager = nil
	}
	agent.isInitialized = false
	agent.apicClient = nil
	agent.agentFeaturesCfg = nil
}

func createCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.SingleURL = url
	cfg.PlatformURL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	cfg.APICDeployment = "apic"
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "serviceaccount_1111"
	authCfg.PrivateKey = "../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../transaction/testdata/public_key"
	return cfg
}

func createOfflineCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.TraceabilityAgent).(*config.CentralConfiguration)
	cfg.EnvironmentID = "abc123"
	cfg.UsageReporting.(*config.UsageReportingConfiguration).Offline = true
	return cfg
}

func createDiscoveryAgentRes(id, name, dataplane, filter string) *v1.ResourceInstance {
	res := &management.DiscoveryAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: management.DiscoveryAgentSpec{
			DataplaneType: dataplane,
			Config: management.DiscoveryAgentSpecConfig{
				Filter: filter,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createTraceabilityAgentRes(id, name, dataplane string, processHeaders bool) *v1.ResourceInstance {
	res := &management.TraceabilityAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: management.TraceabilityAgentSpec{
			DataplaneType: dataplane,
			Config: management.TraceabilityAgentSpecConfig{
				ProcessHeaders: processHeaders,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createComplianceAgentRes(id, name, dataplane string) *v1.ResourceInstance {
	res := &management.ComplianceAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: management.ComplianceAgentSpec{
			DataplaneType: dataplane,
			Config:        management.ComplianceAgentSpecConfig{},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

type TestConfig struct {
	resourceChanged bool
}

func (a *TestConfig) ApplyResources(agentResource *v1.ResourceInstance) error {
	a.resourceChanged = true
	return nil
}

func TestAgentInitialize(t *testing.T) {
	const (
		daName = "discovery"
		taName = "traceability"
		caName = "compliance"
	)

	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &management.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "v7",
			Title:    "v7",
		},
	}
	discoveryAgentRes := createDiscoveryAgentRes("111", daName, testDataplaneV7, "")
	traceabilityAgentRes := createTraceabilityAgentRes("111", taName, testDataplaneV7, false)
	complianceAgentRes := createComplianceAgentRes("111", caName, "ca-dataplane")

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := testAuthTokenResponse
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, testDiscoveryAgentsV7URL+daName) {
			buf, err := json.Marshal(discoveryAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/v7/traceabilityagents/"+taName) {
			buf, err := json.Marshal(traceabilityAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/v7/complianceagents/"+caName) {
			buf, err := json.Marshal(complianceAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testEnvironmentsV7URL) {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testPlatformTeamsURL) {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))

	defer s.Close()

	cfg := createOfflineCentralCfg(s.URL, "v7")
	// Test with offline mode
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)
	da := GetAgentResource()
	assert.Nil(t, da)

	cfg = createCentralCfg(s.URL, "v7")
	// Test with no agent name - config to be validate successfully as no calls made to get agent and dataplane resource
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)
	da = GetAgentResource()
	assert.Nil(t, da)

	cfg.AgentType = config.DiscoveryAgent
	cfg.AgentName = daName
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	assertResource(t, da, discoveryAgentRes)

	cfg.AgentType = config.ComplianceAgent
	cfg.AgentName = caName
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	ca := GetAgentResource()
	assertResource(t, ca, complianceAgentRes)

	cfg.AgentType = config.TraceabilityAgent
	cfg.AgentName = taName
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	assertResource(t, da, traceabilityAgentRes)

	agentCfg := &TestConfig{
		resourceChanged: false,
	}

	ApplyResourceToConfig(agentCfg)

	assert.True(t, agentCfg.resourceChanged)

	// Test for resource change
	traceabilityAgentRes = createTraceabilityAgentRes("111", taName, testDataplaneV7, true)
	resetResources()

	agentResChangeHandlerCall := 0
	OnAgentResourceChange(func() { agentResChangeHandlerCall++ })

	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	assertResource(t, da, traceabilityAgentRes)
	assert.Equal(t, 0, agentResChangeHandlerCall)
}

func TestAgentEntitlements(t *testing.T) {
	const daName = "discovery"

	teams := []definitions.PlatformTeam{}
	entitlements := definitions.OrgEntitlementsResponse{
		Success: true,
		Result: definitions.OrgEntitlements{
			Entitlements: map[string]interface{}{
				"traceability": true,
				"discovery":    true,
				"compliance":   1,
				"expiredBool":  false,
				"expiredFloat": 0,
			},
		},
	}
	environmentRes := &management.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "v7",
			Title:    "v7",
		},
	}
	discoveryAgentRes := createDiscoveryAgentRes("111", daName, testDataplaneV7, "")

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := testAuthTokenResponse
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, testDiscoveryAgentsV7URL+daName) {
			buf, err := json.Marshal(discoveryAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testEnvironmentsV7URL) {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testPlatformTeamsURL) {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/org/123456") {
			buf, _ := json.Marshal(entitlements)
			resp.Write(buf)
			return
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")
	// initialize the agnet and validated the expected entitlements
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)
	da := GetAgentResource()
	assert.Nil(t, da)

	// get the entitlements from the agent
	getEntitlements()

	// Validate the entitlements
	assert.NotNil(t, agent.entitlements)
	assert.Len(t, agent.entitlements, 3)
	assert.NotContains(t, agent.entitlements, "expired")
	assert.Contains(t, agent.entitlements, "discovery")
	assert.Contains(t, agent.entitlements, "traceability")
	assert.Contains(t, agent.entitlements, "compliance")
}

func TestInitEnvironment(t *testing.T) {
	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := management.NewEnvironment("v7")
	environmentRes.Title = "v7"
	environmentRes.Metadata.ID = "123"

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := testAuthTokenResponse
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, testEnvironmentsV7URL) {
			if req.Method == "GET" {
				buf, _ := json.Marshal(environmentRes)
				resp.Write(buf)
			} else if req.Method == "PUT" {
				subRes := &management.Environment{}
				json.NewDecoder(req.Body).Decode(subRes)
				environmentRes.ResourceMeta.SubResources = subRes.ResourceMeta.SubResources
			}
			return
		}

		if strings.Contains(req.RequestURI, testPlatformTeamsURL) {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")
	cfg.AgentType = config.GenericService
	agent.cfg = cfg
	initializeTokenRequester(agent.cfg)
	apiClient := apic.New(agent.cfg, agent.tokenRequester, agent.cacheManager)
	// Test with no agent name - config to be validate successfully as no calls made to get agent and dataplane resource

	defer resetResources()
	err := initEnvResources(agent.cfg, apiClient)
	assert.Nil(t, err)

	cfg = createCentralCfg(s.URL, "v7")
	cfg.AgentType = config.DiscoveryAgent
	agent.cfg = cfg
	err = initEnvResources(agent.cfg, apiClient)
	assert.Nil(t, err)

	cfg = createCentralCfg(s.URL, "v7")
	cfg.AgentType = config.TraceabilityAgent
	agent.cfg = cfg
	err = initEnvResources(agent.cfg, apiClient)
	assert.Nil(t, err)
}

// TestInitEnvResourcesUnmanagedEnvironment verifies that initEnvResources derives the
// unmanaged-environment flag from the Environment resource's References.ManagedEnvironments,
// independent of AxwayManaged (which reflects hosting location, not clone/reference status).
func TestInitEnvResourcesUnmanagedEnvironment(t *testing.T) {
	cases := map[string]struct {
		managedEnvironments []string
		axwayManaged        bool
		wantUnmanaged       bool
	}{
		"no environment references reports managed": {
			managedEnvironments: nil,
			wantUnmanaged:       false,
		},
		"environment referencing another environment reports unmanaged": {
			managedEnvironments: []string{"real-env"},
			wantUnmanaged:       true,
		},
		"multiple environment references reports unmanaged": {
			managedEnvironments: []string{"real-env-1", "real-env-2"},
			wantUnmanaged:       true,
		},
		"axway-managed hosting with environment references still reports unmanaged": {
			managedEnvironments: []string{"real-env"},
			axwayManaged:        true,
			wantUnmanaged:       true,
		},
		"axway-managed hosting with no environment references reports managed": {
			managedEnvironments: nil,
			axwayManaged:        true,
			wantUnmanaged:       false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			environmentRes := management.NewEnvironment("v7")
			environmentRes.Title = "v7"
			environmentRes.Metadata.ID = "123"
			environmentRes.Spec.AxwayManaged = tc.axwayManaged
			environmentRes.References.ManagedEnvironments = tc.managedEnvironments

			s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.RequestURI, "/auth") {
					resp.Write([]byte(testAuthTokenResponse))
					return
				}
				if strings.Contains(req.RequestURI, testEnvironmentsV7URL) {
					buf, _ := json.Marshal(environmentRes)
					resp.Write(buf)
					return
				}
				if strings.Contains(req.RequestURI, testPlatformTeamsURL) {
					resp.Write([]byte("[]"))
					return
				}
			}))
			defer s.Close()

			cfg := createCentralCfg(s.URL, "v7")
			cfg.AgentType = config.TraceabilityAgent
			cfg.SetTeamID("existing-team-id") // skip the team lookup call, not under test here
			agent.cfg = cfg
			initializeTokenRequester(agent.cfg)
			apiClient := apic.New(agent.cfg, agent.tokenRequester, agent.cacheManager)

			defer resetResources()
			err := initEnvResources(agent.cfg, apiClient)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantUnmanaged, cfg.IsUnmanagedEnvironment())
		})
	}
}

func TestAgentConfigOverride(t *testing.T) {
	const (
		daName = "discovery"
		taName = "traceability"
	)

	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &management.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "v7",
			Title:    "v7",
		},
	}
	discoveryAgentRes := createDiscoveryAgentRes("111", daName, testDataplaneV7, "")
	traceabilityAgentRes := createTraceabilityAgentRes("111", taName, testDataplaneV7, false)

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := testAuthTokenResponse
			resp.Write([]byte(token))
		}

		if strings.Contains(req.RequestURI, testDiscoveryAgentsV7URL+daName) {
			buf, _ := json.Marshal(discoveryAgentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/v7/traceabilityagents/"+taName) {
			buf, _ := json.Marshal(traceabilityAgentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testEnvironmentsV7URL) {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, testPlatformTeamsURL) {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")

	cfg.AgentName = "discovery"
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	da := GetAgentResource()
	assertResource(t, da, discoveryAgentRes)

}

func TestAgentAgentFeaturesOnByDefault(t *testing.T) {
	cfg := createCentralCfg("http://test", "v7")
	resetResources()
	err := Initialize(cfg)
	assert.NoError(t, err)

	// Assert the agent features are on by default
	assert.True(t, agent.agentFeaturesCfg.ConnectionToCentralEnabled())
	assert.True(t, agent.agentFeaturesCfg.ProcessSystemSignalsEnabled())
	assert.True(t, agent.agentFeaturesCfg.VersionCheckerEnabled())

	assert.NotNil(t, agent.apicClient)
}

func TestAgentAgentFeaturesDisabled(t *testing.T) {
	// Create invalid Central config
	cfg := config.NewCentralConfig(config.GenericService).(*config.CentralConfiguration)
	resetResources()
	agentFeatures := &config.AgentFeaturesConfiguration{
		ConnectToCentral:     false,
		ProcessSystemSignals: false,
		VersionChecker:       false,
	}
	err := InitializeWithAgentFeatures(cfg, agentFeatures, nil)
	assert.NoError(t, err) // This asserts central config is not being validated as ConnectToCentral is false

	assert.False(t, agent.agentFeaturesCfg.ConnectionToCentralEnabled())
	assert.False(t, agent.agentFeaturesCfg.ProcessSystemSignalsEnabled())
	assert.False(t, agent.agentFeaturesCfg.VersionCheckerEnabled())

	// Assert no api client
	assert.Nil(t, agent.apicClient)
}

func assertResource(t *testing.T, res, expectedRes *v1.ResourceInstance) {
	assert.Equal(t, expectedRes.Group, res.Group)
	assert.Equal(t, expectedRes.Kind, res.Kind)
	assert.Equal(t, expectedRes.Name, res.Name)
	assert.Equal(t, expectedRes.Metadata.ID, res.Metadata.ID)
	assert.Equal(t, expectedRes.Spec, res.Spec)
}
