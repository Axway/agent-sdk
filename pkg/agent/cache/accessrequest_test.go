package cache

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createAccessRequest(id, name, appName, instanceID, instanceName string) *mv1.AccessRequest {
	return &mv1.AccessRequest{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: id,
				References: []v1.Reference{
					{
						ID:   instanceID,
						Name: instanceName,
					},
				},
			},
			Name: name,
		},
		Spec: mv1.AccessRequestSpec{
			ManagedApplication: appName,
			ApiServiceInstance: instanceName,
		},
	}
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
	m.AddAPIServiceInstance(instance1)
	m.AddAPIServiceInstance(instance2)

	accReq1 := createAccessRequest("ac1", "access-request-1", "app1", "inst-1", "inst-1")
	accReq2 := createAccessRequest("ac2", "access-request-2", "app2", "inst-2", "inst-2")

	m.AddAccessRequest(accReq1)
	m.AddAccessRequest(accReq2)

	cachedAccessReq = m.GetAccessRequest("ac1")
	assert.Equal(t, accReq1, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequestByAppAndAPI("app1", "testAPI", "")
	assert.Equal(t, accReq1, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequestByAppAndAPI("app2", "testAPI", "testStage")
	assert.Equal(t, accReq2, cachedAccessReq)

	err := m.DeleteAccessRequest("ac1")
	assert.Nil(t, err)

	cachedAccessReq = m.GetAccessRequest("ac1")
	assert.Nil(t, cachedAccessReq)

	cachedAccessReq = m.GetAccessRequest("ac2")
	assert.NotNil(t, cachedAccessReq)

	err = m.DeleteAccessRequest("ac1")
	assert.NotNil(t, err)
}
