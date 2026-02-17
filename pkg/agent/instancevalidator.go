package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/util"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
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
	j.addTags(apiServiceInstancesToUpdate)

	apiServicesToUpdate := j.validateServices()
	j.addTags(apiServicesToUpdate)
}

func (j *instanceValidator) addTags(info resourcesInfo) {
	ivLogger := j.logger.WithField("kind", info.kind).WithField("kindLink", info.kindLink)
	if len(info.names) == 0 {
		j.logger.Trace("no instance validator tags to be added")
		return
	}

	queries := j.constructAPIServerQueries("name", info.names)
	ris := []*v1.ResourceInstance{}
	for _, query := range queries {
		apis, err := agent.apicClient.GetAPIV1ResourceInstances(map[string]string{"query": query}, info.kindLink)
		if err != nil {
			j.logger.WithField("query", query).WithError(err).Error("getting resources")
			return
		}
		ris = append(ris, apis...)
	}

	for _, ri := range ris {
		ivLogger := ivLogger.WithField("name", ri.GetName())
		if util.IsInArray(ri.GetTags(), util.AgentWarningTag) {
			ivLogger.Trace("Agent sync warning tag already existing. Skipping update")
			continue
		}
		ri.SetTags(append(ri.GetTags(), util.AgentWarningTag))
		_, err := agent.apicClient.UpdateResourceInstance(ri)
		if err != nil {
			ivLogger.WithError(err).Error("updating resource instance")
			continue
		}
		ivLogger.Warn("Added agent sync warning tag to API Resource on Amplify Central")
	}
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

		if util.IsInArray(instance.GetTags(), util.AgentWarningTag) {
			ivLogger.WithField("serviceTitle", instance.Title).Trace("skipping already tagged instance")
			continue
		}
		externalAPIID, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if externalAPIID == "" {
			ivLogger.Trace("could not get instance external id. skipping api validation")
			continue // skip service instances without external api id
		}
		ivLogger = ivLogger.WithField(defs.AttrExternalAPIID, externalAPIID)
		externalAPIStage, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIStage)
		if externalAPIStage != "" {
			ivLogger = ivLogger.WithField(defs.AttrExternalAPIStage, externalAPIStage)
		}
		externalPrimaryKey, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIPrimaryKey)
		if externalPrimaryKey != "" {
			ivLogger = ivLogger.WithField(defs.AttrExternalAPIPrimaryKey, externalPrimaryKey)
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
		if util.IsInArray(service.GetTags(), util.AgentWarningTag) {
			ivLogger.WithField("serviceTitle", service.Title).Trace("skipping already tagged service")
			continue
		}
		apiSIs := agent.cacheManager.GetAPIServiceInstancesByService(service.Name)
		count := 0
		for _, apiSI := range apiSIs {
			if util.IsInArray(apiSI.GetTags(), util.AgentWarningTag) {
				count++
			}
		}
		if count != 0 && len(apiSIs) == count {
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
