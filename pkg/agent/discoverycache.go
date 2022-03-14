package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize   = 100
	healthcheckEndpoint = "central"
	apiServerFields     = "metadata,group,kind,name,title,owner,attributes,x-agent-details,finalizers"
	queryFormatString   = "%s>\"%s\""
)

var discoveryCacheLock *sync.Mutex

func init() {
	discoveryCacheLock = &sync.Mutex{}
}

type discoveryCache struct {
	jobs.Job
	lastServiceTime      time.Time
	lastInstanceTime     time.Time
	lastCategoryTime     time.Time
	refreshAll           bool
	getHCStatus          hc.GetStatusLevel
	instanceCacheLock    *sync.Mutex
	agentResourceManager resource.Manager
	migrator             migrate.AttrMigrator
}

func newDiscoveryCache(
	manager resource.Manager, getAll bool, instanceCacheLock *sync.Mutex, migrator migrate.AttrMigrator,
) *discoveryCache {
	return &discoveryCache{
		lastServiceTime:      time.Time{},
		lastInstanceTime:     time.Time{},
		lastCategoryTime:     time.Time{},
		refreshAll:           getAll,
		instanceCacheLock:    instanceCacheLock,
		agentResourceManager: manager,
		getHCStatus:          hc.GetStatus,
		migrator:             migrator,
	}
}

// Ready -
func (j *discoveryCache) Ready() bool {
	status := j.getHCStatus(healthcheckEndpoint)
	return status == hc.OK
}

// Status -
func (j *discoveryCache) Status() error {
	status := j.getHCStatus(healthcheckEndpoint)
	if status == hc.OK {
		return nil
	}
	return fmt.Errorf("could not establish a connection to APIC to update the cache")
}

// Execute -
func (j *discoveryCache) Execute() error {
	discoveryCacheLock.Lock()
	defer discoveryCacheLock.Unlock()
	log.Trace("executing API cache update job")
	err := j.updateAPICache()
	if err != nil {
		return err
	}

	if !agent.cacheManager.HasLoadedPersistedCache() {
		if agent.cfg.GetAgentType() == config.DiscoveryAgent {
			j.updateAPIServiceInstancesCache()
			j.updateCategoryCache()
		}
		if j.agentResourceManager != nil {
			j.agentResourceManager.FetchAgentResource()
		}
		j.updateManagedApplicationCache()
		j.updateAccessRequestCache()
	}

	return nil
}

func (j *discoveryCache) updateAPICache() error {
	log.Trace("updating API cache")

	existingAPIs := make(map[string]bool)
	apiServices, err := j.getAPIServices()
	if err != nil {
		return err
	}

	for _, svc := range apiServices {
		if j.migrator != nil {
			var err error
			svc, err = j.migrator.Migrate(svc)
			if err != nil {
				return fmt.Errorf("failed to migrate service: %s", err)
			}
		}

		externalAPIID, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIID)
		// skip service without external api id
		if externalAPIID == "" {
			continue
		}

		// Update the lastServiceTime based on the newest service found
		thisTime := time.Time(svc.Metadata.Audit.CreateTimestamp)
		if j.lastServiceTime.Before(thisTime) {
			j.lastServiceTime = thisTime
		}

		agent.cacheManager.AddAPIService(svc)
		primaryKey, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIPrimaryKey)
		if primaryKey != "" {
			existingAPIs[primaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}
	}

	if j.refreshAll {
		// Remove items that are not published as Resources
		cacheKeys := agent.cacheManager.GetAPIServiceKeys()
		for _, key := range cacheKeys {
			if _, ok := existingAPIs[key]; !ok {
				agent.cacheManager.DeleteAPIService(key)
			}
		}
	}

	return nil
}

func (j *discoveryCache) getAPIServices() ([]*apiV1.ResourceInstance, error) {
	if agent.cacheManager.HasLoadedPersistedCache() {
		return j.getCachedAPIServices(), nil
	}
	// Update cache with published resources
	return j.fetchAPIServices()
}

func (j *discoveryCache) getCachedAPIServices() []*apiV1.ResourceInstance {
	resources := make([]*apiV1.ResourceInstance, 0)
	cache := agent.cacheManager.GetAPIServiceCache()

	for _, key := range cache.GetKeys() {
		item, _ := cache.Get(key)
		if item == nil {
			continue
		}

		apiSvc, ok := item.(*apiV1.ResourceInstance)
		if ok {
			resources = append(resources, apiSvc)
		}
	}

	return resources
}

func (j *discoveryCache) fetchAPIServices() ([]*apiV1.ResourceInstance, error) {
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastServiceTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(
			queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(apiV1.APIServerTimeFormat),
		)
	}

	return GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetServicesURL(), apiServerPageSize,
	)
}

