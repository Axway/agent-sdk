package sampling

import "github.com/Axway/agent-sdk/pkg/agent/cache"

var cacheManagerGetter func() cache.Manager

func SetCacheManagerGetter(getter func() cache.Manager) {
	cacheManagerGetter = getter
}

func getCacheManager() cache.Manager {
	if cacheManagerGetter != nil {
		return cacheManagerGetter()
	}
	return nil
}
