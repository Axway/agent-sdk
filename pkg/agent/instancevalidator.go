package agent

import (
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type instanceValidator struct {
	jobs.Job
	cacheLock       *sync.Mutex
	isAgentPollMode bool
}

func newInstanceValidator(cacheLock *sync.Mutex, isAgentPollMode bool) *instanceValidator {
	return &instanceValidator{cacheLock: cacheLock, isAgentPollMode: isAgentPollMode}
}

//Ready -
func (j *instanceValidator) Ready() bool {
	return true
}

//Status -
func (j *instanceValidator) Status() error {
	return nil
}

//Execute -
func (j *instanceValidator) Execute() error {
	j.validateAPIOnDataplane()
	return nil
}

func (j *instanceValidator) validateAPIOnDataplane() {
	j.cacheLock.Lock()
	defer j.cacheLock.Unlock()

	log.Info("validating api service instance on dataplane")
	// Validate the API on dataplane.  If API is not valid, mark the consumer instance as "DELETED"
	for _, key := range agent.cacheManager.GetAPIServiceInstanceKeys() {
		serviceInstanceResource, err := agent.cacheManager.GetAPIServiceInstanceByID(key)
		if err != nil {
			continue
		}

		if _, valid := serviceInstanceResource.Attributes[definitions.AttrExternalAPIID]; !valid {
			continue // skip service instances without external api id
		}
		externalAPIID := serviceInstanceResource.Attributes[definitions.AttrExternalAPIID]
		externalAPIStage := serviceInstanceResource.Attributes[definitions.AttrExternalAPIStage]
		externalPrimaryKey := serviceInstanceResource.Attributes[definitions.AttrExternalAPIPrimaryKey]
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
			j.deleteServiceInstanceOrService(serviceInstanceResource, externalPrimaryKey, externalAPIID, externalAPIStage)
		}
	}
}

func (j *instanceValidator) shouldDeleteService(externalAPIPrimaryKey, externalAPIID, externalAPIStage string) bool {
	instanceCount := 0
	if externalAPIPrimaryKey != "" {
		instanceCount = j.getServiceInstanceCount(definitions.AttrExternalAPIPrimaryKey, externalAPIPrimaryKey)
		log.Tracef("Query instances with externalPrimaryKey attribute : %s", externalAPIPrimaryKey)
	} else {
		instanceCount = j.getServiceInstanceCount(definitions.AttrExternalAPIID, externalAPIID)
		log.Tracef("Query instances with externalAPIID attribute :", externalAPIID)
	}

	log.Tracef("Instances count : %d", instanceCount)

	return instanceCount <= 1
}

func (j *instanceValidator) getServiceInstanceCount(attName, attValue string) int {
	instanceCount := 0
	for _, key := range agent.cacheManager.GetAPIServiceInstanceKeys() {
		serviceInstanceResource, _ := agent.cacheManager.GetAPIServiceInstanceByID(key)
		if serviceInstanceResource != nil {
			instaceAttValue := serviceInstanceResource.Attributes[attName]
			if attValue == instaceAttValue {
				instanceCount++
			}
		}
	}
	return instanceCount
}

func (j *instanceValidator) deleteServiceInstanceOrService(resource *apiV1.ResourceInstance, externalAPIPrimaryKey, externalAPIID, externalAPIStage string) {
	msg := ""
	var err error
	var agentError *utilErrors.AgentError

	defer func() {
		// remove the api service instance from the cache for both scenarios
		if j.isAgentPollMode {
			// In GRPC mode delete is done on receiving delete event from service
			agent.cacheManager.DeleteAPIServiceInstance(resource.Metadata.ID)
		}
	}()

	// delete if it is an api service
	if j.shouldDeleteService(externalAPIPrimaryKey, externalAPIID, externalAPIStage) {
		log.Infof("API no longer exists on the dataplane; deleting the API Service and corresponding catalog item %s", resource.Title)
		agentError = ErrDeletingService
		msg = "Deleted API Service for catalog item %s from Amplify Central"

		svc := agent.cacheManager.GetAPIServiceWithAPIID(externalAPIID)
		if svc == nil {
			log.Errorf("api service %s not found in cache. unable to delete it from central", externalAPIID)
			return
		}

		// deleting the service will delete all associated resources, including the consumerInstance
		err = agent.apicClient.DeleteServiceByName(svc.Name)
		if j.isAgentPollMode {
			agent.cacheManager.DeleteAPIService(externalAPIID)
		}
	} else {
		// delete if it is an api service instance
		log.Infof("API no longer exists on the dataplane, deleting the catalog item %s", resource.Title)
		agentError = ErrDeletingCatalogItem
		msg = "Deleted catalog item %s from Amplify Central"

		err = agent.apicClient.DeleteAPIServiceInstance(resource.Name)
	}

	if err != nil {
		log.Error(utilErrors.Wrap(agentError, err.Error()).FormatError(resource.Title))
		return
	}

	log.Debugf(msg, resource.Title)
}
