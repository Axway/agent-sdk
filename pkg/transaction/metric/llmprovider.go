package metric

import (
	"strings"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// llmProviderClient is the subset of the central client used to fetch LLMProvider resources.
type llmProviderClient interface {
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*v1.ResourceInstance, error)
}

// llmProviderCacheManager is the subset of the agent cache manager used to resolve the API
// service instance associated with an external API ID.
type llmProviderCacheManager interface {
	GetAPIServiceWithPrimaryKey(primaryKey string) *v1.ResourceInstance
	GetAPIServiceInstancesByService(svcName string) []*v1.ResourceInstance
}

// llmProviderResolver resolves the LLMProvider resource ID associated with an API service instance
type llmProviderResolver struct {
	lock        sync.RWMutex
	providers   map[string]string // API service instance name -> LLM provider ID
	logger      log.FieldLogger
	getClient   func() llmProviderClient
	getEnv      func() string
	getCacheMgr func() llmProviderCacheManager
}

func newLLMProviderResolver(logger log.FieldLogger) *llmProviderResolver {
	return &llmProviderResolver{
		providers: make(map[string]string),
		logger:    logger,
		getClient: func() llmProviderClient {
			client := agent.GetCentralClient()
			if client == nil {
				return nil
			}
			return client
		},
		getEnv: func() string {
			cfg := agent.GetCentralConfig()
			if cfg == nil {
				return ""
			}
			return cfg.GetEnvironmentName()
		},
		getCacheMgr: func() llmProviderCacheManager {
			cacheManager := agent.GetCacheManager()
			if cacheManager == nil {
				return nil
			}
			return cacheManager
		},
	}
}

// getProviderID returns the LLM provider ID associated with the API service instance that the
// given external API ID belongs to. The API ID is resolved to an API service instance name via
// the agent cache; if that instance name is not already known, the full list of LLMProvider
// resources is fetched from Central and the map is rebuilt before looking up the instance again.
func (r *llmProviderResolver) getProviderID(apiID string) string {
	if apiID == "" {
		return ""
	}

	instanceName := r.getAPIServiceInstanceName(apiID)
	if instanceName == "" {
		return ""
	}

	r.lock.RLock()
	id, found := r.providers[instanceName]
	r.lock.RUnlock()
	if found {
		return id
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if id, found := r.providers[instanceName]; found {
		return id
	}
	r.refresh()
	return r.providers[instanceName]
}

// getAPIServiceInstanceName resolves the API service instance name associated with the given
// external API ID via the agent cache.
func (r *llmProviderResolver) getAPIServiceInstanceName(apiID string) string {
	cacheManager := r.getCacheMgr()
	if cacheManager == nil {
		return ""
	}

	externalAPIID := strings.TrimPrefix(apiID, transutil.SummaryEventProxyIDPrefix)
	svc := cacheManager.GetAPIServiceWithPrimaryKey(externalAPIID)
	if svc == nil {
		return ""
	}

	instances := cacheManager.GetAPIServiceInstancesByService(svc.Name)
	if len(instances) == 0 {
		return ""
	}
	return instances[0].Name
}

// fetches LLMProvider resources Central and builds the API service instance -> provider ID map
func (r *llmProviderResolver) refresh() {
	client := r.getClient()
	if client == nil {
		return
	}

	ri := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: management.LLMProviderGVK(),
			Metadata: v1.Metadata{
				Scope: v1.MetadataScope{
					Kind: management.LLMProviderScopes[0],
					Name: r.getEnv(),
				},
			},
		},
	}

	resources, err := client.GetAPIV1ResourceInstances(map[string]string{"fields": "spec.apiServiceInstance,metadata.id"}, ri.GetKindLink())
	if err != nil {
		r.logger.WithError(err).Error("failed to fetch llm provider resources")
		return
	}

	providers := make(map[string]string, len(resources))
	for _, res := range resources {
		instanceName, _ := res.Spec["apiServiceInstance"].(string)
		if instanceName == "" {
			continue
		}
		providers[instanceName] = res.Metadata.ID
	}
	r.providers = providers
}
