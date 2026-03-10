package cache

// cacheMigrate is deprecated - no longer needed with bbolt persistence
// Left for future compatibility if needed
type cacheMigrate func(key string) error

// migratePersistentCache is deprecated - bbolt handles persistence automatically
// This is a no-op function left for compatibility
func (c *cacheManager) migratePersistentCache(key string) error {
	// No migration needed with bbolt - data is persisted automatically
	return nil
}
