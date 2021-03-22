package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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
	if agent.apicClient != nil {
		ret, err := agent.apicClient.PublishService(serviceBody)
		if err == nil {
			apiSvc, e := ret.AsInstance()
			if e == nil {
				addItemToAPICache(*apiSvc)
				//update the local activity timestamp for the event to compare against
				UpdateLocalActivityTime()
			}
		} else {
			return err
		}
	}
	return nil
}

// RegisterAPIValidator - Registers callback for validating the API on gateway
func RegisterAPIValidator(apiValidator APIValidator) {
	agent.apiValidator = apiValidator
}
