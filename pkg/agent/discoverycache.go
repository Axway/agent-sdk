package agent

import (
	"encoding/json"
	"fmt"
	"strconv"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type discoveryCache struct {
	jobs.Job
}

//Ready -
func (j *discoveryCache) Ready() bool {
	err := j.Status()
	if err != nil {
		return false
	}
	return true
}

//Status -
func (j *discoveryCache) Status() error {
	status := agent.apicClient.Healthcheck("Cache")
	if status.Result == hc.OK {
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
	const pageSize = 20
	apiServerURL := agent.cfg.GetServicesURL()
	page := 1

	// Update cache with published resources
	existingAPIs := make(map[string]bool)

	morePages := true
	for morePages {
		query := map[string]string{
			"query":    "attributes." + apic.AttrExternalAPIID + "!=\"\"",
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(pageSize),
		}

		response, err := agent.apicClient.ExecuteAPI(coreapi.GET, apiServerURL, query, nil)
		if err != nil {
			return
		}
		apiServices := make([]apiV1.ResourceInstance, 0)
		json.Unmarshal(response, &apiServices)

		log.Tracef("found the following API services: %+v", apiServices)
		for _, apiService := range apiServices {
			externalAPIID := addItemToAPICache(apiService)
			existingAPIs[externalAPIID] = true
		}

		if len(apiServices) < pageSize {
			morePages = false
		}
		page++
	}

	// Remove items that are not published as Resources
	cacheKeys := agent.apiMap.GetKeys()
	for _, key := range cacheKeys {
		if _, ok := existingAPIs[key]; !ok {
			agent.apiMap.Delete(key)
		}
	}
}

var updateCacheForExternalAPIID = func(externalAPIID string) (interface{}, error) {
	query := map[string]string{
		"query": "attributes." + apic.AttrExternalAPIID + "==\"" + externalAPIID + "\"",
	}

	return updateCacheForExternalAPI(query)
}

var updateCacheForExternalAPIName = func(externalAPIName string) (interface{}, error) {
	query := map[string]string{
		"query": "attributes." + apic.AttrExternalAPIName + "==\"" + externalAPIName + "\"",
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

	consumerInstancesURL := agent.cfg.GetConsumerInstancesURL()
	query := map[string]string{
		"query": "attributes." + apic.AttrExternalAPIID + "!=\"\"",
	}

	response, err := agent.apicClient.ExecuteAPI(coreapi.GET, consumerInstancesURL, query, nil)
	if err != nil {
		return
	}
	consumerInstances := make([]apiV1.ResourceInstance, 0)
	json.Unmarshal(response, &consumerInstances)

	// Validate the API on dataplane, if not valid mark the consumer instance as "DELETED"
	for _, consumerInstanceResource := range consumerInstances {
		consumerInstance := &v1alpha1.ConsumerInstance{}
		consumerInstance.FromInstance(&consumerInstanceResource)
		externalAPIID := consumerInstance.Attributes[apic.AttrExternalAPIID]
		externalAPIStage := consumerInstance.Attributes[apic.AttrExternalAPIStage]
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
			log.Infof("API deleted from dataplane, deleting the catalog item %s from AMPLIFY Central", consumerInstance.Title)
			err = agent.apicClient.DeleteConsumerInstance(consumerInstance.Name)
			if err != nil {
				log.Errorf("Unable to delete catalog item %s from AMPLIFY Central, %s", consumerInstance.Title, err.Error())
			} else {
				log.Infof("Deleted catalog item %s from AMPLIFY Central", consumerInstance.Title)
			}
		}
	}
}

func addItemToAPICache(apiService apiV1.ResourceInstance) string {
	externalAPIID, ok := apiService.Attributes[apic.AttrExternalAPIID]
	if ok {
		externalAPIName := apiService.Attributes[apic.AttrExternalAPIName]
		agent.apiMap.SetWithSecondaryKey(externalAPIID, externalAPIName, apiService)
		log.Tracef("added api name: %s, id %s to API cache", externalAPIName, externalAPIID)
	}
	return externalAPIID
}
