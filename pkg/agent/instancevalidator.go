package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/util"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	agentWarningTag     = "Agent Sync Warning"
	maxQueryParamLength = 2000
)

type resourcesInfo struct {
	names    []string
	kindLink string
	kind     string
}

type instanceValidator struct {
	jobs.Job
	logger              log.FieldLogger
	cacheLock           *sync.Mutex
	maxQueryParamLength int
}

func newInstanceValidator() *instanceValidator {
	return &instanceValidator{
		logger:              logger.WithComponent("instanceValidator"),
		cacheLock:           &sync.Mutex{},
		maxQueryParamLength: maxQueryParamLength,
	}
}

// Ready -
func (j *instanceValidator) Ready() bool {
	status, _ := hc.GetGlobalStatus()
	return status == string(hc.OK)
}

// Status -
func (j *instanceValidator) Status() error {
	j.logger.Trace("status check")
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
		PublishingLock()
		defer PublishingUnlock()
		j.validateAPIOnDataplane()
	} else {
		j.logger.Trace("no registered validator")
	}

	j.logger.Trace("finished executing")
	return nil
}

func (j *instanceValidator) validateAPIOnDataplane() {
	j.cacheLock.Lock()
	defer j.cacheLock.Unlock()

	apiServiceInstancesToUpdate := j.validateServiceInstances()
	apiServicesToUpdate := j.validateServices()

	j.addTags(apiServiceInstancesToUpdate)
	j.addTags(apiServicesToUpdate)
}

func (j *instanceValidator) addTags(info resourcesInfo) {
	if len(info.names) == 0 {
		j.logger.WithField("kind", info.kind).Trace("no instance validator tags to be added")
		return
	}
	ivLogger := j.logger.WithField("kind", info.kind)

	queries := j.constructAPIServerQueries("name", info.names)
	ris, err := j.getResources(queries, info.kindLink, info.kind)
	if err != nil {
		return
	}

	for _, ri := range ris {
		ivLogger := ivLogger.WithField("name", ri.GetName())
		if util.IsInArray(ri.GetTags(), agentWarningTag) {
			ivLogger.Trace("Agent sync warning tag already existing. Skipping update")
			continue
		}
		ri.SetTags(append(ri.GetTags(), agentWarningTag))
		_, err := agent.apicClient.UpdateResourceInstance(ri)
		if err != nil {
			ivLogger.WithError(err).Error("updating resource instance")
			continue
		}
		ivLogger.Warn("Added agent sync warning tag to API Resource on Amplify Central")
	}

}

func (j *instanceValidator) getResources(queries []string, kindLink, kind string) ([]apiV1.Interface, error) {
	ris := []apiV1.Interface{}
	ivLogger := j.logger.WithField("kindLink", kindLink)
	switch kind {
	case management.APIServiceInstanceGVK().Kind:
		for _, query := range queries {
			apiSIs, err := agent.apicClient.GetAPIServiceInstances(map[string]string{"query": query}, kindLink)
			if err != nil {
				ivLogger.WithField("query", query).WithError(err).Error("getting api service instances")
				return nil, err
			}
			for _, apiSI := range apiSIs {
				ris = append(ris, apiSI)
			}
		}

	case management.APIServiceGVK().Kind:
		for _, query := range queries {
			apis, err := agent.apicClient.GetAPIServices(map[string]string{"query": query}, kindLink)
			if err != nil {
				ivLogger.WithField("query", query).WithField("kindLink", kindLink).WithError(err).Error("getting api services")
				return nil, err
			}
			for _, api := range apis {
				ris = append(ris, api)
			}
		}
	}
	return ris, nil
}

