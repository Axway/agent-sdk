package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/apic"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func resetResources() {
	agent.agentResource = nil
	agent.isInitialized = false
}

func createCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	cfg.UpdateFromAPIServer = true
	cfg.APICDeployment = "apic"
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "DOSA_1111"
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
	res := &v1alpha1.DiscoveryAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.DiscoveryAgentSpec{
			DataplaneType: dataplane,
			Config: v1alpha1.DiscoveryAgentSpecConfig{
				Filter: filter,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createTraceabilityAgentRes(id, name, dataplane string, processHeaders bool) *v1.ResourceInstance {
	res := &v1alpha1.TraceabilityAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.TraceabilityAgentSpec{
			DataplaneType: dataplane,
			Config: v1alpha1.TraceabilityAgentSpecConfig{
				ProcessHeaders: processHeaders,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

type TestConfig struct {
	resourceChanged bool
	childCfg        config.IResourceConfigCallback
}

func (a *TestConfig) ApplyResources(agentResource *v1.ResourceInstance) error {
	a.resourceChanged = true
	return nil
}

func TestAgentInitialize(t *testing.T) {
	const (
		daName = "discovery"
		taName = "traceability"
	)

	teams := []apic.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "v7",
			Title:    "v7",
		},
	}
	discoveryAgentRes := createDiscoveryAgentRes("111", daName, "v7-dataplane", "")
	traceabilityAgentRes := createTraceabilityAgentRes("111", taName, "v7-dataplane", false)

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7/discoveryagents/"+daName) {
			buf, err := json.Marshal(discoveryAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7/traceabilityagents/"+taName) {
			buf, err := json.Marshal(traceabilityAgentRes)
			log.Error(err)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
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
	AgentResourceType = v1alpha1.DiscoveryAgentResourceName
	cfg.AgentName = daName
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	assertResource(t, da, discoveryAgentRes)

	cfg.AgentType = config.TraceabilityAgent
	AgentResourceType = v1alpha1.TraceabilityAgentResourceName
	cfg.AgentName = taName
	agent.isInitialized = false
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
	traceabilityAgentRes = createTraceabilityAgentRes("111", taName, "v7-dataplane", true)
	resetResources()

	agentResChangeHandlerCall := 0
	OnAgentResourceChange(func() { agentResChangeHandlerCall++ })

	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	assertResource(t, da, traceabilityAgentRes)
	assert.Equal(t, 0, agentResChangeHandlerCall)
}

func TestAgentConfigOverride(t *testing.T) {
	const (
		daName = "discovery"
		taName = "traceability"
	)

	teams := []apic.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}
	environmentRes := &v1alpha1.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "v7",
			Title:    "v7",
		},
	}
	discoveryAgentRes := createDiscoveryAgentRes("111", daName, "v7-dataplane", "")
	traceabilityAgentRes := createTraceabilityAgentRes("111", taName, "v7-dataplane", false)

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7/discoveryagents/"+daName) {
			buf, _ := json.Marshal(discoveryAgentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7/traceabilityagents/"+taName) {
			buf, _ := json.Marshal(traceabilityAgentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/v7") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")

	AgentResourceType = v1alpha1.DiscoveryAgentResourceName
	cfg.AgentName = "discovery"
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	da := GetAgentResource()
	assertResource(t, da, discoveryAgentRes)

}

func assertAgentResource(t *testing.T, res, expectedRes *v1.ResourceInstance) {
	assert.Equal(t, expectedRes.Group, res.Group)
	assert.Equal(t, expectedRes.Kind, res.Kind)
	assert.Equal(t, expectedRes.Name, res.Name)
	assert.Equal(t, expectedRes.Metadata.ID, res.Metadata.ID)
	assert.Equal(t, expectedRes.Spec["dataplane"], res.Spec["dataplane"])
	assert.Equal(t, expectedRes.Spec["config"], res.Spec["config"])
}

func assertResource(t *testing.T, res, expectedRes *v1.ResourceInstance) {
	assert.Equal(t, expectedRes.Group, res.Group)
	assert.Equal(t, expectedRes.Kind, res.Kind)
	assert.Equal(t, expectedRes.Name, res.Name)
	assert.Equal(t, expectedRes.Metadata.ID, res.Metadata.ID)
	assert.Equal(t, expectedRes.Spec, res.Spec)
}

func Test_newManager(t *testing.T) {
	manager, err := newWatchManager("https://abc.com:5443", "123", true, apic.MockTokenGetter, "")

	assert.Nil(t, err)
	assert.NotNil(t, manager)
}
