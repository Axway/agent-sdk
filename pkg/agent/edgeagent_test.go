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
	agent.dataplaneResource = nil
	agent.isInitialized = false
}
func createCentralCfg(url, env string) *config.CentralConfiguration {
	cfg := config.NewCentralConfig(config.DiscoveryAgent).(*config.CentralConfiguration)
	cfg.URL = url
	cfg.TenantID = "123456"
	cfg.Environment = env
	authCfg := cfg.Auth.(*config.AuthConfiguration)
	authCfg.URL = url + "/auth"
	authCfg.Realm = "Broker"
	authCfg.ClientID = "DOSA_1111"
	authCfg.PrivateKey = "../transaction/testdata/private_key.pem"
	authCfg.PublicKey = "../transaction/testdata/public_key"
	return cfg
}

func createEdgeDiscoveryAgentRes(id, name, dataplane, filter string) *v1.ResourceInstance {
	res := &v1alpha1.EdgeDiscoveryAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.EdgeDiscoveryAgentSpec{
			Dataplane: dataplane,
			Config: v1alpha1.DiscoveryAgentSpecConfig{
				Filter: filter,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createEdgeTraceabilityAgentRes(id, name, dataplane string, processHeaders bool) *v1.ResourceInstance {
	res := &v1alpha1.EdgeTraceabilityAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.EdgeTraceabilityAgentSpec{
			Dataplane: dataplane,
			Config: v1alpha1.TraceabilityAgentSpecConfig{
				ProcessHeaders: processHeaders,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createEdgeDataplaneRes(id, name, host string, apiManagerPort, apigwPort int) *v1.ResourceInstance {
	res := &v1alpha1.EdgeDataplane{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.EdgeDataplaneSpec{
			ApiManager: v1alpha1.EdgeDataplaneSpecApiManager{
				Host: host,
				Port: int32(apiManagerPort),
			},
			ApiGatewayManager: v1alpha1.EdgeDataplaneSpecApiGatewayManager{
				Host: host,
				Port: int32(apigwPort),
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

func (a *TestConfig) ApplyResources(dataplaneResource *v1.ResourceInstance, agentResource *v1.ResourceInstance) error {
	a.resourceChanged = true
	return nil
}

func TestEdgeAgentInitialize(t *testing.T) {
	var edgeDataplaneRes, edgeDiscoveryAgentRes, edgeTraceabilitAgentRes *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Println(req.RequestURI)
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments") {
			if strings.Contains(req.RequestURI, "v7/edgedataplanes/v7-dataplane") {
				buf, _ := json.Marshal(edgeDataplaneRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "/v7/edgediscoveryagents/v7-discovery") {
				buf, _ := json.Marshal(edgeDiscoveryAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "v7/edgetraceabilityagents/v7-traceability") {
				buf, _ := json.Marshal(edgeTraceabilitAgentRes)
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
	dp := GetDataplaneResource()
	assert.Nil(t, da)
	assert.Nil(t, dp)

	edgeDataplaneRes = createEdgeDataplaneRes("111", "v7-dataplane", "localhost", 8075, 8090)
	edgeDiscoveryAgentRes = createEdgeDiscoveryAgentRes("111", "v7-disconery", "v7-dataplane", "")
	edgeTraceabilitAgentRes = createEdgeTraceabilityAgentRes("111", "v7-traceability", "v7-dataplane", false)

	AgentResourceType = v1alpha1.EdgeDiscoveryAgentResource
	cfg.AgentName = "v7-discovery"
	resetResources()
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	dp = GetDataplaneResource()
	assertResource(t, dp, edgeDataplaneRes)
	assertResource(t, da, edgeDiscoveryAgentRes)

	AgentResourceType = v1alpha1.EdgeTraceabilityAgentResource
	cfg.AgentName = "v7-traceability"
	agent.isInitialized = false
	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	dp = GetDataplaneResource()
	assertResource(t, dp, edgeDataplaneRes)
	assertResource(t, da, edgeTraceabilitAgentRes)

	agentCfg := &TestConfig{
		resourceChanged: false,
	}

	ApplyResouceToConfig(agentCfg)

	assert.True(t, agentCfg.resourceChanged)

	// Test for resource change
	edgeDataplaneRes = createEdgeDataplaneRes("111", "v7-dataplane", "localhost", 9075, 8090)
	resetResources()

	agentResChangeHandlerCall := 0
	OnAgentResourceChange(func() { agentResChangeHandlerCall++ })

	err = Initialize(cfg)
	assert.Nil(t, err)

	da = GetAgentResource()
	dp = GetDataplaneResource()
	assertResource(t, dp, edgeDataplaneRes)
	assertResource(t, da, edgeTraceabilitAgentRes)
	assert.Equal(t, 1, agentResChangeHandlerCall)
}

func TestAgentConfigOverride(t *testing.T) {
	var edgeDataplaneRes, edgeDiscoveryAgentRes, edgeTraceabilitAgentRes *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		fmt.Println(req.RequestURI)
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments") {
			if strings.Contains(req.RequestURI, "v7/edgedataplanes/v7-dataplane") {
				buf, _ := json.Marshal(edgeDataplaneRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "/v7/edgediscoveryagents/v7-discovery") {
				buf, _ := json.Marshal(edgeDiscoveryAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
			if strings.Contains(req.RequestURI, "v7/edgetraceabilityagents/v7-traceability") {
				buf, _ := json.Marshal(edgeTraceabilitAgentRes)
				fmt.Println("Res:" + string(buf))
				resp.Write(buf)
			}
		}
	}))

	defer s.Close()

	cfg := createCentralCfg(s.URL, "v7")

	edgeDataplaneRes = createEdgeDataplaneRes("111", "v7-dataplane", "localhost", 8075, 8090)
	edgeDiscoveryAgentRes = createEdgeDiscoveryAgentRes("111", "v7-disconery", "v7-dataplane", "")
	edgeTraceabilitAgentRes = createEdgeTraceabilityAgentRes("111", "v7-traceability", "v7-dataplane", false)

	AgentResourceType = v1alpha1.EdgeDiscoveryAgentResource
	cfg.AgentName = "v7-discovery"
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	da := GetAgentResource()
	dp := GetDataplaneResource()
	assertResource(t, dp, edgeDataplaneRes)
	assertResource(t, da, edgeDiscoveryAgentRes)

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
