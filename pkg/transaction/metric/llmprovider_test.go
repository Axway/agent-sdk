package metric

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type fakeLLMProviderClient struct {
	resources []*v1.ResourceInstance
	err       error
	calls     int
}

func (f *fakeLLMProviderClient) GetAPIV1ResourceInstances(_ map[string]string, _ string) ([]*v1.ResourceInstance, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.resources, nil
}

// fakeLLMProviderCacheManager resolves a fixed primary key -> service name -> instance names chain.
type fakeLLMProviderCacheManager struct {
	primaryKeyToService map[string]string
	serviceToInstances  map[string][]*v1.ResourceInstance
}

func (f *fakeLLMProviderCacheManager) GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance {
	name, ok := f.primaryKeyToService[primaryKey]
	if !ok {
		return nil
	}
	return &v1.ResourceInstance{ResourceMeta: v1.ResourceMeta{Name: name}}
}

func (f *fakeLLMProviderCacheManager) GetAPIServiceInstancesByService(svcName string) []*v1.ResourceInstance {
	return f.serviceToInstances[svcName]
}

// singleInstanceCache builds a fake cache manager where apiID (used directly as the primary
// key, since it carries no SummaryEventProxyIDPrefix in these tests) resolves through svcName
// to a single API service instance named instanceName.
func singleInstanceCache(apiID, svcName, instanceName string) *fakeLLMProviderCacheManager {
	return &fakeLLMProviderCacheManager{
		primaryKeyToService: map[string]string{apiID: svcName},
		serviceToInstances: map[string][]*v1.ResourceInstance{
			svcName: {{ResourceMeta: v1.ResourceMeta{Name: instanceName}}},
		},
	}
}

func newTestResolver(client *fakeLLMProviderClient, cacheMgr *fakeLLMProviderCacheManager) *llmProviderResolver {
	return &llmProviderResolver{
		providers: make(map[string]string),
		logger:    log.NewFieldLogger(),
		getClient: func() llmProviderClient { return client },
		getEnv:    func() string { return "test-env" },
		getCacheMgr: func() llmProviderCacheManager {
			if cacheMgr == nil {
				return nil
			}
			return cacheMgr
		},
	}
}

func llmProviderInstance(id, apiServiceInstance string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: id},
		},
		Spec: map[string]interface{}{
			"apiServiceInstance": apiServiceInstance,
		},
	}
}

func TestLLMProviderResolverGetProviderID(t *testing.T) {
	t.Run("empty api id never calls central", func(t *testing.T) {
		client := &fakeLLMProviderClient{}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-1"))

		id := r.getProviderID("")

		assert.Empty(t, id)
		assert.Equal(t, 0, client.calls)
	})

	t.Run("unresolvable api id never calls central", func(t *testing.T) {
		client := &fakeLLMProviderClient{}
		r := newTestResolver(client, &fakeLLMProviderCacheManager{})

		id := r.getProviderID("api-unknown")

		assert.Empty(t, id)
		assert.Equal(t, 0, client.calls)
	})

	t.Run("nil cache manager never calls central", func(t *testing.T) {
		client := &fakeLLMProviderClient{}
		r := newTestResolver(client, nil)

		id := r.getProviderID("api-1")

		assert.Empty(t, id)
		assert.Equal(t, 0, client.calls)
	})

	t.Run("fetches and resolves provider id on first lookup", func(t *testing.T) {
		client := &fakeLLMProviderClient{
			resources: []*v1.ResourceInstance{
				llmProviderInstance("provider-1", "instance-1"),
				llmProviderInstance("provider-2", "instance-2"),
			},
		}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-1"))

		id := r.getProviderID("api-1")

		assert.Equal(t, "provider-1", id)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("cached hit does not call central again", func(t *testing.T) {
		client := &fakeLLMProviderClient{
			resources: []*v1.ResourceInstance{
				llmProviderInstance("provider-1", "instance-1"),
			},
		}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-1"))

		first := r.getProviderID("api-1")
		second := r.getProviderID("api-1")

		assert.Equal(t, "provider-1", first)
		assert.Equal(t, "provider-1", second)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("unknown instance refreshes every call", func(t *testing.T) {
		client := &fakeLLMProviderClient{
			resources: []*v1.ResourceInstance{
				llmProviderInstance("provider-1", "instance-1"),
			},
		}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-unknown"))

		first := r.getProviderID("api-1")
		second := r.getProviderID("api-1")

		assert.Empty(t, first)
		assert.Empty(t, second)
		assert.Equal(t, 2, client.calls)
	})

	t.Run("client error leaves map unchanged", func(t *testing.T) {
		client := &fakeLLMProviderClient{err: errors.New("boom")}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-1"))

		id := r.getProviderID("api-1")

		assert.Empty(t, id)
		assert.Equal(t, 1, client.calls)
	})

	t.Run("nil client is a no-op", func(t *testing.T) {
		r := &llmProviderResolver{
			providers: make(map[string]string),
			logger:    log.NewFieldLogger(),
			getClient: func() llmProviderClient { return nil },
			getEnv:    func() string { return "test-env" },
			getCacheMgr: func() llmProviderCacheManager {
				return singleInstanceCache("api-1", "svc-1", "instance-1")
			},
		}

		id := r.getProviderID("api-1")

		assert.Empty(t, id)
	})

	t.Run("resources without apiServiceInstance are skipped", func(t *testing.T) {
		client := &fakeLLMProviderClient{
			resources: []*v1.ResourceInstance{
				llmProviderInstance("provider-1", ""),
			},
		}
		r := newTestResolver(client, singleInstanceCache("api-1", "svc-1", "instance-1"))

		id := r.getProviderID("api-1")

		assert.Empty(t, id)
	})
}

func TestLLMProviderResolverGetAPIServiceInstanceName(t *testing.T) {
	t.Run("resolves through service to instance name", func(t *testing.T) {
		r := newTestResolver(&fakeLLMProviderClient{}, singleInstanceCache("api-1", "svc-1", "instance-1"))

		name := r.getAPIServiceInstanceName("api-1")

		assert.Equal(t, "instance-1", name)
	})

	t.Run("unknown api id resolves to empty", func(t *testing.T) {
		r := newTestResolver(&fakeLLMProviderClient{}, &fakeLLMProviderCacheManager{})

		name := r.getAPIServiceInstanceName("api-unknown")

		assert.Empty(t, name)
	})

	t.Run("service with no instances resolves to empty", func(t *testing.T) {
		cacheMgr := &fakeLLMProviderCacheManager{
			primaryKeyToService: map[string]string{"api-1": "svc-1"},
		}
		r := newTestResolver(&fakeLLMProviderClient{}, cacheMgr)

		name := r.getAPIServiceInstanceName("api-1")

		assert.Empty(t, name)
	})
}
