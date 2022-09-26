package cache

import (
	"os"
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestMigrateInstanceCount(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{AgentName: "test", GRPCCfg: config.GRPCConfig{Enabled: true}}, true)
	assert.NotNil(t, m)

	api1 := createAPIService("apiID", "apiID", "")
	err := m.AddAPIService(api1)
	assert.Nil(t, err)

	instance1 := createAPIServiceInstance("id1", "apiID", "stage")
	m.AddAPIServiceInstance(instance1)

	instance2 := createAPIServiceInstance("id2", "apiID", "stage2")
	m.AddAPIServiceInstance(instance2)

	// remove the instanceCount map before saving
	defer func() {
		// Remove file if it exists
		_, err := os.Stat("./data")
		if !os.IsExist(err) {
			os.RemoveAll("./data")
		}
	}()

	count := m.GetAPIServiceInstanceCount(api1.Name)
	assert.Equal(t, 2, count)

	// instance count not updated properly check
	m.(*cacheManager).instanceCountMap = cache.New()
	count = m.GetAPIServiceInstanceCount(api1.Name)
	assert.Equal(t, 2, count)

	m.(*cacheManager).migratePersistentCache(instanceCountKey)

	count = m.GetAPIServiceInstanceCount(api1.Name)
	assert.Equal(t, 2, count)
}

func TestMigrateAccessRequest(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{AgentName: "test", GRPCCfg: config.GRPCConfig{Enabled: true}}, true)
	assert.NotNil(t, m)

	ar1 := createAccessRequest("id1", "apiID", "appName1", "instID", "instName")
	ri, _ := ar1.AsInstance()
	fakeAddAccessRequest(m.(*cacheManager).accessRequestMap, ri)

	ar2 := createAccessRequest("id2", "apiID2", "appName1", "instID2", "instName2")
	ri, _ = ar2.AsInstance()
	fakeAddAccessRequest(m.(*cacheManager).accessRequestMap, ri)

	// remove the instanceCount map before saving
	defer func() {
		// Remove file if it exists
		_, err := os.Stat("./data")
		if !os.IsExist(err) {
			os.RemoveAll("./data")
		}
	}()

	ars := m.GetAccessRequestsByApp(ar1.Spec.ManagedApplication)
	assert.Len(t, ars, 0)

	m.(*cacheManager).migratePersistentCache(accReqKey)

	ars = m.GetAccessRequestsByApp(ar1.Spec.ManagedApplication)
	assert.Len(t, ars, 2)
}

func fakeAddAccessRequest(instanceCache cache.Cache, ri *v1.ResourceInstance) {
	if ri == nil {
		return
	}

	ar := &management.AccessRequest{}
	if ar.FromInstance(ri) != nil {
		return
	}

	instanceCache.SetWithSecondaryKey(ar.Metadata.ID, "test-"+ar.Metadata.ID, ri)
}
