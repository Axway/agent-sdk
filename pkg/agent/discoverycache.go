package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize   = 100
	healthcheckEndpoint = "central"
	apiServerFields     = "metadata,name,title,owner,attributes"
	queryFormatString   = "%s>\"%s\""
)

var discoveryCacheLock *sync.Mutex

func init() {
	discoveryCacheLock = &sync.Mutex{}
}

type discoveryCache struct {
	jobs.Job
	lastServiceTime      time.Time
	lastInstanceTime     time.Time
	lastCategoryTime     time.Time
	refreshAll           bool
	getHCStatus          hc.GetStatusLevel
	instanceCacheLock    *sync.Mutex
	agentResourceManager resource.Manager
}

func newDiscoveryCache(agentResourceManager resource.Manager, getAll bool, instanceCacheLock *sync.Mutex) *discoveryCache {
	return &discoveryCache{
		lastServiceTime:      time.Time{},
		lastInstanceTime:     time.Time{},
		lastCategoryTime:     time.Time{},
		refreshAll:           getAll,
		instanceCacheLock:    instanceCacheLock,
		agentResourceManager: agentResourceManager,
		getHCStatus:          hc.GetStatus,
	}
}

//Ready -
func (j *discoveryCache) Ready() bool {
	status := j.getHCStatus(healthcheckEndpoint)
	return status == hc.OK
}

//Status -
func (j *discoveryCache) Status() error {
	status := j.getHCStatus(healthcheckEndpoint)
	if status == hc.OK {
		return nil
	}
	return fmt.Errorf("could not establish a connection to APIC to update the cache")
}

//Execute -
func (j *discoveryCache) Execute() error {
	discoveryCacheLock.Lock()
	defer discoveryCacheLock.Unlock()
	log.Trace("executing API cache update job")
	j.updateAPICache()
	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		j.updateAPIServiceInstancesCache()
		j.updateCategoryCache()
	}
	if j.agentResourceManager != nil {
		j.agentResourceManager.FetchAgentResource()
	}
	return agent.instanceValidatorJob.Execute()
}

func (j *discoveryCache) updateAPICache() {
	log.Trace("updating API cache")

	// Update cache with published resources
	existingAPIs := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastServiceTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(apiV1.APIServerTimeFormat))
	}
	apiServices, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetServicesURL(), apiServerPageSize)

	for _, apiService := range apiServices {
		if _, valid := apiService.Attributes[definitions.AttrExternalAPIID]; !valid {
			continue // skip service without external api id
		}
		// Update the lastServiceTime based on the newest service found
		thisTime := time.Time(apiService.Metadata.Audit.CreateTimestamp)
		if j.lastServiceTime.Before(thisTime) {
			j.lastServiceTime = thisTime
		}

		externalAPIID := agent.cacheManager.AddAPIService(apiService)
		if externalAPIPrimaryKey, found := apiService.Attributes[definitions.AttrExternalAPIPrimaryKey]; found {
			existingAPIs[externalAPIPrimaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}
	}

	if j.refreshAll {
		// Remove items that are not published as Resources
		cacheKeys := agent.cacheManager.GetAPIServiceKeys()
		for _, key := range cacheKeys {
			if _, ok := existingAPIs[key]; !ok {
				agent.cacheManager.DeleteAPIService(key)
			}
		}
	}
}

func (j *discoveryCache) updateAPIServiceInstancesCache() {
	if agent.apiValidator == nil {
		return
	}

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastInstanceTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(apiV1.APIServerTimeFormat))
	}

	j.lastInstanceTime = time.Now()
	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetInstancesURL(), apiServerPageSize)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances"))
		return
	}

	j.instanceCacheLock.Lock()
	defer j.instanceCacheLock.Unlock()
	if j.refreshAll {
		agent.cacheManager.DeleteAllAPIServiceInstance()
	}
	for _, instance := range serviceInstances {
		if _, valid := instance.Attributes[definitions.AttrExternalAPIID]; !valid {
			continue // skip instance without external api id
		}
		agent.cacheManager.AddAPIServiceInstance(instance)
		if !j.refreshAll {
			// Update the lastInstanceTime based on the newest instance found
			thisTime := time.Time(instance.Metadata.Audit.CreateTimestamp)
			if j.lastInstanceTime.Before(thisTime) {
				j.lastInstanceTime = thisTime
			}
		}
	}
}

func (j *discoveryCache) updateCategoryCache() {
	log.Trace("updating category cache")

	// Update cache with published resources
	existingCategories := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastCategoryTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastCategoryTime.Format(apiV1.APIServerTimeFormat))
	}
	categories, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetCategoriesURL(), apiServerPageSize)

	for _, category := range categories {
		// Update the lastCategoryTime based on the newest category found
		thisTime := time.Time(category.Metadata.Audit.CreateTimestamp)
		if j.lastCategoryTime.Before(thisTime) {
			j.lastCategoryTime = thisTime
		}

		agent.cacheManager.AddCategory(category)
		existingCategories[category.Name] = true
	}

	if j.refreshAll {
		// Remove categories that no longer exist
		cacheKeys := agent.cacheManager.GetCategoryKeys()
		for _, key := range cacheKeys {
			if _, ok := existingCategories[key]; !ok {
				agent.cacheManager.DeleteCategory(key)
			}
		}
	}
}
