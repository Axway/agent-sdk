package agent

import (
	"encoding/json"
	"fmt"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize    = 100
	healthcheckEndpoint  = "central"
	attributesQueryParam = "attributes."
	apiServerFields      = "name,title,attributes"
)

type discoveryCache struct {
	jobs.Job
	lastServicesTime  time.Time
	lastInstancesTime time.Time
	getAll            bool
}

func newDiscoveryCache(getAll bool) *discoveryCache {
	return &discoveryCache{
		lastServicesTime:  time.Time{},
		lastInstancesTime: time.Time{},
		getAll:            getAll,
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
	log.Trace("executing API cache update job")
	j.updateAPICache()
	if agent.cfg.GetAgentType() == config.DiscoveryAgent && j.getAll {
		j.validateAPIServiceInstances()
	}
	fetchConfig()
	return nil
}

func (j *discoveryCache) updateAPICache() {
	log.Trace("updating API cache")

	// Update cache with published resources
	existingAPIs := make(map[string]bool)
	query := map[string]string{
		"fields": apiServerFields,
	}

	if !j.lastServicesTime.IsZero() && !j.getAll {
		query["query"] = fmt.Sprintf("metadata.audit.createTimestamp>\"%s\"", j.lastServicesTime.Format(v1.APIServerTimeFormat))
	}

	j.lastServicesTime = time.Now()
	apiServices, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetServicesURL(), apiServerPageSize)

	for _, apiService := range apiServices {
		if _, valid := apiService.Attributes[apic.AttrExternalAPIID]; !valid {
			continue // skip service without external api id
		}
		externalAPIID := addItemToAPICache(*apiService)
		if externalAPIPrimaryKey, found := apiService.Attributes[apic.AttrExternalAPIPrimaryKey]; found {
			existingAPIs[externalAPIPrimaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}
	}

	if j.getAll {
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
		"fields": apiServerFields,
	}

	if !j.lastInstancesTime.IsZero() && !j.getAll {
		query["query"] = fmt.Sprintf("metadata.audit.createTimestamp>%d", j.lastInstancesTime.UnixNano())
	}

	j.lastInstancesTime = time.Now()
	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, agent.cfg.GetInstancesURL(), apiServerPageSize)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances"))
		return
	}
	validateAPIOnDataplane(serviceInstances)
}

var updateCacheForExternalAPIPrimaryKey = func(externalAPIPrimaryKey string) (interface{}, error) {
	query := map[string]string{
		"query": attributesQueryParam + apic.AttrExternalAPIPrimaryKey + "==\"" + externalAPIPrimaryKey + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPIID = func(externalAPIID string) (interface{}, error) {
	query := map[string]string{
		"query": attributesQueryParam + apic.AttrExternalAPIID + "==\"" + externalAPIID + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPIName = func(externalAPIName string) (interface{}, error) {
	query := map[string]string{
		"query": attributesQueryParam + apic.AttrExternalAPIName + "==\"" + externalAPIName + "\"",
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

func validateAPIOnDataplane(serviceInstances []*apiV1.ResourceInstance) {
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
			deleteServiceInstanceOrService(serviceInstance, externalAPIID, externalAPIStage)
		}
	}
}

func shouldDeleteService(apiID, stage string) bool {
	// no agent-specific validator means to delete the service
	if agent.deleteServiceValidator == nil {
		return true
	}
	// let the agent decide if service should be deleted
	return agent.deleteServiceValidator(apiID, stage)
}

func deleteServiceInstanceOrService(consumerInstance *v1alpha1.APIServiceInstance, externalAPIID, externalAPIStage string) {
	if shouldDeleteService(externalAPIID, externalAPIStage) {
		log.Infof("API no longer exists on the dataplane; deleting the API Service and corresponding catalog item %s", consumerInstance.Title)
		// deleting the service will delete all associated resources, including the consumerInstance
		err := agent.apicClient.DeleteServiceByAPIID(externalAPIID)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingService, err.Error()).FormatError(consumerInstance.Title))
		} else {
			log.Debugf("Deleted API Service for catalog item %s from Amplify Central", consumerInstance.Title)
		}
	} else {
		log.Infof("API no longer exists on the dataplane, deleting the catalog item %s", consumerInstance.Title)
		err := agent.apicClient.DeleteConsumerInstance(consumerInstance.Name)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingCatalogItem, err.Error()).FormatError(consumerInstance.Title))
		} else {
			log.Debugf("Deleted catalog item %s from Amplify Central", consumerInstance.Title)
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