func (j *instanceValidator) validateServiceInstances() resourcesInfo {
	apiServiceInstancesToUpdate := resourcesInfo{}
	for _, key := range agent.cacheManager.GetAPIServiceInstanceKeys() {
		ivLogger := j.logger
		ivLogger.Trace("validating api service instance on dataplane")

		instance, err := agent.cacheManager.GetAPIServiceInstanceByID(key)
		if err != nil || instance == nil {
			ivLogger.WithError(err).WithField("instanceCacheID", key).Trace("could not get instance from cache")
			continue
		}
		ivLogger = ivLogger.WithField("name", instance.Name)

		externalAPIID, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if externalAPIID == "" {
			ivLogger.Trace("could not get instance external id. skipping api validation")
			continue // skip service instances without external api id
		}
		ivLogger = ivLogger.WithField("externalAPIID", externalAPIID)
		externalAPIStage, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
		if externalAPIStage != "" {
			ivLogger = ivLogger.WithField("externalAPIStage", externalAPIStage)
		}
		externalPrimaryKey, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIPrimaryKey)
		if externalPrimaryKey != "" {
			ivLogger = ivLogger.WithField("externalPrimaryKey", externalPrimaryKey)
		}

		ivLogger.Trace("checking agent api validator")
		apiValidator := getAPIValidator()
		if !apiValidator(externalAPIID, externalAPIStage) {
			ivLogger.WithField("serviceTitle", instance.Title).Warn("API Service Instance no longer exists on the dataplane. Adding agent sync tag to the API Service Instance")
			apiServiceInstancesToUpdate.names = append(apiServiceInstancesToUpdate.names, instance.Name)
			if apiServiceInstancesToUpdate.kindLink == "" {
				apiServiceInstancesToUpdate.kindLink = instance.GetKindLink()
			}
		}
	}
	apiServiceInstancesToUpdate.kind = management.APIServiceInstanceGVK().Kind

	return apiServiceInstancesToUpdate
}

func (j *instanceValidator) validateServices() resourcesInfo {
	apiServicesToUpdate := resourcesInfo{}
	for _, key := range agent.cacheManager.GetAPIServiceKeys() {
		ivLogger := j.logger
		ivLogger.Trace("validating api service has at least one instance on dataplane")

		service := agent.cacheManager.GetAPIServiceWithPrimaryKey(key)
		if service == nil {
			ivLogger.WithField("serviceCacheID", key).Trace("service was no longer in the cache")
			continue
		}
		instanceCount := agent.cacheManager.GetAPIServiceInstanceCount(service.Name)
		ivLogger = ivLogger.WithField("instanceCount", instanceCount).WithField("name", service.Name)

		if agent.cacheManager.GetAPIServiceInstanceCount(service.Name) == 0 {
			ivLogger.WithField("serviceTitle", service.Title).Warn("API Service no longer has a service instance")
			apiServicesToUpdate.names = append(apiServicesToUpdate.names, service.Name)
			if apiServicesToUpdate.kindLink == "" {
				apiServicesToUpdate.kindLink = service.GetKindLink()
			}
		}
	}
	apiServicesToUpdate.kind = management.APIServiceGVK().Kind
	return apiServicesToUpdate
}

func (j *instanceValidator) constructAPIServerQueries(filterName string, resNames []string) []string {
	queries := []string{}
	if len(resNames) == 0 {
		return queries
	}
	offset := 0
	query := ""
	// safeguard against infinite loop just in case, not required
	for range len(resNames) {
		offset, query = j.addNames(filterName, resNames, offset)
		queries = append(queries, query)
		// all names have been added, exit
		if offset == -1 {
			return queries
		}
	}
	return queries
}

func (j *instanceValidator) addNames(filterName string, resNames []string, offset int) (int, string) {
	// format: `name=="name1" or name=="name2" or name=="name3"`
	query := fmt.Sprintf(`%s=="%s"`, filterName, resNames[offset])
	for i := offset + 1; i < len(resNames); i++ {
		extendedQuery := fmt.Sprintf(`%s or %s=="%s"`, query, filterName, resNames[i])
		if len(extendedQuery) >= j.maxQueryParamLength {
			return i, query
		}
		query = extendedQuery
	}
	return -1, query
}
