package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// PublishAPIFunc definition for the PublishAPI func
type PublishAPIFunc func(serviceBody apic.ServiceBody) error

// getAPIByPrimaryKey - finds the api by the Primary Key from cache or API Server query
func getAPIByPrimaryKey(primaryKey string) *apiV1.ResourceInstance {
	var api *apiV1.ResourceInstance
	if agent.cacheManager != nil {
		api = agent.cacheManager.GetAPIServiceWithPrimaryKey(primaryKey)
	}
	return api
}

// getAPIByID - finds the api by the ID from cache or API Server query
func getAPIByID(externalAPIID string) *apiV1.ResourceInstance {
	var api *apiV1.ResourceInstance
	if agent.cacheManager != nil {
		api = agent.cacheManager.GetAPIServiceWithAPIID(externalAPIID)
	}
	return api
}

// getAPIByName - finds the api by the Name from cache or API Server query
func getAPIByName(apiName string) *apiV1.ResourceInstance {
	var api *apiV1.ResourceInstance
	if agent.cacheManager != nil {
		api = agent.cacheManager.GetAPIServiceWithName(apiName)
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
	return getAttributeFromResource(api, attrName)
}

// GetAttributeOnPublishedAPI - DEPRECATED Returns the value on published proxy
func GetAttributeOnPublishedAPI(externalAPIID string, attrName string) string {
	// DEPRECATED
	log.DeprecationWarningReplace("GetAttributeOnPublishedAPI", "GetAttributeOnPublishedAPIByID")
	return GetAttributeOnPublishedAPIByID(externalAPIID, attrName)
}

func getAttributeFromResource(resource *apiV1.ResourceInstance, attrName string) string {
	if resource == nil {
		return ""
	}
	v, _ := util.GetAgentDetailsValue(resource, attrName)
	return v
}

// GetAttributeOnPublishedAPIByID - Returns the value on published proxy
func GetAttributeOnPublishedAPIByID(externalAPIID string, attrName string) string {
	api := getAPIByID(externalAPIID)
	return getAttributeFromResource(api, attrName)
}

// GetAttributeOnPublishedAPIByPrimaryKey - Returns the value on published proxy
func GetAttributeOnPublishedAPIByPrimaryKey(primaryKey string, attrName string) string {
	api := getAPIByPrimaryKey(primaryKey)
	return getAttributeFromResource(api, attrName)
}

// GetOwnerOnPublishedAPIByName - Returns the owner spec of the published proxy
func GetOwnerOnPublishedAPIByName(apiName string) *apiV1.Owner {
	api := getAPIByName(apiName)
	if api == nil {
		return nil
	}
	return api.Owner
}

// GetOwnerOnPublishedAPIByID - Returns the owner spec of the published proxy
func GetOwnerOnPublishedAPIByID(externalAPIID string) *apiV1.Owner {
	api := getAPIByID(externalAPIID)
	if api == nil {
		return nil
	}
	return api.Owner
}

// GetOwnerOnPublishedAPIByPrimaryKey - Returns the owner spec of the published proxy
func GetOwnerOnPublishedAPIByPrimaryKey(primaryKey string) *apiV1.Owner {
	api := getAPIByPrimaryKey(primaryKey)
	if api == nil {
		return nil
	}
	return api.Owner
}

func PublishingLock() {
	agent.publishingLock.Lock()
}

func PublishingUnlock() {
	agent.publishingLock.Unlock()
}

// PublishAPI - Publishes the API
func PublishAPI(serviceBody apic.ServiceBody) error {
	if agent.apicClient != nil {

		var err error
		_, err = publishAccessRequestDefinition(&serviceBody)
		if err != nil {
			return err
		}
	}

	_, err := agent.apicClient.PublishService(&serviceBody)
	if err != nil {
		return err
	}
	log.Infof("Published API %v-%v in environment %v", serviceBody.APIName, serviceBody.Version, agent.cfg.GetEnvironmentName())

	return nil
}

// RemovePublishedAPIAgentDetail -
func RemovePublishedAPIAgentDetail(externalAPIID, detailKey string) error {
	apiSvc := agent.cacheManager.GetAPIServiceWithAPIID(externalAPIID)

	details := util.GetAgentDetails(apiSvc)
	if _, ok := details[detailKey]; !ok {
		return nil
	}

	delete(details, detailKey)

	util.SetAgentDetails(apiSvc, details)

	err := agent.apicClient.CreateSubResource(apiSvc.ResourceMeta, map[string]interface{}{definitions.XAgentDetails: details})
	if err != nil {
		return err
	}

	err = agent.cacheManager.AddAPIService(apiSvc)
	return err

}

func publishAccessRequestDefinition(serviceBody *apic.ServiceBody) (*apiV1.ResourceInstance, error) {
	agent.ardLock.Lock()
	defer agent.ardLock.Unlock()

	if serviceBody.GetAccessRequestDefinition() != nil {
		newARD, err := createOrUpdateAccessRequestDefinition(serviceBody.GetAccessRequestDefinition())
		if err == nil && newARD != nil {
			serviceBody.SetAccessRequestDefinitionName(newARD.Name, true)

			ard, err := newARD.AsInstance()
			if err == nil {
				agent.cacheManager.AddAccessRequestDefinition(ard)
			}
			return ard, err
		}
		return nil, err
	}
	return nil, nil
}

func getAPIValidator() APIValidator {
	agent.apiValidatorLock.Lock()
	defer agent.apiValidatorLock.Unlock()

	return agent.apiValidator
}

func setAPIValidator(apiValidator APIValidator) {
	agent.apiValidatorLock.Lock()
	defer agent.apiValidatorLock.Unlock()

	agent.apiValidator = apiValidator
}

// RegisterAPIValidator - Registers callback for validating the API on gateway
func RegisterAPIValidator(apiValidator APIValidator) {
	setAPIValidator(apiValidator)
}

// RegisterDeleteServiceValidator - DEPRECATED Registers callback for validating if the service should be deleted
func RegisterDeleteServiceValidator(validator interface{}) {
	log.Warnf("the RegisterDeleteServiceValidator is no longer used, please remove the call to it")
}
