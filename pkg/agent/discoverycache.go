package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	utilErrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize = 100
	apiServerFields   = "metadata,group,kind,name,title,owner,attributes,x-agent-details,finalizers"
)

type discoveryCache struct {
	instanceCacheLock    *sync.Mutex
	agentResourceManager resource.Manager
	migrator             migrate.Migrator
	logger               log.FieldLogger
	discoveryCacheLock   *sync.Mutex
}

func newDiscoveryCache(manager resource.Manager, migrations migrate.Migrator) *discoveryCache {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("discoveryCache")
	return &discoveryCache{
		instanceCacheLock:    &sync.Mutex{},
		agentResourceManager: manager,
		migrator:             migrations,
		logger:               logger,
		discoveryCacheLock:   &sync.Mutex{},
	}
}

// execute rebuilds the discovery cache
func (j *discoveryCache) execute() error {
	j.discoveryCacheLock.Lock()
	defer j.discoveryCacheLock.Unlock()

	j.logger.Trace("executing resource cache update job")
	err := j.updateAPICache()
	if err != nil {
		return err
	}

	j.updateAPIServiceInstancesCache()

	switch agent.cfg.GetAgentType() {
	case config.DiscoveryAgent:
		j.updateCategoryCache()
		j.updateCRDCache()
		j.updateARDCache()
	case config.TraceabilityAgent:
		j.updateManagedApplicationCache()
		j.updateAccessRequestCache()
	}

	if j.agentResourceManager != nil {
		j.agentResourceManager.FetchAgentResource()
	}

	agent.cacheManager.SaveCache()
	j.logger.Trace("cache has been updated and saved")

	return nil
}

func (j *discoveryCache) updateAPICache() error {
	existingAPIs := make(map[string]bool)
	apiServices, err := j.fetchAPIServices()
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

		agent.cacheManager.AddAPIService(svc)
		primaryKey, _ := util.GetAgentDetailsValue(svc, defs.AttrExternalAPIPrimaryKey)
		if primaryKey != "" {
			existingAPIs[primaryKey] = true
		} else {
			existingAPIs[externalAPIID] = true
		}
	}

	return nil
}

func (j *discoveryCache) fetchAPIServices() ([]*apiV1.ResourceInstance, error) {
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	return GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetServicesURL(), apiServerPageSize,
	)
}

func (j *discoveryCache) updateAPIServiceInstancesCache() {
	j.instanceCacheLock.Lock()
	defer j.instanceCacheLock.Unlock()

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetInstancesURL(), apiServerPageSize,
	)
	if err != nil {
		j.logger.Error(utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances"))
		return
	}

	for _, instance := range serviceInstances {
		id, _ := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
		if id == "" {
			continue // skip instance without external api id
		}
		agent.cacheManager.AddAPIServiceInstance(instance)
	}
}

func (j *discoveryCache) updateCategoryCache() {
	j.logger.Trace("updating category cache")

	// Update cache with published resources
	existingCategories := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	categories, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetCategoriesURL(), apiServerPageSize,
	)

	for _, category := range categories {
		agent.cacheManager.AddCategory(category)
		existingCategories[category.Name] = true
	}
}

func (j *discoveryCache) updateARDCache() {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return
	}
	j.logger.Trace("updating access request definition cache")

	// create an empty accessrequestdef to gen url
	url := fmt.Sprintf("%s/apis%s", agent.cfg.GetURL(), mv1.NewAccessRequestDefinition("", agent.cfg.GetEnvironmentName()).GetKindLink())

	// Update cache with published resources
	existingARDs := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	ards, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)

	for _, ard := range ards {
		agent.cacheManager.AddAccessRequestDefinition(ard)
		existingARDs[ard.Metadata.ID] = true
	}
}

func (j *discoveryCache) updateManagedApplicationCache() {
	j.logger.Trace("updating managed application cache")

	// Update cache with published resources
	existingManagedApplications := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.MarketplaceSubResource,
	}

	managedApps, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetEnvironmentURL()+"/managedapplications", apiServerPageSize,
	)

	for _, managedApp := range managedApps {
		agent.cacheManager.AddManagedApplication(managedApp)
		existingManagedApplications[managedApp.Metadata.ID] = true
	}
}

func (j *discoveryCache) updateCRDCache() {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return
	}
	j.logger.Trace("updating credential request definition cache")

	// create an empty credentialrequestdef to gen url
	crd := mv1.NewCredentialRequestDefinition("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := fmt.Sprintf("%s/apis%s", agent.cfg.GetURL(), crd)

	// Update cache with published resources
	existingCRDs := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	crds, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)

	for _, crd := range crds {
		agent.cacheManager.AddCredentialRequestDefinition(crd)
		existingCRDs[crd.Metadata.ID] = true
	}
}

func (j *discoveryCache) updateAccessRequestCache() {
	j.logger.Trace("updating access request cache")

	// Update cache with published resources
	existingAccessRequests := make(map[string]bool)
	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.Spec + "," + defs.ReferencesSubResource,
	}

	accessRequests, _ := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, agent.cfg.GetEnvironmentURL()+"/accessrequests", apiServerPageSize,
	)

	for _, accessRequest := range accessRequests {
		ar := &mv1.AccessRequest{}
		ar.FromInstance(accessRequest)
		agent.cacheManager.AddAccessRequest(ar)
		existingAccessRequests[accessRequest.Metadata.ID] = true
		j.addSubscription(ar)
	}
}

func (j *discoveryCache) addSubscription(ar *mv1.AccessRequest) {
	subscriptionName := defs.GetSubscriptionNameFromAccessRequest(ar)
	if subscriptionName == "" {
		return
	}

	subscription := agent.cacheManager.GetSubscriptionByName(subscriptionName)
	if subscription == nil {
		subscription, err := j.fetchSubscription(subscriptionName)
		if err == nil {
			agent.cacheManager.AddSubscription(subscription)
		}
	}
}

func (j *discoveryCache) fetchSubscription(subscriptionName string) (*apiV1.ResourceInstance, error) {
	if subscriptionName == "" {
		return nil, nil
	}

	url := fmt.Sprintf(
		"/catalog/v1alpha1/subscriptions/%s",
		subscriptionName,
	)
	return GetCentralClient().GetResource(url)
}
