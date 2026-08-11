package cache

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createAccessRequest(id, name, appName, instanceID, instanceName string) *management.AccessRequest {
	ri := &management.AccessRequest{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: id,
				References: []v1.Reference{
					{
						Group: management.APIServiceInstanceGVK().Group,
						Kind:  management.APIServiceInstanceGVK().Kind,
						ID:    instanceID,
						Name:  instanceName,
					},
				},
			},
			Name: name,
		},
		Spec: management.AccessRequestSpec{
			ManagedApplication: appName,
			ApiServiceInstance: instanceName,
		},
	}
	ri.CreateHashes()
	return ri
}

// add access request
// get access request by id
// get access request by app name and api id
// delete access request
func TestAccessRequestCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)
	assert.NotNil(t, m)

	cachedAccessReq := m.GetAccessRequest("ac1")
	assert.Nil(t, cachedAccessReq)
	instance1 := createAPIServiceInstance("inst-1", "testAPI", "")
	instance2 := createAPIServiceInstance("inst-2", "testAPI", "testStage")
	instance3 := createAPIServiceInstanceWithVersion("inst-3", "testAPI", "testStage", "testVersion")
	m.AddAPIServiceInstance(instance1)
	m.AddAPIServiceInstance(instance2)
	m.AddAPIServiceInstance(instance3)

	accReq1 := createAccessRequest("ac1", "access-request-1", "app1", "inst-1", "inst-1")
	ar1ri, _ := accReq1.AsInstance()
	accReq2 := createAccessRequest("ac2", "access-request-2", "app2", "inst-2", "inst-2")
	ar2ri, _ := accReq2.AsInstance()
	accReq3 := createAccessRequest("ac3", "access-request-3", "app3", "inst-3", "inst-3")
	ar3ri, _ := accReq3.AsInstance()

	m.AddAccessRequest(ar1ri)
	m.AddAccessRequest(ar2ri)
	m.AddAccessRequest(ar3ri)

	keys := m.GetAccessRequestCacheKeys()
	assert.ElementsMatch(t, []string{"ac1", "ac2", "ac3"}, keys)

	cachedAccessReq = m.GetAccessRequest("ac1")
	assert.Equal(t, ar1ri, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequestByAppAndAPI("app1", "testAPI", "")
	assert.Equal(t, ar1ri, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequestByAppAndAPI("app2", "testAPI", "testStage")
	assert.Equal(t, ar2ri, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequestByAppAndAPIStageVersion("app3", "testAPI", "testStage", "testVersion")
	assert.Equal(t, ar3ri, cachedAccessReq)

	cachedAccessReqs := m.GetAccessRequestsByApp("app1")
	assert.Len(t, cachedAccessReqs, 1)
	assert.Equal(t, ar1ri, cachedAccessReqs[0])

	cachedAccessReqs = m.GetAccessRequestsByApp("app2")
	assert.Len(t, cachedAccessReqs, 1)
	assert.Equal(t, ar2ri, cachedAccessReqs[0])

	cachedAccessReqs = m.GetAccessRequestsByApp("app3")
	assert.Len(t, cachedAccessReqs, 1)
	assert.Equal(t, ar3ri, cachedAccessReqs[0])

	err := m.DeleteAccessRequest("ac1")
	assert.Nil(t, err)

	cachedAccessReq = m.GetAccessRequest("ac1")
	assert.Nil(t, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequest("ac2")
	assert.NotNil(t, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequest("ac3")
	assert.NotNil(t, cachedAccessReq)

	err = m.DeleteAccessRequest("ac1")
	assert.NotNil(t, err)
}

func TestAccessRequestCacheEnabledDefaults(t *testing.T) {
	tests := []struct {
		name      string
		agentType config.AgentType
		expected  bool
	}{
		{
			name:      "discovery agent defaults to disabled",
			agentType: config.DiscoveryAgent,
			expected:  false,
		},
		{
			name:      "traceability agent defaults to enabled",
			agentType: config.TraceabilityAgent,
			expected:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAgentCacheManager(config.NewTestCentralConfig(tc.agentType), false)
			assert.Equal(t, tc.expected, m.IsAccessRequestCacheEnabled())
		})
	}
}

func TestAccessRequestCacheEnabledToggle(t *testing.T) {
	m := NewAgentCacheManager(config.NewTestCentralConfig(config.DiscoveryAgent), false)
	assert.False(t, m.IsAccessRequestCacheEnabled())

	instance := createAPIServiceInstance("inst-1", "testAPI", "")
	m.AddAPIServiceInstance(instance)
	accReq := createAccessRequest("ac1", "access-request-1", "app1", "inst-1", "inst-1")
	ri, _ := accReq.AsInstance()

	// caching is disabled, so adding an AccessRequest should be a no-op
	m.AddAccessRequest(ri)
	assert.Nil(t, m.GetAccessRequest("ac1"))
	assert.Empty(t, m.GetAccessRequestCacheKeys())

	// enabling it should let subsequent adds be cached
	m.SetAccessRequestCacheEnabled(true)
	assert.True(t, m.IsAccessRequestCacheEnabled())
	m.AddAccessRequest(ri)
	assert.Equal(t, ri, m.GetAccessRequest("ac1"))

	// disabling it again stops new adds, but doesn't clear what's already cached
	m.SetAccessRequestCacheEnabled(false)
	accReq2 := createAccessRequest("ac2", "access-request-2", "app2", "inst-1", "inst-1")
	ri2, _ := accReq2.AsInstance()
	m.AddAccessRequest(ri2)
	assert.Nil(t, m.GetAccessRequest("ac2"))
	assert.NotNil(t, m.GetAccessRequest("ac1"))
}

func TestGetAccessRequestsByAPI(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{}, false)

	instance1 := createAPIServiceInstance("inst-1", "testAPI", "")
	instance2 := createAPIServiceInstanceWithVersion("inst-2", "testAPI", "testStage", "testVersion")
	instance3 := createAPIServiceInstance("inst-3", "otherAPI", "")
	m.AddAPIServiceInstance(instance1)
	m.AddAPIServiceInstance(instance2)
	m.AddAPIServiceInstance(instance3)

	accReq1 := createAccessRequest("ac1", "access-request-1", "app1", "inst-1", "inst-1")
	ar1ri, _ := accReq1.AsInstance()
	accReq2 := createAccessRequest("ac2", "access-request-2", "app2", "inst-2", "inst-2")
	ar2ri, _ := accReq2.AsInstance()
	accReq3 := createAccessRequest("ac3", "access-request-3", "app3", "inst-3", "inst-3")
	ar3ri, _ := accReq3.AsInstance()

	m.AddAccessRequest(ar1ri)
	m.AddAccessRequest(ar2ri)
	m.AddAccessRequest(ar3ri)

	// ac1 and ac2 reference instances backed by "testAPI", across different apps/stages/versions
	results := m.GetAccessRequestsByAPI("testAPI")
	assert.ElementsMatch(t, []*v1.ResourceInstance{ar1ri, ar2ri}, results)

	// ac3 references a different API entirely
	results = m.GetAccessRequestsByAPI("otherAPI")
	assert.ElementsMatch(t, []*v1.ResourceInstance{ar3ri}, results)

	// no access requests reference this API
	results = m.GetAccessRequestsByAPI("unknownAPI")
	assert.Empty(t, results)
}
