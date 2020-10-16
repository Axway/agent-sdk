package agent

import (
	"encoding/json"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	apiV1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
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

func addItemToAPICache(apiService apiV1.ResourceInstance) string {
	externalAPIID, ok := apiService.Attributes[apic.AttrExternalAPIID]
	if ok {
		externalAPIName := apiService.Attributes[apic.AttrExternalAPIName]
		agent.apiMap.SetWithSecondaryKey(externalAPIID, externalAPIName, apiService)
	}
	return externalAPIID
}
