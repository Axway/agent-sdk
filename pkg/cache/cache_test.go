package cache

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalCache(t *testing.T) {
	// global cache should have been initialized on package import
	assert.NotNil(t, globalCache, "Expected the Global Cache to have been initialized")

	// GetCache returns the global cache
	assert.Equal(t, globalCache, GetCache(), "The GetCache method did not return the global cache")

	// Create a new cache, set it to the as the global cache, validate it was set
	newCache := New()
	newCache.Set("temp", "test data") // set an item to differ the newCache
	assert.NotEqual(t, newCache, GetCache(), "The newly created cache was set as the global, when it should not have been")
	assert.NotEqual(t, newCache, globalCache, "The newly created cache was set as the global, when it should not have been")
	SetCache(nil)
	assert.NotNil(t, globalCache, "The cache was set to nil, when it should not have been")
	SetCache(newCache)
	assert.Equal(t, newCache, GetCache(), "The newly created cache was not set as the global, when it should not have been")
	assert.Equal(t, newCache, globalCache, "The newly created cache was not set as the global, when it should not have been")
}

func TestSetItemsCache(t *testing.T) {
	// Create new cache
	cache := New()

	// Set an item, validate that it is in the cache
	key := "key"
	val := "check this"
	err := cache.Set(key, val)
	assert.Nil(t, err, "There was an unexpected error setting a cache item")
	cacheCfg := cache.(*itemCache)
	item, found := cacheCfg.Items[key]
	assert.True(t, found, "The cache item that was just set was not found")
	assert.Equal(t, val, item.GetObject().(string), "The value of the cached item was not what was expected")
	assert.NotNil(t, item.GetUpdateTime(), "The update time was nil")
	assert.NotNil(t, item.GetHash(), "The hash was nil")
	assert.Len(t, item.SecondaryKeys, 0, "The length of the secondary keys on item was more than 0")
	assert.Len(t, cacheCfg.SecKeys, 0, "The length of the global secondary keys was more than 0")
	err = cache.Set("error", map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "There was not an error when sending in a value that can't be marshaled")

	// Set the secondary key, validate that it is in the cache secKeys map and on the cache item
	secKey := "secTemp"
	secKey2 := "secTemp2"
	err = cache.SetSecondaryKey(key, secKey)
	assert.Nil(t, err, "There was an unexpected error setting a secondary key")
	assert.Len(t, item.SecondaryKeys, 1, "The length of the secondary keys on item was not 1")
	assert.Len(t, cacheCfg.SecKeys, 1, "The length of the global secondary keys was not 1")
	err = cache.SetSecondaryKey("bad", secKey2)
	assert.NotNil(t, err, "There was not an error when setting a secondary key with a bad primary key")
	assert.Len(t, item.SecondaryKeys, 1, "The length of the secondary keys changed and was not 1")
	assert.Len(t, cacheCfg.SecKeys, 1, "The length of the global secondary keys changed and was not 1")
	err = cache.SetSecondaryKey(key, key)
	assert.NotNil(t, err, "There was not an error when setting a secondary key with the same value as a primary key")
	err = cache.SetSecondaryKey(key, secKey)
	assert.NotNil(t, err, "There was not an error when setting a secondary key with an existing key")

	// Set an item with a secondary key, validate that it is in the cache
	cache = New() // Use new cache
	newKey := "newTemp"
	newSecKey := "newSecTemp"
	newVal := "new check this"
	cache.SetWithSecondaryKey(newKey, newSecKey, map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "There was not an error when sending in a value that can't be marshaled")
	cache.SetWithSecondaryKey(newKey, newSecKey, newVal)
	cacheCfg = cache.(*itemCache)
	item, found = cacheCfg.Items[newKey]
	assert.True(t, found, "The cache item that was just set was not found")
	assert.Equal(t, newVal, item.GetObject().(string), "The value of the cached item was not what was expected")
	assert.NotNil(t, item.GetUpdateTime(), "The update time was nil")
	assert.NotNil(t, item.GetHash(), "The hash was nil")
	assert.Len(t, item.SecondaryKeys, 1, "The length of the secondary keys on item was not 1")
	assert.Len(t, cacheCfg.SecKeys, 1, "The length of the global secondary keys was not 1")
}

