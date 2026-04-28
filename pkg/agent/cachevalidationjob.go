package agent

import (
	"fmt"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var errCacheOutOfSync = fmt.Errorf("persisted cache is out of sync with API server")

type cacheGetter interface {
	GetCachedResourcesByKind(group, kind, scopeName string) map[string]time.Time
}

// cacheValidator validates the persisted cache against the API server
type cacheValidator struct {
	logger     log.FieldLogger
	client     resourceClient
	watchTopic *management.WatchTopic
	cacheMan   cacheGetter
}

func newCacheValidator(
	client resourceClient,
	watchTopic *management.WatchTopic,
	cacheMan cacheGetter,
) *cacheValidator {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("cacheValidator")

	return &cacheValidator{
		logger:     logger,
		client:     client,
		watchTopic: watchTopic,
		cacheMan:   cacheMan,
	}
}

func (cv *cacheValidator) Execute() error {
	cv.logger.Debug("executing cache validation")

	for _, filter := range cv.watchTopic.Spec.Filters {
		if !cv.validateKind(filter) {
			return errCacheOutOfSync
		}
	}

	cv.logger.Debug("cache validation passed")
	return nil
}

func (cv *cacheValidator) validateKind(filter management.WatchTopicSpecFilters) bool {
	logger := cv.logger.WithField("kind", filter.Kind).WithField("group", filter.Group)

	if !agentcache.IsCachedKind(filter.Kind) {
		logger.Trace("skipping validation for kind")
		return true
	}

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

	url := ri.GetKindLink()
	if url == "" {
		logger.Trace("skipping validation, could not build resource URL")
		return true
	}

	// Fetch only name, audit metadata, and scope
	query := map[string]string{
		"fields": "name,kind,metadata.audit,metadata.scope",
	}

	logger.Trace("fetching resource metadata from API server for validation")
	serverResources, err := cv.client.GetAPIV1ResourceInstances(query, url)
	if err != nil {
		logger.WithError(err).Error("failed to fetch resource metadata for cache validation")
		return false
	}

	scopeName := ""
	if filter.Scope != nil {
		scopeName = filter.Scope.Name
	}
	cachedResources := cv.cacheMan.GetCachedResourcesByKind(filter.Group, filter.Kind, scopeName)

	// Build a map from server resources: composite key -> modifyTimestamp
	serverMap := make(map[string]time.Time, len(serverResources))
	for _, res := range serverResources {
		modTime := time.Time(res.Metadata.Audit.ModifyTimestamp)
		serverMap[agentcache.ResourceCacheKey(res.Kind, res.Metadata.Scope.Name, res.Name)] = modTime
	}

	// Count mismatch
	if len(serverMap) != len(cachedResources) {
		logger.
			WithField("serverCount", len(serverMap)).
			WithField("cacheCount", len(cachedResources)).
			Info("cache validation failed: resource count mismatch")
		return false
	}

	// Each server resource exists in cache with matching mod date
	for name, serverModTime := range serverMap {
		cacheModTime, exists := cachedResources[name]
		if !exists {
			logger.
				WithField("resource", name).
				Info("cache validation failed: resource not found in cache")
			return false
		}

		if !serverModTime.IsZero() && !cacheModTime.IsZero() && !serverModTime.Equal(cacheModTime) {
			logger.
				WithField("resource", name).
				WithField("serverModTime", serverModTime).
				WithField("cacheModTime", cacheModTime).
				Info("cache validation failed: modification time mismatch")
			return false
		}
	}

	logger.WithField("count", len(serverMap)).Trace("cache validation passed for kind")
	return true
}