func (j *discoveryCache) updateAPIServiceInstancesCache() {
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastInstanceTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(
			queryFormatString, apic.CreateTimestampQueryKey, j.lastServiceTime.Format(apiV1.APIServerTimeFormat),
		)
	}

	j.lastInstanceTime = time.Now()
	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetInstancesURL(), apiServerPageSize,
	)
	if err != nil {
		log.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances"))
		return
	}

	j.instanceCacheLock.Lock()
	defer j.instanceCacheLock.Unlock()
	if j.refreshAll {
		agent.cacheManager.DeleteAllAPIServiceInstance()
	}
	for _, instance := range serviceInstances {
		id, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if id == "" {
			continue // skip instance without external api id
		}
		agent.cacheManager.AddAPIServiceInstance(instance)
		if !j.refreshAll {
			// Update the lastInstanceTime based on the newest instance found
			thisTime := time.Time(instance.Metadata.Audit.CreateTimestamp)
			if j.lastInstanceTime.Before(thisTime) {
				j.lastInstanceTime = thisTime
			}
		}
	}
}

func (j *discoveryCache) updateCategoryCache() {
	log.Trace("updating category cache")

	// Update cache with published resources
	existingCategories := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	if !j.lastCategoryTime.IsZero() && !j.refreshAll {
		query[apic.QueryKey] = fmt.Sprintf(
			queryFormatString, apic.CreateTimestampQueryKey, j.lastCategoryTime.Format(apiV1.APIServerTimeFormat),
		)
	}
	categories, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetCategoriesURL(), apiServerPageSize,
	)

	for _, category := range categories {
		// Update the lastCategoryTime based on the newest category found
		thisTime := time.Time(category.Metadata.Audit.CreateTimestamp)
		if j.lastCategoryTime.Before(thisTime) {
			j.lastCategoryTime = thisTime
		}

		agent.cacheManager.AddCategory(category)
		existingCategories[category.Name] = true
	}

	if j.refreshAll {
		// Remove categories that no longer exist
		cacheKeys := agent.cacheManager.GetCategoryKeys()
		for _, key := range cacheKeys {
			if _, ok := existingCategories[key]; !ok {
				agent.cacheManager.DeleteCategory(key)
			}
		}
	}
}

func (j *discoveryCache) updateManagedApplicationCache() {
	log.Trace("updating managed application cache")

	// Update cache with published resources
	existingManagedApplications := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields + ",x-marketplace-subject",
	}

	managedApps, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetEnvironmentURL()+"/managedapplications", apiServerPageSize,
	)

	for _, managedApp := range managedApps {
		agent.cacheManager.AddManagedApplication(managedApp)
		existingManagedApplications[managedApp.Metadata.ID] = true
	}

	if j.refreshAll {
		// Remove managed applications that no longer exist
		cacheKeys := agent.cacheManager.GetManagedApplicationCacheKeys()
		for _, key := range cacheKeys {
			if _, ok := existingManagedApplications[key]; !ok {
				agent.cacheManager.DeleteManagedApplication(key)
			}
		}
	}
}

func (j *discoveryCache) updateAccessRequestCache() {
	log.Trace("updating access request cache")

	// Update cache with published resources
	existingAccessRequests := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields + ",spec,x-marketplace-subscription",
	}

	accessRequests, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetEnvironmentURL()+"/accessrequests", apiServerPageSize,
	)

	for _, accessRequest := range accessRequests {
		agent.cacheManager.AddAccessRequest(accessRequest)
		ar := &mv1.AccessRequest{}
		ar.FromInstance(accessRequest)
		existingAccessRequests[accessRequest.Metadata.ID] = true
		j.addSubscription(ar)
	}

	if j.refreshAll {
		// Remove access requests that no longer exist
		cacheKeys := agent.cacheManager.GetManagedApplicationCacheKeys()
		for _, key := range cacheKeys {
			if _, ok := existingAccessRequests[key]; !ok {
				agent.cacheManager.DeleteAccessRequest(key)
			}
		}
	}
}

func (j *discoveryCache) addSubscription(ar *mv1.AccessRequest) {
	ars := ar.GetSubResource("x-marketplace-subscription")
	if ars != nil {
		arSubscription := ars.(map[string]interface{})
		propertyVal := arSubscription["name"]
		if propertyVal == nil {
			return
		}
		subscriptionName := propertyVal.(string)
		subscription := agent.cacheManager.GetSubscriptionByName(subscriptionName)
		if subscription == nil {
			subscription, err := j.fetchSubscription(subscriptionName)
			if err == nil {
				agent.cacheManager.AddSubscription(subscription)
			}
		}
	}
}

func (j *discoveryCache) fetchSubscription(subscriptionName string) (*apiV1.ResourceInstance, error) {
	if subscriptionName != "" {
		url := fmt.Sprintf(
			"/catalog/v1alpha1/subscriptions/%s",
			subscriptionName,
		)
		return GetCentralClient().GetResource(url)
	}
	return nil, nil
}
