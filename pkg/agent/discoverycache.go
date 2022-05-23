package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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
	envName                  string
	centralURL               string
	migrator                 migrate.Migrator
	logger                   log.FieldLogger
	instanceCacheLock        *sync.Mutex
	discoveryCacheLock       *sync.Mutex
	handlers                 []handler.Handler
	agentType                config.AgentType
	isMpEnabled              bool
	client                   resourceClient
	additionalDiscoveryFuncs []discoverFunc
}

type resourceClient interface {
	GetAPIV1ResourceInstancesWithPageSize(query map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error)
}

// discoverFunc is the func definition for discovering resources to cache
type discoverFunc func() error

// discoveryOpt is a func that updates fields on the discoveryCache
type discoveryOpt func(dc *discoveryCache)

func withAdditionalDiscoverFuncs(funcs ...discoverFunc) discoveryOpt {
	return func(dc *discoveryCache) {
		dc.additionalDiscoveryFuncs = funcs
	}
}

func withMigration(mig migrate.Migrator) discoveryOpt {
	return func(dc *discoveryCache) {
		dc.migrator = mig
	}
}

func withMpEnabled(isEnabled bool) discoveryOpt {
	return func(dc *discoveryCache) {
		dc.isMpEnabled = isEnabled
	}
}

func newDiscoveryCache(
	cfg config.CentralConfig,
	client resourceClient,
	handlers []handler.Handler,
	opts ...discoveryOpt,
) *discoveryCache {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("discoveryCache")

	dc := &discoveryCache{
		agentType:          cfg.GetAgentType(),
		instanceCacheLock:  &sync.Mutex{},
		logger:             logger,
		discoveryCacheLock: &sync.Mutex{},
		handlers:           handlers,
		envName:            cfg.GetEnvironmentName(),
		centralURL:         cfg.GetURL(),
		client:             client,
	}

	for _, opt := range opts {
		opt(dc)
	}
	return dc
}

func (dc *discoveryCache) daFuncs() []discoverFunc {
	return []discoverFunc{
		dc.handleAPISvc,
		dc.handleServiceInstance,
		dc.handleCategories,
		dc.handleARD,
		dc.handleCRD,
		dc.handleAccessControlList,
		dc.handleMarketplaceResources,
	}
}

func (dc *discoveryCache) taFuncs() []discoverFunc {
	return []discoverFunc{
		dc.handleAPISvc,
		dc.handleServiceInstance,
		dc.handleMarketplaceResources,
	}
}

func (dc *discoveryCache) getDiscoveryFuncs() []discoverFunc {
	switch dc.agentType {
	case config.DiscoveryAgent:
		return dc.daFuncs()
	case config.TraceabilityAgent:
		return dc.taFuncs()
	}
	return nil
}

