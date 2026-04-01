package cache

import (
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

// GetCachedResourcesByKind returns a map of resource name to modification timestamp
// for all cached resources of the given group and kind. When scopeName is non-empty,
// only resources whose scope matches scopeName are included.
func (c *cacheManager) GetCachedResourcesByKind(group, kind, scopeName string) map[string]time.Time {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	resourceCache := c.getCacheForKind(kind)
	if resourceCache == nil {
		// Fall back to the watch resource map for kinds not in a dedicated cache
		return c.getWatchResourcesByKind(group, kind, scopeName)
	}

	return c.extractResourceSummary(resourceCache, scopeName)
}

// getCacheForKind returns the dedicated cache for the given resource kind, or nil
// if the kind is stored in the watch resource map.
func (c *cacheManager) getCacheForKind(kind string) cache.Cache {
	switch kind {
	case management.APIServiceGVK().Kind:
		return c.apiMap
	case management.APIServiceInstanceGVK().Kind:
		return c.instanceMap
	case management.ManagedApplicationGVK().Kind:
		return c.managedApplicationMap
	case management.AccessRequestGVK().Kind:
		return c.accessRequestMap
	case management.AccessRequestDefinitionGVK().Kind:
		return c.ardMap
	case management.CredentialRequestDefinitionGVK().Kind:
		return c.crdMap
	case management.ApplicationProfileDefinitionGVK().Kind:
		return c.apdMap
	case management.ComplianceRuntimeResultGVK().Kind:
		return c.crrMap
	default:
		return nil
	}
}

// FlushKind empties the dedicated cache for the given resource kind.
// Kinds without a dedicated cache are silently ignored.
func (c *cacheManager) FlushKind(kind string) {
	c.ApplyResourceReadLock()
	defer c.ReleaseResourceReadLock()

	if resourceCache := c.getCacheForKind(kind); resourceCache != nil {
		resourceCache.Flush()
	}
}

// ResourceCacheKey builds a unique cache key from kind, scope name, and resource name.
func ResourceCacheKey(kind, scopeName, name string) string {
	return kind + "/" + scopeName + "/" + name
}

// extractResourceSummary iterates the cache keys and returns a composite key -> modifyTimestamp.
// When scopeName is non-empty, only resources whose scope matches scopeName are included.
func (c *cacheManager) extractResourceSummary(resourceCache cache.Cache, scopeName string) map[string]time.Time {
	result := make(map[string]time.Time)
	keys := resourceCache.GetKeys()

	for _, key := range keys {
		item, err := resourceCache.Get(key)
		if err != nil {
			continue
		}
		ri, ok := item.(*v1.ResourceInstance)
		if !ok || ri == nil {
			continue
		}
		if scopeName != "" && ri.Metadata.Scope.Name != scopeName {
			continue
		}
		modTime := time.Time(ri.Metadata.Audit.ModifyTimestamp)
		result[ResourceCacheKey(ri.Kind, ri.Metadata.Scope.Name, ri.Name)] = modTime
	}

	return result
}

// getWatchResourcesByKind iterates the watch resource map and returns resources
// matching the given group and kind. When scopeName is non-empty, only resources
// whose scope matches scopeName are included.
func (c *cacheManager) getWatchResourcesByKind(group, kind, scopeName string) map[string]time.Time {
	result := make(map[string]time.Time)

	keys := c.watchResourceMap.GetKeys()
	for _, key := range keys {
		item, err := c.watchResourceMap.Get(key)
		if err != nil {
			continue
		}
		ri, ok := item.(*v1.ResourceInstance)
		if !ok || ri == nil {
			continue
		}
		if scopeName != "" && ri.Metadata.Scope.Name != scopeName {
			continue
		}
		if ri.Group == group && ri.Kind == kind {
			modTime := time.Time(ri.Metadata.Audit.ModifyTimestamp)
			result[ResourceCacheKey(ri.Kind, ri.Metadata.Scope.Name, ri.Name)] = modTime
		}
	}

	return result
}
