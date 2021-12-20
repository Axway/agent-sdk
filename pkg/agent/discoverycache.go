package agent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize        = 100
	healthcheckEndpoint      = "central"
	attributesQueryParam     = "attributes."
	apiServerFields          = "name,title,attributes"
	serviceInstanceCache     = "ServiceInstances"
	serviceInstanceNameCache = "ServiceInstanceNames"
	queryFormatString        = "%s>\"%s\""
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
	}
}

//Ready -
func (j *discoveryCache) Ready() bool {
	status := hc.GetStatus(healthcheckEndpoint)
	return status == hc.OK
}

//Status -
func (j *discoveryCache) Status() error {
	status := hc.GetStatus(healthcheckEndpoint)
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
		j.updatePIServiceInstancesCache()
		j.updateCategoryCache()
	}
	if j.agentResourceManager != nil {
		j.agentResourceManager.FetchAgentResource()
	}
	return nil
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
		if _, valid := apiService.Attributes[apic.AttrExternalAPIID]; !valid {
			continue // skip service without external api id
		}
		// Update the lastServiceTime based on the newest service found
		thisTime := time.Time(apiService.Metadata.Audit.CreateTimestamp)
		if j.lastServiceTime.Before(thisTime) {
			j.lastServiceTime = thisTime
		}

		externalAPIID := addItemToAPICache(*apiService)
		if externalAPIPrimaryKey, found := apiService.Attributes[apic.AttrExternalAPIPrimaryKey]; found {
			existingAPIs[externalAPIPrimaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}
	}

	if j.refreshAll {
		// Remove items that are not published as Resources
		cacheKeys := agent.apiMap.GetKeys()
		for _, key := range cacheKeys {
			if _, ok := existingAPIs[key]; !ok {
				agent.apiMap.Delete(key)
			}
		}
	}
}

func (j *discoveryCache) updatePIServiceInstancesCache() {
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
		agent.instanceMap.Flush()
	}
	for _, instance := range serviceInstances {
		if _, valid := instance.Attributes[apic.AttrExternalAPIID]; !valid {
			continue // skip instance without external api id
		}
		agent.instanceMap.Set(instance.Metadata.ID, instance)
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

		agent.categoryMap.SetWithSecondaryKey(category.Name, category.Title, category)
		existingCategories[category.Name] = true
	}

	if j.refreshAll {
		// Remove categories that no longer exist
		cacheKeys := agent.categoryMap.GetKeys()
		for _, key := range cacheKeys {
			if _, ok := existingCategories[key]; !ok {
				agent.categoryMap.Delete(key)
			}
		}
	}
}

var updateCacheForExternalAPIPrimaryKey = func(externalAPIPrimaryKey string) (interface{}, error) {
	query := map[string]string{
		apic.QueryKey: attributesQueryParam + apic.AttrExternalAPIPrimaryKey + "==\"" + externalAPIPrimaryKey + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPIID = func(externalAPIID string) (interface{}, error) {
	query := map[string]string{
		apic.QueryKey: attributesQueryParam + apic.AttrExternalAPIID + "==\"" + externalAPIID + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPIName = func(externalAPIName string) (interface{}, error) {
	query := map[string]string{
		apic.QueryKey: attributesQueryParam + apic.AttrExternalAPIName + "==\"" + externalAPIName + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPI = func(query map[string]string) (interface{}, error) {
	apiServerURL := agent.cfg.GetServicesURL()

	response, err := agent.apicClient.ExecuteAPI(coreapi.GET, apiServerURL, query, nil)
	if err != nil {
		return nil, err
	}
	apiService := apiV1.ResourceInstance{}
	json.Unmarshal(response, &apiService)
	addItemToAPICache(apiService)
	return apiService, nil
}

func addItemToAPICache(apiService apiV1.ResourceInstance) string {
	externalAPIID, ok := apiService.Attributes[apic.AttrExternalAPIID]
	if ok {
		externalAPIName := apiService.Attributes[apic.AttrExternalAPIName]
		if externalAPIPrimaryKey, found := apiService.Attributes[apic.AttrExternalAPIPrimaryKey]; found {
			// Verify secondary key and validate if we need to remove it from the apiMap (cache)
			if _, err := agent.apiMap.Get(externalAPIID); err != nil {
				agent.apiMap.Delete(externalAPIID)
			}

			agent.apiMap.SetWithSecondaryKey(externalAPIPrimaryKey, externalAPIID, apiService)
			agent.apiMap.SetSecondaryKey(externalAPIPrimaryKey, externalAPIName)
		} else {
			agent.apiMap.SetWithSecondaryKey(externalAPIID, externalAPIName, apiService)
		}
		log.Tracef("added api name: %s, id %s to API cache", externalAPIName, externalAPIID)
	}
	return externalAPIID
}
