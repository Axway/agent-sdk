package agent

import (
	"encoding/json"
	"fmt"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize    = 20
	healthcheckEndpoint  = "central"
	attributesQueryParam = "attributes."
	apiServerFields      = "name,title,attributes"
)

type discoveryCache struct {
	jobs.Job
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
	updateAPICache()
	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		validateConsumerInstances()
	}
	fetchConfig()
	return nil
}

func updateAPICache() {
	log.Trace("updating API cache")

	// Update cache with published resources
	existingAPIs := make(map[string]bool)
	query := map[string]string{
		"query":  attributesQueryParam + apic.AttrExternalAPIID + "!=\"\"",
		"fields": apiServerFields,
	}

	apiServices, _ := GetCentralClient().GetAPIV1ResourceInstances(query, agent.cfg.GetServicesURL())

	for _, apiService := range apiServices {
		externalAPIID := addItemToAPICache(*apiService)
		if externalAPIPrimaryKey, found := apiService.Attributes[apic.AttrExternalAPIPrimaryKey]; found {
			existingAPIs[externalAPIPrimaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}

	}

	// Remove items that are not published as Resources
	cacheKeys := agent.apiMap.GetKeys()
	for _, key := range cacheKeys {
		if _, ok := existingAPIs[key]; !ok {
			agent.apiMap.Delete(key)
		}
	}
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

func validateConsumerInstances() {
	if agent.apiValidator == nil {
		return
	}

	query := map[string]string{
		"query":  attributesQueryParam + apic.AttrExternalAPIID + "!=\"\"",
		"fields": apiServerFields,
	}

	consumerInstances, err := GetCentralClient().GetAPIV1ResourceInstances(query, agent.cfg.GetConsumerInstancesURL())
	if err != nil {
		log.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("ConsumerInstance"))
		return
	}
	validateAPIOnDataplane(consumerInstances)
}

func validateAPIOnDataplane(consumerInstances []*apiV1.ResourceInstance) {
	// Validate the API on dataplane.  If API is not valid, mark the consumer instance as "DELETED"
	for _, consumerInstanceResource := range consumerInstances {
		consumerInstance := &v1alpha1.ConsumerInstance{}
		consumerInstance.FromInstance(consumerInstanceResource)
		externalAPIID := consumerInstance.Attributes[apic.AttrExternalAPIID]
		externalAPIStage := consumerInstance.Attributes[apic.AttrExternalAPIStage]
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
			deleteConsumerInstanceOrService(consumerInstance, externalAPIID, externalAPIStage)
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

func deleteConsumerInstanceOrService(consumerInstance *v1alpha1.ConsumerInstance, externalAPIID, externalAPIStage string) {
	if shouldDeleteService(externalAPIID, externalAPIStage) {
		log.Infof("API no longer exists on the dataplane; deleting the API Service and corresponding catalog item %s from Amplify Central", consumerInstance.Title)
		// deleting the service will delete all associated resources, including the consumerInstance
		err := agent.apicClient.DeleteServiceByAPIID(externalAPIID)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingService, err.Error()).FormatError(consumerInstance.Title))
		} else {
			log.Infof("Deleted API Service for catalog item %s from Amplify Central", consumerInstance.Title)
		}
	} else {
		log.Infof("API no longer exists on the dataplane, deleting the catalog item %s from Amplify Central", consumerInstance.Title)
		err := agent.apicClient.DeleteConsumerInstance(consumerInstance.Name)
		if err != nil {
			log.Error(utilErrors.Wrap(ErrDeletingCatalogItem, err.Error()).FormatError(consumerInstance.Title))
		} else {
			log.Infof("Deleted catalog item %s from Amplify Central", consumerInstance.Title)
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
