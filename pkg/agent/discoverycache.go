package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

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
	handlers             []handler.Handler
}

type discoverFunc func() error

func newDiscoveryCache(
	manager resource.Manager, migrations migrate.Migrator, handlers []handler.Handler,
) *discoveryCache {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("discoveryCache")
	return &discoveryCache{
		instanceCacheLock:    &sync.Mutex{},
		agentResourceManager: manager,
		migrator:             migrations,
		logger:               logger,
		discoveryCacheLock:   &sync.Mutex{},
		handlers:             handlers,
	}
}

func (j *discoveryCache) daEndpoints() []discoverFunc {
	endpoints := []discoverFunc{
		j.handleCategories,
		j.handleCRD,
		j.handleARD,
		j.handleMarketplaceResources,
	}
	return endpoints
}

func (j *discoveryCache) taEndpoints() []discoverFunc {
	endpoints := []discoverFunc{
		j.handleMarketplaceResources,
	}
	return endpoints
}

func (j *discoveryCache) discoveryFuncs() []discoverFunc {
	endpoints := []discoverFunc{
		j.handleAPISvc,
		j.handleServiceInstance,
	}

	switch agent.cfg.GetAgentType() {
	case config.DiscoveryAgent:
		endpoints = append(endpoints, j.daEndpoints()...)
	case config.TraceabilityAgent:
		endpoints = append(endpoints, j.taEndpoints()...)
	}

	return endpoints
}

// execute rebuilds the discovery cache
func (j *discoveryCache) execute() error {
	j.discoveryCacheLock.Lock()
	defer j.discoveryCacheLock.Unlock()

	j.logger.Debug("executing resource cache update job")

	discoveryFuncs := j.discoveryFuncs()
	if j.agentResourceManager != nil {
		discoveryFuncs = append(discoveryFuncs, j.agentResourceManager.FetchAgentResource)
	}

	errCh := make(chan error, len(discoveryFuncs))
	wg := &sync.WaitGroup{}

	for _, fun := range discoveryFuncs {
		wg.Add(1)

		go func(f func() error) {
			defer wg.Done()

			err := f()
			errCh <- err
		}(fun)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	agent.cacheManager.SaveCache()
	j.logger.Debug("cache has been updated and saved")

	return nil
}

func (j *discoveryCache) handleAPISvc() error {
	j.logger.WithField("kind", "APIService").Trace("fetching API Services and updating cache")
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}
	svcLink := mv1.NewAPIService("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(svcLink)

	apiServices, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
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

		if err := j.handleResource(svc); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleServiceInstance() error {
	j.logger.WithField("kind", "APIServiceInstance").Trace("fetching API Service Instances and updating cache")
	j.instanceCacheLock.Lock()
	defer j.instanceCacheLock.Unlock()

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	svcInstanceLink := mv1.NewAPIServiceInstance("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(svcInstanceLink)

	serviceInstances, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		e := utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances")
		j.logger.Error(e)
		return e
	}

	for _, instance := range serviceInstances {
		if err := j.handleResource(instance); err != nil {
			return err
		}
	}
	return nil
}

func (j *discoveryCache) handleCategories() error {
	j.logger.WithField("kind", "Category").Trace("fetching Categories and updating cache")

	// Update cache with published resources
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	categoriesLink := catalog.NewCategory("").GetKindLink()
	url := formatResourceURL(categoriesLink)

	categories, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, category := range categories {
		if err := j.handleResource(category); err != nil {
			return err
		}
	}
	return nil
}

func (j *discoveryCache) handleARD() error {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil
	}
	j.logger.WithField("kind", "AccessRequestDefinition").Trace("fetching AccessRequestDefinitions and updating cache")

	ardLink := mv1.NewAccessRequestDefinition("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(ardLink)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	ards, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)
	if err != nil {
		return err
	}

	for _, ard := range ards {
		if err := j.handleResource(ard); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleManagedApp() error {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil
	}

	j.logger.WithField("kind", "ManagedApplication").Trace("fetching ManagedApplications and updating cache")

	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.MarketplaceSubResource,
	}

	managedAppLink := mv1.NewManagedApplication("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(managedAppLink)
	managedApps, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, app := range managedApps {
		if err := j.handleResource(app); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleCRD() error {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil
	}

	j.logger.WithField("kind", "CredentialRequestDefinition").Trace("fetching CredentialRequestDefinitions and updating cache")

	crd := mv1.NewCredentialRequestDefinition("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(crd)

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	crds, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)
	if err != nil {
		return err
	}

	for _, crd := range crds {
		if err := j.handleResource(crd); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleAccessRequest() error {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil
	}

	j.logger.WithField("kind", "AccessRequest").Trace("fetching AccessRequests and updating cache")

	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.Spec + "," + defs.ReferencesSubResource,
	}

	arLink := mv1.NewAccessRequest("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(arLink)
	accessRequests, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)

	if err != nil {
		return err
	}

	for _, req := range accessRequests {
		if err := j.handleResource(req); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleCredential() error {
	if agent.agentFeaturesCfg == nil || !agent.agentFeaturesCfg.MarketplaceProvisioningEnabled() {
		return nil
	}

	j.logger.WithField("kind", "Credential").Trace("fetching Credentials and updating cache")

	credLink := mv1.NewCredential("", agent.cfg.GetEnvironmentName()).GetKindLink()
	url := formatResourceURL(credLink)

	credentials, err := GetCentralClient().GetAPIV1ResourceInstancesWithPageSize(
		nil, url, apiServerPageSize,
	)

	if err != nil {
		return err
	}

	for _, cred := range credentials {
		if err := j.handleResource(cred); err != nil {
			return err
		}
	}
	return nil
}

func (j *discoveryCache) handleMarketplaceResources() error {
	funcs := []discoverFunc{
		j.handleManagedApp,
		j.handleAccessRequest,
		j.handleCredential,
	}

	for _, f := range funcs {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (j *discoveryCache) handleResource(ri *apiV1.ResourceInstance) error {
	for _, h := range j.handlers {
		ctx := handler.NewEventContext(proto.Event_CREATED, nil, ri.Name, ri.Kind)
		err := h.Handle(ctx, nil, ri)
		if err != nil {
			return err
		}
	}

	return nil
}

func formatResourceURL(s string) string {
	return fmt.Sprintf("%s/apis%s", agent.cfg.GetURL(), s)
}
