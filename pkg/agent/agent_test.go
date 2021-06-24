package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	var discoveryAgentRes, traceabilityAgentRes *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Println(req.RequestURI)
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments") {
			if strings.Contains(req.RequestURI, "/v7/discoveryagents/"+daName) {
				buf, _ := json.Marshal(discoveryAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "v7/traceabilityagents/"+taName) {
				buf, _ := json.Marshal(traceabilityAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")
	// Test with no agent name - config to be validate successfully as no calls made to get agent and dataplane resource
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)
	da := GetAgentResource()
	assert.Nil(t, da)

	discoveryAgentRes = createDiscoveryAgentRes("111", daName, "v7-dataplane", "")
	traceabilityAgentRes = createTraceabilityAgentRes("111", taName, "v7-dataplane", false)

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
	assert.Equal(t, 1, agentResChangeHandlerCall)
}

func TestAgentConfigOverride(t *testing.T) {
	const (
		daName = "discovery"
		taName = "traceability"
	)
	var discoveryAgentRes, traceabilityAgentRes *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Println(req.RequestURI)
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments") {
			if strings.Contains(req.RequestURI, "/v7/discoveryagents/"+daName) {
				buf, _ := json.Marshal(discoveryAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "v7/traceabilityagents/"+taName) {
				buf, _ := json.Marshal(traceabilityAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")

	discoveryAgentRes = createDiscoveryAgentRes("111", daName, "v7-dataplane", "")
	traceabilityAgentRes = createTraceabilityAgentRes("111", taName, "v7-dataplane", false)

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
