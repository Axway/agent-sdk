package resource

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/util/errors"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/json"
)

func createDiscoveryAgentRes(id, name, dataplane, teamID string) *v1.ResourceInstance {
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
				Owner: &v1.Owner{
					Type: v1.TeamOwner,
					ID:   teamID,
				},
			},
		},
	}
	instance, _ := res.AsInstance()
	return instance
}

func createTraceabilityAgentRes(id, name, dataplane, teamID string) *v1.ResourceInstance {
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
				Owner: &v1.Owner{
					Type: v1.TeamOwner,
					ID:   teamID,
				},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			var resource *v1.ResourceInstance
			svcClient := &mock.Client{
				GetResourceMock: func(url string) (*v1.ResourceInstance, error) {
					return resource, nil
				},
				CreateSubResourceMock: func(rm v1.ResourceMeta, subs map[string]interface{}) error {
					resource.SubResources = subs
					return nil
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
			name:      "Create DA resource",
			agentType: config.DiscoveryAgent,
			agentName: "env-da",
			resource:  nil,
		},
		{
			name:      "Create TA resource",
			agentType: config.TraceabilityAgent,
			agentName: "env-ta",
			resource:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.CentralConfiguration{}
			cfg.AgentName = tc.agentName
			cfg.AgentType = tc.agentType

			var resource *v1.ResourceInstance
			svcClient := &mock.Client{
				GetResourceMock: func(url string) (*v1.ResourceInstance, error) {
					if resource == nil {
						return nil, errors.New(1111, "404 - Not found")
					}
					return resource, nil
				},
				CreateSubResourceMock: func(rm v1.ResourceMeta, subs map[string]interface{}) error {
					resource.SubResources = subs
					return nil
				},
				CreateResourceInstanceMock: func(ri v1.Interface) (*v1.ResourceInstance, error) {
					resource, _ = ri.AsInstance()
					tc.agentName = ri.GetName()
					return resource, nil
				},
			}
			resource = tc.resource

			m, err := NewAgentResourceManager(cfg, svcClient, nil)

			assert.Nil(t, err)
			assert.NotNil(t, m)
			m.UpdateAgentStatus("stopped", "running", "test")
			assertAgentStatusResource(t, resource, tc.agentName, "stopped", "running", "test")
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

func assertAgentStatusResource(t *testing.T, agentRes *v1.ResourceInstance, agentName, state, previousState, message string) {
	statusSubRes := agentRes.GetSubResource("status").(management.DiscoveryAgentStatus)

	assert.NotNil(t, statusSubRes)
	assert.Equal(t, state, statusSubRes.State)
	assert.Equal(t, previousState, statusSubRes.PreviousState)
	assert.Equal(t, message, statusSubRes.Message)
}