// execute rebuilds the discovery cache
func (dc *discoveryCache) execute() error {
	dc.discoveryCacheLock.Lock()
	defer dc.discoveryCacheLock.Unlock()

	dc.logger.Debug("executing resource cache update job")

	discoveryFuncs := dc.getDiscoveryFuncs()
	if dc.additionalDiscoveryFuncs != nil {
		discoveryFuncs = append(discoveryFuncs, dc.additionalDiscoveryFuncs...)
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

	dc.logger.Debug("cache has been updated")

	return nil
}

func (dc *discoveryCache) handleAPISvc() error {
	dc.logger.WithField("kind", "APIService").Trace("fetching API Services and updating cache")
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}
	svcLink := mv1.NewAPIService("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(svcLink)

	apiServices, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, svc := range apiServices {
		if dc.migrator != nil {
			var err error
			svc, err = dc.migrator.Migrate(svc)
			if err != nil {
				return fmt.Errorf("failed to migrate service: %s", err)
			}
		}

		if err := dc.handleResource(svc); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleServiceInstance() error {
	dc.logger.WithField("kind", "APIServiceInstance").Trace("fetching API Service Instances and updating cache")
	dc.instanceCacheLock.Lock()
	defer dc.instanceCacheLock.Unlock()

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	svcInstanceLink := mv1.NewAPIServiceInstance("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(svcInstanceLink)

	serviceInstances, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		e := utilErrors.Wrap(ErrUnableToGetAPIV1Resources, err.Error()).FormatError("APIServiceInstances")
		dc.logger.Error(e)
		return e
	}

	for _, instance := range serviceInstances {
		if err := dc.handleResource(instance); err != nil {
			return err
		}
	}
	return nil
}

func (dc *discoveryCache) handleCategories() error {
	dc.logger.WithField("kind", "Category").Trace("fetching Categories and updating cache")

	// Update cache with published resources
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	categoriesLink := catalog.NewCategory("").GetKindLink()
	url := dc.formatResourceURL(categoriesLink)

	categories, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, category := range categories {
		if err := dc.handleResource(category); err != nil {
			return err
		}
	}
	return nil
}

func (dc *discoveryCache) handleAccessControlList() error {
	dc.logger.WithField("kind", "AccessControlList").Trace("fetching AccessControlList and updating cache")

	// Update cache with published resources
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	acl, _ := mv1.NewAccessControlList("", mv1.EnvironmentGVK().Kind, dc.envName)
	aclLink := acl.GetKindLink()
	url := dc.formatResourceURL(aclLink)

	categories, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, category := range categories {
		if err := dc.handleResource(category); err != nil {
			return err
		}
	}
	return nil
}

func (dc *discoveryCache) handleARD() error {
	if !dc.isMpEnabled {
		return nil
	}
	dc.logger.WithField("kind", "AccessRequestDefinition").Trace("fetching AccessRequestDefinitions and updating cache")

	ardLink := mv1.NewAccessRequestDefinition("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(ardLink)
	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	ards, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)
	if err != nil {
		return err
	}

	for _, ard := range ards {
		if err := dc.handleResource(ard); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleManagedApp() error {
	if !dc.isMpEnabled {
		return nil
	}

	dc.logger.WithField("kind", "ManagedApplication").Trace("fetching ManagedApplications and updating cache")

	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.MarketplaceSubResource,
	}

	managedAppLink := mv1.NewManagedApplication("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(managedAppLink)
	managedApps, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)
	if err != nil {
		return err
	}

	for _, app := range managedApps {
		if err := dc.handleResource(app); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleCRD() error {
	if !dc.isMpEnabled {
		return nil
	}

	dc.logger.WithField("kind", "CredentialRequestDefinition").Trace("fetching CredentialRequestDefinitions and updating cache")

	crd := mv1.NewCredentialRequestDefinition("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(crd)

	query := map[string]string{
		apic.FieldsKey: apiServerFields,
	}

	crds, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(query, url, apiServerPageSize)
	if err != nil {
		return err
	}

	for _, crd := range crds {
		if err := dc.handleResource(crd); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleAccessRequest() error {
	if !dc.isMpEnabled {
		return nil
	}

	dc.logger.WithField("kind", "AccessRequest").Trace("fetching AccessRequests and updating cache")

	query := map[string]string{
		apic.FieldsKey: apiServerFields + "," + defs.Spec + "," + defs.ReferencesSubResource,
	}

	arLink := mv1.NewAccessRequest("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(arLink)
	accessRequests, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		query, url, apiServerPageSize,
	)

	if err != nil {
		return err
	}

	for _, req := range accessRequests {
		if err := dc.handleResource(req); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleCredential() error {
	if !dc.isMpEnabled {
		return nil
	}

	dc.logger.WithField("kind", "Credential").Trace("fetching Credentials and updating cache")

	credLink := mv1.NewCredential("", dc.envName).GetKindLink()
	url := dc.formatResourceURL(credLink)

	credentials, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(
		nil, url, apiServerPageSize,
	)

	if err != nil {
		return err
	}

	for _, cred := range credentials {
		if err := dc.handleResource(cred); err != nil {
			return err
		}
	}
	return nil
}

func (dc *discoveryCache) handleMarketplaceResources() error {
	funcs := []discoverFunc{
		dc.handleManagedApp,
		dc.handleAccessRequest,
	}

	if dc.agentType == config.DiscoveryAgent {
		funcs = append(funcs, dc.handleCredential)
	}

	for _, f := range funcs {
		if err := f(); err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) handleResource(ri *apiV1.ResourceInstance) error {
	ctx := handler.NewEventContext(proto.Event_CREATED, nil, ri.Name, ri.Kind)
	for _, h := range dc.handlers {
		err := h.Handle(ctx, nil, ri)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dc *discoveryCache) formatResourceURL(s string) string {
	return fmt.Sprintf("%s/apis%s", dc.centralURL, s)
}
