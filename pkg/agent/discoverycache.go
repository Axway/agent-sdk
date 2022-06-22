package agent

import (
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const (
	apiServerPageSize = 100
)

type discoveryCache struct {
	centralURL               string
	migrator                 migrate.Migrator
	logger                   log.FieldLogger
	handlers                 []handler.Handler
	isMpEnabled              bool
	client                   resourceClient
	additionalDiscoveryFuncs []discoverFunc
	watchTopic               *mv1.WatchTopic
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
		dc.additionalDiscoveryFuncs = append(dc.additionalDiscoveryFuncs, funcs...)
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
	watchTopic *mv1.WatchTopic,
	opts ...discoveryOpt,
) *discoveryCache {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("discoveryCache")

	dc := &discoveryCache{
		logger:                   logger,
		handlers:                 handlers,
		centralURL:               cfg.GetURL(),
		client:                   client,
		additionalDiscoveryFuncs: make([]discoverFunc, 0),
		watchTopic:               watchTopic,
	}

	for _, opt := range opts {
		opt(dc)
	}
	return dc
}

// execute rebuilds the discovery cache
func (dc *discoveryCache) execute() error {
	dc.logger.Debug("executing discovery cache")

	discoveryFuncs := dc.buildDiscoveryFuncs()
	if dc.additionalDiscoveryFuncs != nil {
		discoveryFuncs = append(discoveryFuncs, dc.additionalDiscoveryFuncs...)
	}

	err := dc.executeDiscoveryFuncs(discoveryFuncs)
	if err != nil {
		return err
	}

	// Now do the marketplace discovery funcs as the other functions have completed
	// AccessRequest cache need the APIServiceInstance cache to be fully loaded.
	if dc.isMpEnabled {
		marketplaceDiscoveryFuncs := dc.buildMarketplaceDiscoveryFuncs()
		err := dc.executeDiscoveryFuncs(marketplaceDiscoveryFuncs)
		if err != nil {
			return err
		}
	}

	dc.logger.Debug("cache has been updated")

	return nil
}

func (dc *discoveryCache) executeDiscoveryFuncs(discoveryFuncs []discoverFunc) error {
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

	return nil
}

func (dc *discoveryCache) buildDiscoveryFuncs() []discoverFunc {
	resources := make(map[string]discoverFunc)

	for _, filter := range dc.watchTopic.Spec.Filters {
		f := dc.buildResourceFunc(filter)
		if !isMPResource(filter.Kind) {
			resources[filter.Kind] = f
		}
	}

	var funcs []discoverFunc
	for _, f := range resources {
		funcs = append(funcs, f)
	}

	return funcs
}

func (dc *discoveryCache) buildMarketplaceDiscoveryFuncs() []discoverFunc {
	mpResources := make(map[string]discoverFunc)

	for _, filter := range dc.watchTopic.Spec.Filters {
		if isMPResource(filter.Kind) {
			f := dc.buildResourceFunc(filter)
			mpResources[filter.Kind] = f
		}
	}

	var funcs []discoverFunc
	marketplaceFuncs := dc.buildMarketplaceFuncs(mpResources)
	funcs = append(funcs, dc.handleMarketplaceFuncs(marketplaceFuncs))
	return funcs
}

func (dc *discoveryCache) buildMarketplaceFuncs(mpResources map[string]discoverFunc) []discoverFunc {
	var marketplaceFuncs []discoverFunc

	mApps, ok := mpResources[mv1.ManagedApplicationGVK().Kind]
	if ok {
		marketplaceFuncs = append(marketplaceFuncs, mApps)
	}

	accessReq, ok := mpResources[mv1.AccessRequestGVK().Kind]
	if ok {
		marketplaceFuncs = append(marketplaceFuncs, accessReq)
	}

	creds, ok := mpResources[mv1.CredentialGVK().Kind]
	if ok {
		marketplaceFuncs = append(marketplaceFuncs, creds)
	}

	return marketplaceFuncs
}

func (dc *discoveryCache) handleMarketplaceFuncs(marketplaceFuncs []discoverFunc) discoverFunc {
	return func() error {
		for _, f := range marketplaceFuncs {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	}
}

func (dc *discoveryCache) buildResourceFunc(filter mv1.WatchTopicSpecFilters) discoverFunc {
	return func() error {
		url := fmt.Sprintf("/%s/v1alpha1", filter.Group)
		if filter.Scope != nil {
			scopePlural, _ := v1.GetPluralFromKind(filter.Scope.Kind)
			url = fmt.Sprintf("%s/%s/%s", url, scopePlural, filter.Scope.Name)
		}

		var kindPlural, _ = v1.GetPluralFromKind(filter.Kind)
		url = fmt.Sprintf("%s/%s", url, kindPlural)

		logger := dc.logger.WithField("kind", filter.Kind)
		logger.Tracef("fetching %s and updating cache", filter.Kind)

		url = dc.formatResourceURL(url)
		resources, err := dc.client.GetAPIV1ResourceInstancesWithPageSize(nil, url, apiServerPageSize)
		if err != nil {
			return fmt.Errorf("failed to fetch resources of kind %s: %s", filter.Kind, err)
		}

		return dc.handleResourcesList(resources)
	}
}

func (dc *discoveryCache) handleResourcesList(list []*apiV1.ResourceInstance) error {
	for _, ri := range list {
		if dc.migrator != nil {
			var err error
			ri, err = dc.migrator.Migrate(ri)
			if err != nil {
				dc.logger.WithError(err).Error("failed to migrate resource")
			}
		}

		action := getAction(ri.Metadata.State)
		if err := dc.handleResource(ri, action); err != nil {
			dc.logger.
				WithError(err).
				WithField("kind", ri.Kind).
				WithField("name", ri.Name).
				Error("failed to handle resource")
		}
	}

	return nil
}

func (dc *discoveryCache) handleResource(ri *apiV1.ResourceInstance, action proto.Event_Type) error {
	ctx := handler.NewEventContext(action, nil, ri.Name, ri.Kind)
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

func getAction(state string) proto.Event_Type {
	if state == v1.ResourceDeleting {
		return proto.Event_UPDATED
	}
	return proto.Event_CREATED
}

func isMPResource(kind string) bool {
	switch kind {
	case mv1.ManagedApplicationGVK().Kind:
		return true
	case mv1.AccessRequestGVK().Kind:
		return true
	case mv1.CredentialGVK().Kind:
		return true
	default:
		return false
	}
}
