package cache

import (
	"sync"
)

type cacheMigrate func(key string) error

// migratePersistentCache is the top level migrator for all cache migrations
func (c *cacheManager) migratePersistentCache(key string) error {
	c.logger.Trace("checking if the persisted cache needs migrations")

	wg := sync.WaitGroup{}
	errs := make([]error, len(c.migrators))
	for i, m := range c.migrators {
		wg.Add(1)
		go func(index int, migFunc cacheMigrate) {
			defer wg.Done()
			errs[index] = migFunc(key)
		}(i, m)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
