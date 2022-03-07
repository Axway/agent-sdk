package resource

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/mock"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
)

func createDiscoveryAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
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
				OwningTeam: team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createTraceabilityAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
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
				OwningTeam: team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createGovernanceAgentRes(id, name, dataplane, team string) *v1.ResourceInstance {
	res := &v1alpha1.GovernanceAgent{
		ResourceMeta: v1.ResourceMeta{
			Name: name,
			Metadata: v1.Metadata{
				ID: id,
			},
		},
		Spec: v1alpha1.GovernanceAgentSpec{
			DataplaneType: dataplane,
			Config: map[string]interface{}{
				"team": team,
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func TestNewManager(t *testing.T) {
	cfg := &config.CentralConfiguration{}
	m, err := NewAgentResourceManager(cfg, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, m)

	cfg.AgentName = "Test-DA"
	m, err = NewAgentResourceManager(cfg, nil, nil)
	assert.NotNil(t, err)
	assert.Nil(t, m)

	resource := createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "")
	svcClient := &mock.Client{
		ExecuteAPIMock: func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
			if method == api.PUT {
				return buffer, nil
			}
			return json.Marshal(resource)
		},
	}
	agentResChangeHandlerCall := 0
	f := func() { agentResChangeHandlerCall++ }
	cfg.AgentType = config.DiscoveryAgent
	cfg.AgentName = "Test-DA"
	m, err = NewAgentResourceManager(cfg, svcClient, f)
	assert.Nil(t, err)
	assert.NotNil(t, m)
	m.SetAgentResource(createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "update"))
	assert.Equal(t, 1, agentResChangeHandlerCall)
}

func TestAgentConfigOverride(t *testing.T) {
	tests := []struct {
		name            string
		agentType       config.AgentType
		agentName       string
		resource        *v1.ResourceInstance
		updatedResource *v1.ResourceInstance
	}{
		{
			name:            "DiscoveryAgent override",
			agentType:       config.DiscoveryAgent,
			agentName:       "Test-DA",
			resource:        createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", ""),
			updatedResource: createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "TestTeam"),
		},
		{
			name:            "TraceabilityAgent override",
			agentType:       config.TraceabilityAgent,
			agentName:       "Test-TA",
			resource:        createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", ""),
			updatedResource: createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", "TestTeam"),
		},
		{
			name:            "GovernanceAgent override",
			agentType:       config.GovernanceAgent,
			agentName:       "Test-GA",
			resource:        createGovernanceAgentRes("111", "Test-GA", "test-dataplane", ""),
			updatedResource: createGovernanceAgentRes("111", "Test-GA", "test-dataplane", "TestTeam"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			var resource *v1.ResourceInstance
			svcClient := &mock.Client{
				ExecuteAPIMock: func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
					if method == api.PUT {
						return buffer, nil
					}
					return json.Marshal(resource)
				},
			}
			resource = tc.resource

			agentResChangeHandlerCall := 0
			f := func() { agentResChangeHandlerCall++ }
			m, err := NewAgentResourceManager(cfg, svcClient, f)
			assert.Nil(t, err)
			assert.NotNil(t, m)

			res := m.GetAgentResource()
			assertAgentResource(t, res, tc.resource)
			assert.Equal(t, 0, agentResChangeHandlerCall)

			// Get same resource does not invoke change handler
			m.FetchAgentResource()
			res = m.GetAgentResource()
			assertAgentResource(t, res, tc.resource)
			assert.Equal(t, 0, agentResChangeHandlerCall)

			// Updated resource invokes change handler
			resource = tc.updatedResource
			m.FetchAgentResource()

			res = m.GetAgentResource()
			assertAgentResource(t, res, tc.updatedResource)
			assert.Equal(t, 1, agentResChangeHandlerCall)
		})
	}
}

func TestAgentUpdateStatus(t *testing.T) {
	tests := []struct {
		name      string
		agentType config.AgentType
		agentName string
		resource  *v1.ResourceInstance
	}{
		{
			name:      "DiscoveryAgent override",
			agentType: config.DiscoveryAgent,
			agentName: "Test-DA",
			resource:  createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", ""),
		},
		{
			name:      "TraceabilityAgent override",
			agentType: config.TraceabilityAgent,
			agentName: "Test-TA",
			resource:  createTraceabilityAgentRes("111", "Test-TA", "test-dataplane", ""),
		},
		{
			name:      "GovernanceAgent override",
			agentType: config.GovernanceAgent,
			agentName: "Test-GA",
			resource:  createGovernanceAgentRes("111", "Test-GA", "test-dataplane", ""),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			var response []byte
			svcClient := &mock.Client{
				ExecuteAPIMock: func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
					if method == api.PUT {
						response = buffer
						return buffer, nil
					}
					response, _ = json.Marshal(tc.resource)
					return response, nil
				},
			}

			m, err := NewAgentResourceManager(cfg, svcClient, nil)

			assert.Nil(t, err)
			assert.NotNil(t, m)
			m.UpdateAgentStatus("stopped", "running", "test")
			assertAgentStatusResource(t, response, tc.agentName, "stopped", "running", "test")
		})
	}
}

func assertAgentResource(t *testing.T, res, expectedRes *v1.ResourceInstance) {
	assert.Equal(t, expectedRes.Group, res.Group)
	assert.Equal(t, expectedRes.Kind, res.Kind)
	assert.Equal(t, expectedRes.Name, res.Name)
	assert.Equal(t, expectedRes.Metadata.ID, res.Metadata.ID)
	assert.Equal(t, expectedRes.Spec["dataplane"], res.Spec["dataplane"])
	assert.Equal(t, expectedRes.Spec["config"], res.Spec["config"])
}

func assertAgentStatusResource(t *testing.T, res []byte, agentName, state, previousState, message string) {
	var agentRes map[string]interface{}
	fmt.Println(string(res))
	json.Unmarshal(res, &agentRes)
	statusSubRes := agentRes["status"].(map[string]interface{})

	assert.NotNil(t, statusSubRes)
	assert.Equal(t, state, statusSubRes["state"])
	assert.Equal(t, previousState, statusSubRes["previousState"])
	assert.Equal(t, message, statusSubRes["message"])
}
