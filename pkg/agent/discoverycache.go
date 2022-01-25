package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize        = 100
	healthcheckEndpoint      = "central"
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
	lastServiceTime  time.Time
	lastInstanceTime time.Time
	lastCategoryTime time.Time
	refreshAll       bool
	getHCStatus      hc.GetStatusLevel
}

func newDiscoveryCache(getAll bool) *discoveryCache {
	return &discoveryCache{
		lastServiceTime:  time.Time{},
		lastInstanceTime: time.Time{},
		lastCategoryTime: time.Time{},
		refreshAll:       getAll,
		getHCStatus:      hc.GetStatus,
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
		j.validateAPIServiceInstances()
		j.updateCategoryCache()
	}
	fetchConfig()
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
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(v1.APIServerTimeFormat))
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

func (j *discoveryCache) validateAPIServiceInstances() {
	if agent.apiValidator == nil {
		return
	}

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastInstanceTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(v1.APIServerTimeFormat))
	}

	j.lastInstanceTime = time.Now()
	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetInstancesURL(), apiServerPageSize)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances"))
		return
	}

	for _, instance := range serviceInstances {
		if j.refreshAll {
			break // no need to do this loop when refreshing the entire cache
		}
		if _, valid := instance.Attributes[apic.AttrExternalAPIID]; !valid {
			continue // skip instance without external api id
		}
		// Update the lastInstanceTime based on the newest instance found
		thisTime := time.Time(instance.Metadata.Audit.CreateTimestamp)
		if j.lastInstanceTime.Before(thisTime) {
			j.lastInstanceTime = thisTime
		}
	}

	// When reloading all api service instances we can just write over the existing cache
	if !j.refreshAll {
		serviceInstances = j.loadServiceInstancesFromCache(serviceInstances)
	}
	serviceInstances = validateAPIOnDataplane(serviceInstances)
	j.saveServiceInstancesToCache(serviceInstances)
}

func (j *discoveryCache) updateCategoryCache() {
	log.Trace("updating category cache")

	// Update cache with published resources
	existingCategories := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastCategoryTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(queryFormatString, apic.CreateTimestampQueryKey, j.lastCategoryTime.Format(v1.APIServerTimeFormat))
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

func (j *discoveryCache) saveServiceInstancesToCache(serviceInstances []*apiV1.ResourceInstance) {
	// Save all the instance names to make sure the map is unique
	instanceNames := make(map[string]struct{})
	for _, instance := range serviceInstances {
		instanceNames[instance.Name] = struct{}{}
	}
	cache.GetCache().Set(serviceInstanceCache, serviceInstances)
	cache.GetCache().Set(serviceInstanceNameCache, instanceNames)
}

func (j *discoveryCache) loadServiceInstancesFromCache(serviceInstances []*apiV1.ResourceInstance) []*apiV1.ResourceInstance {
	cachedInstancesInterface, err := cache.GetCache().Get(serviceInstanceCache)
	if err != nil {
		return serviceInstances
	}
	cachedInstancesNames, err := cache.GetCache().Get(serviceInstanceNameCache)
	if err != nil {
		return serviceInstances
	}
	cachedInstances := cachedInstancesInterface.([]*apiV1.ResourceInstance)
	for _, instance := range serviceInstances {
		// validate that the instance is not already in the array
		if _, found := cachedInstancesNames.(map[string]struct{})[instance.Name]; !found {
			cachedInstances = append(cachedInstances, instance)
		}
	}

	// return the full list
	return cachedInstances
}

func validateAPIOnDataplane(serviceInstances []*apiV1.ResourceInstance) []*apiV1.ResourceInstance {
	cleanServiceInstances := make([]*apiV1.ResourceInstance, 0)
	// Validate the API on dataplane.  If API is not valid, mark the consumer instance as "DELETED"
	for _, serviceInstanceResource := range serviceInstances {
		if _, valid := serviceInstanceResource.Attributes[apic.AttrExternalAPIID]; !valid {
			continue // skip service instances without external api id
		}
		serviceInstance := &v1alpha1.APIServiceInstance{}
		serviceInstance.FromInstance(serviceInstanceResource)
		externalAPIID := serviceInstance.Attributes[apic.AttrExternalAPIID]
		externalAPIStage := serviceInstance.Attributes[apic.AttrExternalAPIStage]
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
			deleteServiceInstanceOrService(serviceInstance, externalAPIID)
		} else {
			cleanServiceInstances = append(cleanServiceInstances, serviceInstanceResource)
		}
	}
	return cleanServiceInstances
}

func shouldDeleteService(apiID string) bool {
	list, err := agent.apicClient.GetConsumerInstancesByExternalAPIID(apiID)
	if err != nil {
		return false
	}

	// if there is only 1 consumer instance left, we can signal to delete the service too
	return len(list) <= 1
}

func deleteServiceInstanceOrService(serviceInstance *v1alpha1.APIServiceInstance, externalAPIID string) {
	if shouldDeleteService(externalAPIID) {
		log.Infof("API no longer exists on the dataplane; deleting the API Service and corresponding catalog item %s", serviceInstance.Title)
		// deleting the service will delete all associated resources, including the consumerInstance
		err := agent.apicClient.DeleteServiceByAPIID(externalAPIID)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingService, err.Error()).FormatError(serviceInstance.Title))
		} else {
			log.Debugf("Deleted API Service for catalog item %s from Amplify Central", serviceInstance.Title)
		}
	} else {
		log.Infof("API no longer exists on the dataplane, deleting the catalog item %s", serviceInstance.Title)
		err := agent.apicClient.DeleteAPIServiceInstance(serviceInstance.Name)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingCatalogItem, err.Error()).FormatError(serviceInstance.Title))
		} else {
			log.Debugf("Deleted catalog item %s from Amplify Central", serviceInstance.Title)
		}
	}
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