func TestGetItemsCache(t *testing.T) {
	// Create new cache
	cache := New()

	// Add some cache items
	key1 := "key1"
	val1 := "key1 val1"
	cache.Set(key1, val1)
	key2 := "key2"
	key2sec := "key2sec"
	val2 := "key2 val2"
	cache.SetWithSecondaryKey(key2, key2sec, val2)

	// Get
	badKey := "bad"
	bad, err := cache.Get(badKey)
	assert.NotNil(t, err, "An error was expected from Get with a bad key")
	assert.Nil(t, bad, "A value was returned on a bad key")
	iVal, err := cache.Get(key1)
	assert.Nil(t, err, "There was an unexpected error getting key1")
	assert.Equal(t, val1, iVal.(string), "The stored value in cache was wrong")

	// GetItem
	bad, err = cache.GetItem(badKey)
	assert.NotNil(t, err, "An error was expected from GetItem with a bad key")
	assert.Nil(t, bad, "The Item returned was not nil")
	item, err := cache.GetItem(key1)
	assert.Nil(t, err, "There was an unexpected error getting key1")
	assert.Equal(t, val1, item.GetObject().(string), "The stored value in cache was wrong")

	// GetBySecondaryKey
	badSecondaryKey := "badSecondaryKey"
	bad, err = cache.GetBySecondaryKey(badSecondaryKey)
	assert.NotNil(t, err, "An error was expected from GetItem with a bad key")
	assert.Nil(t, bad, "The value returned by GetBySecondaryKey was not an nil")
	// add bad secondary key ref in map, should not happen in use
	badSecKeyToPrim := "secKeyNoPrim"
	cache.(*itemCache).SecKeys[badSecKeyToPrim] = badKey
	bad, err = cache.GetBySecondaryKey(badSecKeyToPrim)
	assert.NotNil(t, err, "An error was expected from GetBySecondaryKey with a bad key")
	assert.Nil(t, bad, "The value returned by GetBySecondaryKey was not an nil")
	iVal, err = cache.GetBySecondaryKey(key2sec)
	assert.Nil(t, err, "There was an unexpected error getting key2 with GetBySecondaryKey")
	assert.Equal(t, val2, iVal.(string), "The stored value in cache was wrong")

	// GetBySecondaryKey
	bad, err = cache.GetItemBySecondaryKey(badSecondaryKey)
	assert.NotNil(t, err, "An error was expected from GetItemBySecondaryKey with a bad key")
	assert.Nil(t, bad, "The Item returned was not nil")
	bad, err = cache.GetItemBySecondaryKey(badSecKeyToPrim)
	assert.NotNil(t, err, "An error was expected from GetItemBySecondaryKey with a bad key")
	assert.Nil(t, bad, "The Item returned by GetItemBySecondaryKey was not an nil")
	item, err = cache.GetItemBySecondaryKey(key2sec)
	assert.Nil(t, err, "There was an unexpected error getting key2 with GetItemBySecondaryKey")
	assert.Equal(t, val2, item.GetObject().(string), "The stored value in cache was wrong")
}

func TestHasItemChangedCache(t *testing.T) {
	// Create new cache
	cache := New()

	// Add some cache items
	key1 := "key1"
	val1 := "key1 val1"
	cache.Set(key1, val1)
	key2 := "key2"
	key2sec := "key2sec"
	val2 := "key2 val2"
	cache.SetWithSecondaryKey(key2, key2sec, val2)

	// HasItemChanged
	badKey := "bad"
	newVal1 := "key1 val1 2"
	changed, err := cache.HasItemChanged(badKey, newVal1)
	assert.NotNil(t, err, "An error was expected from HasItemChanged with a bad key")
	assert.True(t, changed, "Expected true since the item will not have been found in HasItemChanged")
	changed, err = cache.HasItemChanged(key1, map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "An error was expected from HasItemChanged with a value that can't be marshaled to json")
	assert.False(t, changed, "Expected false since HasItemChanged returned an error")
	changed, err = cache.HasItemChanged(key1, val1)
	assert.Nil(t, err, "There was an unexpected error when checking if the value of cached item, key1, has changed")
	assert.False(t, changed, "HasItemChanged did not return false as expected")
	changed, err = cache.HasItemChanged(key1, newVal1)
	assert.Nil(t, err, "There was an unexpected error when checking if the value of cached item, key1, has changed")
	assert.True(t, changed, "HasItemChanged did not return true as expected")

	// HasItemBySecondaryKeyChanged
	changed, err = cache.HasItemBySecondaryKeyChanged(badKey, newVal1)
	assert.NotNil(t, err, "An error was expected from HasItemBySecondaryKeyChanged with a bad key")
	assert.False(t, changed, "Expected false since HasItemBySecondaryKeyChanged returned an error")
	changed, err = cache.HasItemBySecondaryKeyChanged(key2sec, map[string]interface{}{"foo": make(chan int)})
	assert.NotNil(t, err, "An error was expected from HasItemBySecondaryKeyChanged with a value that can't be marshaled to json")
	assert.False(t, changed, "Expected false since HasItemBySecondaryKeyChanged returned an error")
	changed, err = cache.HasItemBySecondaryKeyChanged(key2sec, val2)
	assert.Nil(t, err, "There was an unexpected error when checking if the value of cached item, key2sec, has changed")
	assert.False(t, changed, "HasItemBySecondaryKeyChanged did not return false as expected")
	changed, err = cache.HasItemBySecondaryKeyChanged(key2sec, newVal1)
	assert.Nil(t, err, "There was an unexpected error when checking if the value of cached item, key2sec, has changed")
	assert.True(t, changed, "HasItemBySecondaryKeyChanged did not return true as expected")
}

