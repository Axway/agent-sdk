package agent

import (
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	apiV1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

// IsAPIPublished  - Returns true if the API Service is already published
func IsAPIPublished(externalAPIID string) bool {
	if agent.apiMap != nil {
		api, _ := agent.apiMap.Get(externalAPIID)
		return api != nil
	}
	return false
}

// GetAttributeOnPublishedAPIByName - Returns the value on published proxy
func GetAttributeOnPublishedAPIByName(apiName string, attrName string) string {
	if agent.apiMap != nil {
		api, _ := agent.apiMap.GetBySecondaryKey(apiName)
		if api != nil {
			apiSvc := api.(apiV1.ResourceInstance)
			attrVal := apiSvc.ResourceMeta.Attributes[attrName]
			return attrVal
		}
	}
	return ""
}

// GetAttributeOnPublishedAPI - Returns the value on published proxy
func GetAttributeOnPublishedAPI(externalAPIID string, attrName string) string {
	if agent.apiMap != nil {
		api, _ := agent.apiMap.Get(externalAPIID)
		if api != nil {
			apiSvc := api.(apiV1.ResourceInstance)
			attrVal := apiSvc.ResourceMeta.Attributes[attrName]
			return attrVal
		}
	}
	return ""
}

// PublishAPI - Publishes the API
func PublishAPI(serviceBody apic.ServiceBody) error {
	var err error
	if agent.apicClient != nil {
		ret, err := agent.apicClient.PublishService(serviceBody)
		if err == nil {
			apiSvc, e := ret.AsInstance()
			if e == nil {
				addItemToAPICache(*apiSvc)
			}
		}
	}
	return err
}
