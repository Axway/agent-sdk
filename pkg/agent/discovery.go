package agent

import (
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/jobs"
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

// PublishAPI - Publishes the API
func PublishAPI(serviceBody apic.ServiceBody) error {
	if agent.apicClient != nil {
		var accReqDef *apiV1.ResourceInstance
		if agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
			var err error
			accReqDef, err = publishAccessRequestDefinition(&serviceBody)
			if err != nil {
				return err
			}
		}

		ret, err := agent.apicClient.PublishService(&serviceBody)
		if err == nil {
			log.Infof("Published API %v-%v in environment %v", serviceBody.APIName, serviceBody.Version, agent.cfg.GetEnvironmentName())
			// when in grpc mode cache updates happen when events are received. Only update the cache here for poll mode.
			apiSvc, e := ret.AsInstance()
			if e == nil {
				addErr := agent.cacheManager.AddAPIService(apiSvc)
				if addErr != nil {
					log.Error(addErr)
				}
			}
		} else {
			if accReqDef != nil {
				// rollback the access request definition if an error was hit publishing the linked service
				agent.apicClient.DeleteResourceInstance(accReqDef)
			}
			return err
		}
	}
	return nil
}

func publishAccessRequestDefinition(serviceBody *apic.ServiceBody) (*apiV1.ResourceInstance, error) {
	if serviceBody.GetAccessRequestDefintion() != nil {
		newARD, err := createOrUpdateAccessRequestDefinition(serviceBody.GetAccessRequestDefintion())
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

// RegisterAPIValidator - Registers callback for validating the API on gateway
func RegisterAPIValidator(apiValidator APIValidator) {
	agent.apiValidator = apiValidator

	if agent.instanceValidatorJobID == "" && apiValidator != nil {
		validator := newInstanceValidator()
		jobID, err := jobs.RegisterIntervalJobWithName(validator, agent.cfg.GetPollInterval(), "API service instance validator")
		agent.instanceValidatorJobID = jobID
		if err != nil {
			log.Error(err)
		}
	}
}

// RegisterDeleteServiceValidator - DEPRECATED Registers callback for validating if the service should be deleted
func RegisterDeleteServiceValidator(validator interface{}) {
	log.Warnf("the RegisterDeleteServiceValidator is no longer used, please remove the call to it")
}
