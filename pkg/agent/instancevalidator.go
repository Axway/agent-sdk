package agent

import (
	"fmt"
	"sync"
	"time"

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
	logger    log.FieldLogger
	cacheLock *sync.Mutex
}

func newInstanceValidator() *instanceValidator {
	return &instanceValidator{
		logger:    logger.WithComponent("instanceValidator"),
		cacheLock: &sync.Mutex{},
	}
}

// Ready -
func (j *instanceValidator) Ready() bool {
	status, _ := hc.GetGlobalStatus()
	return status == string(hc.OK)
}

// Status -
func (j *instanceValidator) Status() error {
	j.logger.Trace("validating status")
	if status, _ := hc.GetGlobalStatus(); status != string(hc.OK) {
		err := fmt.Errorf("agent is marked as not running")
		j.logger.WithError(err).Trace("status failed")
		return err
	}
	return nil
}

// Execute -
func (j *instanceValidator) Execute() error {
	if getAPIValidator() != nil {
		j.logger.Trace("executing")
		if agent.validatingLock.value() != 0 {
			j.logger.Debug("skipping as previous instanceValidator is still running, will run on next interval")
			return nil
		}
		if err := agent.publishingLock.waitMaxDuration(time.Minute); err != nil {
			j.logger.Debug("skipping as the max duration waiting for publishing processes to finish was hit, will run on next interval")
			return nil
		}
		agent.validatingLock.increment()
		defer agent.validatingLock.decrement()
		j.validateAPIOnDataplane()
	} else {
		j.logger.Trace("no registered validator")
	}

	return nil
}

func (j *instanceValidator) validateAPIOnDataplane() {
	j.cacheLock.Lock()
	defer j.cacheLock.Unlock()

	logger := j.logger

	logger.Debug("validating api service instances on dataplane")
	// Validate the API on dataplane.  If API is not valid, mark the consumer instance as "DELETED"
	for _, key := range agent.cacheManager.GetAPIServiceInstanceKeys() {
		logger := logger.WithField("instanceCacheID", key)
		logger.Tracef("validating")

		instance, err := agent.cacheManager.GetAPIServiceInstanceByID(key)
		if err != nil {
			logger.WithError(err).Trace("could not get instance from cache")
			continue
		}
		logger = logger.WithField("name", instance.Name)

		externalAPIID, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if externalAPIID == "" {
			logger.Trace("could not get instance external id")
			continue // skip service instances without external api id
		} else if err != nil {
			logger.WithError(err).Trace("could not get instance external id")
		}
		logger = logger.WithField("externalAPIID", externalAPIID)
		externalAPIStage, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
		if externalAPIStage != "" {
			logger = logger.WithField("externalAPIStage", externalAPIStage)
		}
		externalPrimaryKey, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIPrimaryKey)
		if externalPrimaryKey != "" {
			logger = logger.WithField("externalPrimaryKey", externalPrimaryKey)
		}
		// Check if the consumer instance was published by agent, i.e. following attributes are set
		// - externalAPIID should not be empty
		// - externalAPIStage could be empty for dataplanes that do not support it
		logger.Trace("validating API Instance on dataplane")
		apiValidator := getAPIValidator()
		if externalAPIID != "" && !apiValidator(externalAPIID, externalAPIStage) {
			logger.Trace("removing API Instance no longer on dataplane")
			j.deleteServiceInstance(logger, instance, externalPrimaryKey, externalAPIID)
		}
	}

	logger.Debug("validating api services have at least one instance on dataplane")
	for _, key := range agent.cacheManager.GetAPIServiceKeys() {
		logger := logger.WithField("serviceCacheID", key)
		logger.Tracef("validating")

		service := agent.cacheManager.GetAPIServiceWithPrimaryKey(key)
		if service == nil {
			logger.Trace("service was no longer in the cache")
			continue
		}
		logger = logger.WithField("name", service.Name)
		instanceCount := agent.cacheManager.GetAPIServiceInstanceCount(service.Name)
		logger = logger.WithField("instanceCount", instanceCount)

		if agent.cacheManager.GetAPIServiceInstanceCount(service.Name) == 0 {
			logger.Trace("service has no more instances")
			j.deleteService(logger, service)
		}
	}
}

func (j *instanceValidator) deleteServiceInstance(logger log.FieldLogger, ri *apiV1.ResourceInstance, primaryKey, apiID string) {
	// delete if it is an api service instance
	logger = logger.WithField("instanceTitle", ri.Title)
	logger.Infof("API no longer exists on the dataplane, deleting the API Service Instance")

	err := agent.apicClient.DeleteAPIServiceInstance(ri.Name)
	if err != nil {
		logger.WithError(utilErrors.Wrap(ErrDeletingServiceInstanceItem, err.Error()).FormatError(ri.Title)).Error("deleting instance")
		return
	}
	agent.cacheManager.DeleteAPIServiceInstance(ri.Metadata.ID)

	logger.Debugf("Deleted API Service Instance item from Amplify Central")
}

func (j *instanceValidator) deleteService(logger log.FieldLogger, ri *apiV1.ResourceInstance) {
	logger = logger.WithField("serviceTitle", ri.Title)
	logger.Infof("API Service no longer has a service instance; deleting the API Service")

	// deleting the service will delete all associated resources, including the consumerInstance
	err := agent.apicClient.DeleteServiceByName(ri.Name)
	if err != nil {
		logger.WithError(utilErrors.Wrap(ErrDeletingService, err.Error()).FormatError(ri.Title)).Error("deleting service")
		return
	}
	agent.cacheManager.DeleteAPIService(ri.Name)

	logger.Debugf("Deleted API Service from Amplify Central")
}
