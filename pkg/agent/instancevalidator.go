package agent

import (
	"sync"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util/errors"
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
	for _, key := range agent.instanceMap.GetKeys() {
		value, err := agent.instanceMap.Get(key)
		if err != nil {
			continue
		}
		if serviceInstanceResource, ok := value.(*apiV1.ResourceInstance); ok {
			if _, valid := serviceInstanceResource.Attributes[apic.AttrExternalAPIID]; !valid {
				continue // skip service instances without external api id
			}
			externalAPIID := serviceInstanceResource.Attributes[apic.AttrExternalAPIID]
			externalAPIStage := serviceInstanceResource.Attributes[apic.AttrExternalAPIStage]
			// Check if the consumer instance was published by agent, i.e. following attributes are set
			// - externalAPIID should not be empty
			// - externalAPIStage could be empty for dataplanes that do not support it
			if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
				j.deleteServiceInstanceOrService(serviceInstanceResource, externalAPIID, externalAPIStage)
			}

		}
	}
}

func (j *instanceValidator) shouldDeleteService(apiID, stage string) bool {
	list, err := agent.apicClient.GetConsumerInstancesByExternalAPIID(apiID)
	if err != nil {
		return false
	}

	// if there is only 1 consumer instance left, we can signal to delete the service too
	return len(list) <= 1
}

func (j *instanceValidator) deleteServiceInstanceOrService(serviceInstance *apiV1.ResourceInstance, externalAPIID, externalAPIStage string) {
	msg := ""
	var err error
	var agentError *errors.AgentError
	if j.shouldDeleteService(externalAPIID, externalAPIStage) {
		log.Infof("API no longer exists on the dataplane; deleting the API Service and corresponding catalog item %s", serviceInstance.Title)
		agentError = ErrDeletingService
		msg = "Deleted API Service for catalog item %s from Amplify Central"

		// deleting the service will delete all associated resources, including the consumerInstance
		err = agent.apicClient.DeleteServiceByAPIID(externalAPIID)
		// Todo clean up other cached apiserviceinstances related to apiservice
		if j.isAgentPollMode {
			err := agent.apiMap.Delete(externalAPIID)
			if err != nil {
				agent.apiMap.DeleteBySecondaryKey(externalAPIID)
			}
		}
	} else {
		log.Infof("API no longer exists on the dataplane, deleting the catalog item %s", serviceInstance.Title)
		agentError = ErrDeletingCatalogItem
		msg = "Deleted catalog item %s from Amplify Central"

		err = agent.apicClient.DeleteAPIServiceInstance(serviceInstance.Name)
	}

	if err != nil {
		log.Error(utilErrors.Wrap(agentError, err.Error()).FormatError(serviceInstance.Title))
		return
	}
	if j.isAgentPollMode {
		// In GRPC mode the delete is done on receiving delete event from serice
		agent.instanceMap.Delete(serviceInstance.Metadata.ID)
	}
	log.Debugf(msg, serviceInstance.Title)
}
