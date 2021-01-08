package agent

import (
	"encoding/json"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

func updateAPICache() {
	apiServerURL := agent.cfg.GetServicesURL()
	query := map[string]string{
		"query": "attributes." + apic.AttrExternalAPIID + "!=\"\"",
	}

	response, err := agent.apicClient.ExecuteAPI(coreapi.GET, apiServerURL, query, nil)
	if err != nil {
		return
	}
	apiServices := make([]apiV1.ResourceInstance, 0)
	json.Unmarshal(response, &apiServices)

	// Update cache with published resources
	existingAPIs := make(map[string]bool)
	for _, apiService := range apiServices {
		externalAPIID := addItemToAPICache(apiService)
		existingAPIs[externalAPIID] = true
	}

	// Remove items that are not published as Resources
	cacheKeys := agent.apiMap.GetKeys()
	for _, key := range cacheKeys {
		if _, ok := existingAPIs[key]; !ok {
			agent.apiMap.Delete(key)
		}
	}
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
		if externalAPIID != "" {
			if !agent.apiValidator(externalAPIID, externalAPIStage) {
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
}

func addItemToAPICache(apiService apiV1.ResourceInstance) string {
	externalAPIID, ok := apiService.Attributes[apic.AttrExternalAPIID]
	if ok {
		externalAPIName := apiService.Attributes[apic.AttrExternalAPIName]
		agent.apiMap.SetWithSecondaryKey(externalAPIID, externalAPIName, apiService)
	}
	return externalAPIID
}
