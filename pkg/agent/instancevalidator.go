package agent

import (
	"sync"

	"github.com/Axway/agent-sdk/pkg/util"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type instanceValidator struct {
	jobs.Job
	cacheLock *sync.Mutex
}

func newInstanceValidator() *instanceValidator {
	return &instanceValidator{cacheLock: &sync.Mutex{}}
}

// Ready -
func (j *instanceValidator) Ready() bool {
	status := hc.GetStatus(util.CentralHealthCheckEndpoint)
	return status == hc.OK
}

// Status -
func (j *instanceValidator) Status() error {
	return nil
}

// Execute -
func (j *instanceValidator) Execute() error {
	agent.publishingGroup.Wait()
	agent.validatingGroup.Add(1)
	defer agent.validatingGroup.Done()
	j.validateAPIOnDataplane()
	return nil
}

func (j *instanceValidator) validateAPIOnDataplane() {
	j.cacheLock.Lock()
	defer j.cacheLock.Unlock()

	log.Debug("validating api service instances on dataplane")
	// Validate the API on dataplane.  If API is not valid, mark the consumer instance as "DELETED"
	for _, key := range agent.cacheManager.GetAPIServiceInstanceKeys() {
		instance, err := agent.cacheManager.GetAPIServiceInstanceByID(key)
		if err != nil {
			continue
		}

		externalAPIID, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if externalAPIID == "" {
			continue // skip service instances without external api id
		}
		externalAPIStage, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
		externalPrimaryKey, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIPrimaryKey)
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		if externalAPIID != "" && !agent.apiValidator(externalAPIID, externalAPIStage) {
			j.deleteServiceInstance(instance, externalPrimaryKey, externalAPIID)
		}
	}

	log.Debug("validating api services have at least one instance on dataplane")
	for _, key := range agent.cacheManager.GetAPIServiceKeys() {
		service := agent.cacheManager.GetAPIServiceWithPrimaryKey(key)
		if service == nil {
			continue
		}

		if agent.cacheManager.GetAPIServiceInstanceCount(service.Name) == 0 {
			j.deleteService(service)
		}
	}
}

func (j *instanceValidator) deleteServiceInstance(ri *apiV1.ResourceInstance, primaryKey, apiID string) {
	// delete if it is an api service instance
	log.Infof("API no longer exists on the dataplane, deleting the API Service Instance %s", ri.Title)

	err := agent.apicClient.DeleteAPIServiceInstance(ri.Name)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrDeletingServiceInstanceItem, err.Error()).FormatError(ri.Title))
		return
	}
	agent.cacheManager.DeleteAPIServiceInstance(ri.Metadata.ID)

	log.Debugf("Deleted API Service Instance item %s from Amplify Central", ri.Title)
}

func (j *instanceValidator) deleteService(ri *apiV1.ResourceInstance) {
	log.Infof("API Service no longer has a service instance; deleting the API Service %s", ri.Title)

	// deleting the service will delete all associated resources, including the consumerInstance
	err := agent.apicClient.DeleteServiceByName(ri.Name)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrDeletingService, err.Error()).FormatError(ri.Title))
		return
	}
	agent.cacheManager.DeleteAPIService(ri.Name)

	log.Debugf("Deleted API Service %s from Amplify Central", ri.Title)
}
