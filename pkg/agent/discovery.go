package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// PublishAPIFunc definition for the PublishAPI func
type PublishAPIFunc func(serviceBody apic.ServiceBody) error

// getAPIByPrimaryKey - finds the api by the Primary Key from cache or API Server query
func getAPIByPrimaryKey(primaryKey string) interface{} {
	var api interface{}
	if agent.apiMap != nil {
		api, _ = agent.apiMap.Get(primaryKey)
	}
	return api
}

// getAPIByID - finds the api by the ID from cache or API Server query
func getAPIByID(externalAPIID string) interface{} {
	var api interface{}
	if agent.apiMap != nil {
		api, _ = agent.apiMap.Get(externalAPIID)
		if api == nil {
			api, _ = agent.apiMap.GetBySecondaryKey(externalAPIID) // try to get the API by a secondary key
		}
	}
	return api
}

// getAPIByName - finds the api by the Name from cache or API Server query
func getAPIByName(apiName string) interface{} {
	var api interface{}
	if agent.apiMap != nil {
		api, _ = agent.apiMap.GetBySecondaryKey(apiName)
	}
	return api
}

// IsAPIPublished  - DEPRECATED Returns true if the API Service is already published
func IsAPIPublished(externalAPIID string) bool {
	// DEPRECATED
	log.DeprecationWarningReplace("IsAPIPublished", "IsAPIPublishedByID")
	return IsAPIPublishedByID(externalAPIID)
}

// IsAPIPublishedByID  - Returns true if the API Service is already published
func IsAPIPublishedByID(externalAPIID string) bool {
	return getAPIByID(externalAPIID) != nil
}

// IsAPIPublishedByPrimaryKey  - Returns true if the API Service is already published
func IsAPIPublishedByPrimaryKey(primaryKey string) bool {
	return getAPIByPrimaryKey(primaryKey) != nil
}

// GetAttributeOnPublishedAPIByName - Returns the value on published proxy
func GetAttributeOnPublishedAPIByName(apiName string, attrName string) string {
	api := getAPIByName(apiName)
	if api != nil {
		apiSvc := api.(apiV1.ResourceInstance)
		attrVal := apiSvc.ResourceMeta.Attributes[attrName]
		return attrVal
	}
	return ""
}

// GetAttributeOnPublishedAPI - DEPRECATED Returns the value on published proxy
func GetAttributeOnPublishedAPI(externalAPIID string, attrName string) string {
	// DEPRECATED
	log.DeprecationWarningReplace("GetAttributeOnPublishedAPI", "GetAttributeOnPublishedAPIByID")
	return GetAttributeOnPublishedAPIByID(externalAPIID, attrName)
}

// GetAttributeOnPublishedAPIByID - Returns the value on published proxy
func GetAttributeOnPublishedAPIByID(externalAPIID string, attrName string) string {
	api := getAPIByID(externalAPIID)
	if api != nil {
		apiSvc := api.(apiV1.ResourceInstance)
		attrVal := apiSvc.ResourceMeta.Attributes[attrName]
		return attrVal
	}
	return ""
}

// GetAttributeOnPublishedAPIByPrimaryKey - Returns the value on published proxy
func GetAttributeOnPublishedAPIByPrimaryKey(primaryKey string, attrName string) string {
	api := getAPIByPrimaryKey(primaryKey)
	if api != nil {
		apiSvc := api.(apiV1.ResourceInstance)
		attrVal := apiSvc.ResourceMeta.Attributes[attrName]
		return attrVal
	}
	return ""
}

// PublishAPI - Publishes the API
func PublishAPI(serviceBody apic.ServiceBody) error {
	if agent.apicClient != nil {
		ret, err := agent.apicClient.PublishService(&serviceBody)
		if err == nil {
			log.Infof("Published API %v-%v in environment %v", serviceBody.APIName, serviceBody.Version, agent.cfg.GetEnvironmentName())
			apiSvc, e := ret.AsInstance()
			if e == nil {
				addItemToAPICache(*apiSvc)
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

// RegisterDeleteServiceValidator - DEPRECATED Registers callback for validating if the service should be deleted
func RegisterDeleteServiceValidator(validator interface{}) {
	log.Warnf("the RegisterDeleteServiceValidator is no longer used, please remove the call to it")
}