func TestDeleteItemCache(t *testing.T) {
	// Create new cache
	cache := New()

	// Add some cache items
	key1 := "key1"
	val1 := "key1 val1"
	cache.Set(key1, val1)
	key2 := "key2"
	key2sec := "key2sec"
	val2 := "key2 val2"
	cache.SetWithSecondaryKey(key2, key2sec, val2)
	key3 := "key3"
	key3sec1 := "key3sec1"
	key3sec2 := "key3sec2"
	val3 := "key3 val3"
	cache.SetWithSecondaryKey(key3, key3sec1, val3)
	cache.SetSecondaryKey(key3, key3sec2)
	key4 := "key4"
	key4sec1 := "key4sec1"
	key4sec2 := "key4sec2"
	key4sec3 := "key4sec3"
	val4 := "key4 val4"
	cache.SetWithSecondaryKey(key4, key4sec1, val4)
	cache.SetSecondaryKey(key4, key4sec2)
	cache.SetSecondaryKey(key4, key4sec3)

	// Delete
	badKey := "bad"
	err := cache.Delete(badKey)
	assert.Len(t, cache.(*itemCache).Items, 4, "Expected 4 items in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 6, "Expected 6 items in secKeys")
	assert.NotNil(t, err, "An error was expected from Delete with a bad key")
	assert.Len(t, cache.(*itemCache).Items, 4, "Expected 4 items to still be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 6, "Expected 6 items to still be in secKeys")
	iVal2, err := cache.Get(key2)
	assert.Nil(t, err, "Expected no error, item has not been deleted")
	assert.Equal(t, val2, iVal2.(string), "Expected the correct value from val2")
	iVal2Sec, err := cache.GetBySecondaryKey(key2sec)
	assert.Nil(t, err, "Expected no error, item has not been deleted")
	assert.Equal(t, val2, iVal2Sec.(string), "Expected the correct value from val2 secondary key")
	err = cache.Delete(key2)
	assert.Nil(t, err, "An error was not expected from Delete with key2")
	assert.Len(t, cache.(*itemCache).Items, 3, "Expected 3 items to be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 5, "Expected 5 items to still be in secKeys")

	// DeleteBySecondaryKey
	badSecKey := "badSecKey"
	assert.Len(t, cache.(*itemCache).Items, 3, "Expected 3 items in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 5, "Expected 5 items in secKeys")
	err = cache.DeleteBySecondaryKey(badSecKey)
	assert.NotNil(t, err, "An error was expected from DeleteBySecondaryKey with a bad key")
	assert.Len(t, cache.(*itemCache).Items, 3, "Expected 3 items to still be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 5, "Expected 5 items to still be in secKeys")
	err = cache.DeleteBySecondaryKey(key3sec2)
	assert.Nil(t, err, "An error was not expected from DeleteBySecondaryKey with key3sec2")
	assert.Len(t, cache.(*itemCache).Items, 2, "Expected 2 items to be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 3, "Expected 3 items to still be in secKeys")

	// DeleteSecondaryKey
	assert.Len(t, cache.(*itemCache).Items, 2, "Expected 2 items in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 3, "Expected 3 items in secKeys")
	err = cache.DeleteSecondaryKey(badSecKey)
	assert.NotNil(t, err, "An error was expected from DeleteSecondaryKey with a bad key")
	assert.Len(t, cache.(*itemCache).Items, 2, "Expected 2 items to still be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 3, "Expected 3 items to still be in secKeys")
	err = cache.DeleteSecondaryKey(key4sec2)
	assert.Nil(t, err, "An error was not expected from DeleteSecondaryKey with key4sec2")
	assert.Len(t, cache.(*itemCache).Items, 2, "Expected 2 items to be in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 2, "Expected 2 items to still be in secKeys")

	// Flush
	assert.Len(t, cache.(*itemCache).Items, 2, "Expected 2 items in the cache")
	assert.Len(t, cache.(*itemCache).SecKeys, 2, "Expected 2 items in secKeys")
	cache.Flush()
	assert.Len(t, cache.(*itemCache).Items, 0, "Expected cache to be empty")
	assert.Len(t, cache.(*itemCache).SecKeys, 0, "Expected secKeys to be empty")
}

func TestSaveAndLoad(t *testing.T) {
	// Create new cache
	cache := New()

	// Add some cache items
	key1 := "key1"
	val1 := "key1 val1"
	cache.Set(key1, val1)
	key2 := "key2"
	key2sec := "key2sec"
	val2 := "key2 val2"
	cache.SetWithSecondaryKey(key2, key2sec, val2)
	key3 := "key3"
	key3sec1 := "key3sec1"
	key3sec2 := "key3sec2"
	val3 := "key3 val3"
	cache.SetWithSecondaryKey(key3, key3sec1, val3)
	cache.SetSecondaryKey(key3, key3sec2)
	key4 := "key4"
	key4sec1 := "key4sec1"
	key4sec2 := "key4sec2"
	key4sec3 := "key4sec3"
	val4 := "key4 val4"
	cache.SetWithSecondaryKey(key4, key4sec1, val4)
	cache.SetSecondaryKey(key4, key4sec2)
	cache.SetSecondaryKey(key4, key4sec3)
	cacheFile := "cache_save_file.json"

	// Remove file if it exists
	_, err := os.Stat(cacheFile)
	if os.IsExist(err) {
		os.Remove(cacheFile)
	}

	// Save
	err = cache.Save(cacheFile)
	assert.Nil(t, err, "An unexpected error was returned by the Save cache method")

	// Load
	cache2 := Load(cacheFile) // Create a new cache object to load
	assert.Nil(t, err, "An unexpected error was returned by the Load cache method")
	assert.Equal(t, cache, cache2, "The loaded cache was not the same as the cache saved")
	assert.Equal(t, cache.(*itemCache).Items[key1].Object, cache2.(*itemCache).Items[key1].Object, "The loaded key1 value was not the same")
	assert.Equal(t, cache.(*itemCache).Items[key2].Object, cache2.(*itemCache).Items[key2].Object, "The loaded key2 value was not the same")
	assert.Equal(t, cache.(*itemCache).Items[key3].Object, cache2.(*itemCache).Items[key3].Object, "The loaded key3 value was not the same")
	assert.Equal(t, cache.(*itemCache).Items[key4].Object, cache2.(*itemCache).Items[key4].Object, "The loaded key4 value was not the same")
	assert.Equal(t, cache.(*itemCache).SecKeys, cache2.(*itemCache).SecKeys, "The secondary keys were not loaded properly")
	assert.Equal(t, cache.(*itemCache).Items[key1].SecondaryKeys, cache2.(*itemCache).Items[key1].SecondaryKeys, "The secondary keys for 'key1' were not loaded properly")
	assert.Equal(t, cache.(*itemCache).Items[key2].SecondaryKeys, cache2.(*itemCache).Items[key2].SecondaryKeys, "The secondary keys for 'key2' were not loaded properly")
	assert.Equal(t, cache.(*itemCache).Items[key3].SecondaryKeys, cache2.(*itemCache).Items[key3].SecondaryKeys, "The secondary keys for 'key3' were not loaded properly")
	assert.Equal(t, cache.(*itemCache).Items[key4].SecondaryKeys, cache2.(*itemCache).Items[key4].SecondaryKeys, "The secondary keys for 'key4' were not loaded properly")

	// Change original cache, make sure loaded has not changed
	cache.Set(key1, "key1 val2")
	assert.NotEqual(t, cache.(*itemCache).Items[key1].Object, cache2.(*itemCache).Items[key1].Object, "Updating the oringal cache seemed to have changed the loaded cache")
}
