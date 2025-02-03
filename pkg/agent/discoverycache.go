package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type discoveryCache struct {
	centralURL               string
	migrator                 migrate.Migrator
	logger                   log.FieldLogger
	handlers                 []handler.Handler
	client                   resourceClient
	additionalDiscoveryFuncs []discoverFunc
	watchTopic               *management.WatchTopic
	preMPFunc                func() error
	initialized              bool
}

type resourceClient interface {
	GetAPIV1ResourceInstances(query map[string]string, URL string) ([]*apiv1.ResourceInstance, error)
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

// set a function to call after syncing all the cached resources except marketplace resources
func preMarketplaceSetup(f func() error) discoveryOpt {
	return func(dc *discoveryCache) {
		dc.preMPFunc = f
	}
}

func newDiscoveryCache(
	cfg config.CentralConfig,
	client resourceClient,
	handlers []handler.Handler,
	watchTopic *management.WatchTopic,
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

	err = dc.callPreMPFunc()
	if err != nil {
		dc.logger.WithError(err).Error("error finalizing setup prior to marketplace resource syncing")
		return err
	}

	// Now do the marketplace discovery funcs as the other functions have completed
	// AccessRequest cache need the APIServiceInstance cache to be fully loaded.
	marketplaceDiscoveryFuncs := dc.buildMarketplaceDiscoveryFuncs()
	err = dc.executeDiscoveryFuncs(marketplaceDiscoveryFuncs)
	if err != nil {
		return err
	}

	dc.logger.Debug("cache has been updated")
	return nil
}

func (dc *discoveryCache) callPreMPFunc() error {
	if dc.preMPFunc == nil || dc.initialized {
		return nil
	}
	defer func() {
		dc.initialized = true
	}()
	return dc.preMPFunc()
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
		kind := filter.Kind
		scope := ""
		if filter.Scope != nil && filter.Scope.Name != "" {
			scope = filter.Scope.Name
		}
		key := fmt.Sprintf("%s:%s", kind, scope)
		dc.logger.Debugf("adding function kind:%s,scope:%s to be executed", kind, scope)
		f := dc.buildResourceFunc(filter)
		if !isMPResource(filter.Kind) {
			resources[key] = f
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

	mApps, ok := mpResources[management.ManagedApplicationGVK().Kind]
	if ok {
		marketplaceFuncs = append(marketplaceFuncs, mApps)
	}

	accessReq, ok := mpResources[management.AccessRequestGVK().Kind]
	if ok {
		marketplaceFuncs = append(marketplaceFuncs, accessReq)
	}

	creds, ok := mpResources[management.CredentialGVK().Kind]
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

func (dc *discoveryCache) buildResourceFunc(filter management.WatchTopicSpecFilters) discoverFunc {
	return func() error {
		ri := apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				GroupVersionKind: apiv1.GroupVersionKind{
					GroupKind: apiv1.GroupKind{
						Group: filter.Group,
						Kind:  filter.Kind,
					},
					APIVersion: "v1alpha1",
				},
			},
		}
		if filter.Scope != nil {
			ri.Metadata.Scope.Kind = filter.Scope.Kind
			ri.Metadata.Scope.Name = filter.Scope.Name
		}

		logger := dc.logger.WithField("kind", filter.Kind)
		logger.Tracef("fetching %s and updating cache", filter.Kind)

		resources, err := dc.client.GetAPIV1ResourceInstances(nil, ri.GetKindLink())
		if err != nil {
			return fmt.Errorf("failed to fetch resources of kind %s: %s", filter.Kind, err)
		}

		return dc.handleResourcesList(resources)
	}
}

func (dc *discoveryCache) handleResourcesList(list []*apiv1.ResourceInstance) error {
	for _, ri := range list {
		if dc.migrator != nil {
			ctx := context.Background()
			ctx = context.WithValue(context.WithValue(ctx, log.KindCtx, ri.Kind), log.NameCtx, ri.Name)

			logger := log.NewLoggerFromContext(ctx)

			logger.Trace("handle migration")
			var err error
			ri, err = dc.migrator.Migrate(ctx, ri)
			if err != nil {
				dc.logger.WithError(err).Error("failed to migrate resource")
			}
		}

		action := getAction(ri.Metadata.State)
		if err := dc.handleResource(ri, action); err != nil {
			logger.
				WithError(err).
				Error("failed to migrate resource")
		}
	}

	return nil
}

func (dc *discoveryCache) handleResource(ri *apiv1.ResourceInstance, action proto.Event_Type) error {
	ctx := handler.NewEventContext(action, nil, ri.Name, ri.Kind)
	for _, h := range dc.handlers {
		err := h.Handle(ctx, nil, ri)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAction(state string) proto.Event_Type {
	if state == apiv1.ResourceDeleting {
		return proto.Event_UPDATED
	}
	return proto.Event_CREATED
}

func isMPResource(kind string) bool {
	switch kind {
	case management.ManagedApplicationGVK().Kind:
		return true
	case management.AccessRequestGVK().Kind:
		return true
	case management.CredentialGVK().Kind:
		return true
	default:
		return false
	}
}
