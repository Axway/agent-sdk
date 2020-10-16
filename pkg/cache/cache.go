package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	util "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util"
)

var globalCache Cache

// Cache - Interface for managing the proxy cache
type Cache interface {
	Get(key string) (interface{}, error)
	GetItem(key string) (*Item, error)
	GetBySecondaryKey(secondaryKey string) (interface{}, error)
	GetItemBySecondaryKey(secondaryKey string) (*Item, error)
	GetKeys() []string
	HasItemChanged(key string, data interface{}) (bool, error)
	HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error)
	Set(key string, data interface{}) error
	SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error
	SetSecondaryKey(key string, secondaryKey string) error
	Delete(key string) error
	DeleteBySecondaryKey(secondaryKey string) error
	DeleteSecondaryKey(secondaryKey string) error
	Flush()
	Save(path string) error
	Load(path string) error
}

// itemCache
type itemCache struct {
	Items       map[string]*Item  `json:"cache"`
	SecKeys     map[string]string `json:"secondaryKeys"`
	itemsLock   *sync.RWMutex     // Use lock when making changes/reading the items map
	secKeysLock *sync.RWMutex     // Use lock when making changes/reading the secKeys map
}

func init() {
	// Creates the global cache on first import of cache module
	SetCache(New())
}

// SetCache - sets the global cache
func SetCache(c Cache) {
	if c != nil {
		globalCache = c
	}
}

// GetCache - get the global cache object
func GetCache() Cache {
	return globalCache
}

// New - create a new cache object
func New() Cache {
	newCache := &itemCache{
		Items:       make(map[string]*Item),
		SecKeys:     make(map[string]string),
		itemsLock:   &sync.RWMutex{},
		secKeysLock: &sync.RWMutex{},
	}
	return newCache
}

// Load - create a new cache object and load saved data
func Load(path string) Cache {
	newCache := &itemCache{
		Items:       make(map[string]*Item),
		SecKeys:     make(map[string]string),
		itemsLock:   &sync.RWMutex{},
		secKeysLock: &sync.RWMutex{},
	}
	newCache.Load(path)
	return newCache
}

// check the current hash vs the newHash, return true if it has changed
func (c *itemCache) hasItemChanged(key string, data interface{}) (bool, error) {
	c.itemsLock.RLock()
	defer c.itemsLock.RUnlock()

	// Get the current item by key
	item, err := c.get(key)
	if err != nil {
		return true, err
	}

	// Get the hash of the new data
	newHash, err := util.ComputeHash(data)
	if err != nil {
		return false, err
	}

	// Check the hash
	if item.Hash == newHash {
		return false, nil
	}
	return true, nil
}

// returns the entire item, if found
func (c *itemCache) get(key string) (*Item, error) {
	c.itemsLock.RLock()
	defer c.itemsLock.RUnlock()

	if item, ok := c.Items[key]; ok {
		return item, nil
	}
	return nil, fmt.Errorf("Could not find item with key: %s", key)
}

// returns the primary key based on the secondary key
func (c *itemCache) findPrimaryKey(secondaryKey string) (string, error) {
	c.secKeysLock.RLock()
	defer c.secKeysLock.RUnlock()

	if key, ok := c.SecKeys[secondaryKey]; ok {
		return key, nil
	}

	return "", fmt.Errorf("Could not find secondary key: %s", secondaryKey)
}

// set the Item object to the key specified, updates the hash
func (c *itemCache) set(key string, data interface{}) error {
	c.itemsLock.Lock()
	defer c.itemsLock.Unlock()

	hash, err := util.ComputeHash(data)
	if err != nil {
		return err
	}

	c.Items[key] = &Item{
		Object:        data,
		UpdateTime:    time.Now().Unix(),
		Hash:          hash,
		SecondaryKeys: make(map[string]bool),
	}
	return nil
}

// set the secondaryKey for the key given
func (c *itemCache) setSecondaryKey(key string, secondaryKey string) error {
	c.secKeysLock.Lock()
	defer c.secKeysLock.Unlock()

	// check that the secondary key given is not used as primary
	if _, ok := c.Items[secondaryKey]; ok {
		return fmt.Errorf("Can't use %s as a secondary key, it is already a primary key", secondaryKey)
	}

	// check that the secondary key given is not already a secondary key
	if _, ok := c.SecKeys[secondaryKey]; ok {
		return fmt.Errorf("Can't use %s as a secondary key, it is already a secondary key", secondaryKey)
	}

	c.itemsLock.Lock()
	defer c.itemsLock.Unlock()

	item, ok := c.Items[key]
	// Check that the key given is in the cache
	if !ok {
		return fmt.Errorf("Can't set secondary key, %s, for a key, %s, as %s is not a known key", secondaryKey, key, key)
	}

	c.SecKeys[secondaryKey] = key
	item.SecondaryKeys[secondaryKey] = true
	return nil
}

// delete an item from the cache
func (c *itemCache) delete(key string) error {
	c.itemsLock.RLock()
	defer c.itemsLock.RUnlock()
	// Check that the key given is in the cache
	if _, ok := c.Items[key]; !ok {
		return fmt.Errorf("Cache item with key %s does not exist", key)
	}

	// Remove all secondary keys
	for secKey := range c.Items[key].SecondaryKeys {
		c.deleteSecondaryKey(secKey)
	}

	delete(c.Items, key)
	return nil
}

//deleteSecondaryKey - removes a secondary key reference in the cache
func (c *itemCache) deleteSecondaryKey(secondaryKey string) error {
	c.secKeysLock.Lock()
	defer c.secKeysLock.Unlock()

	// Check that the secondaryKey given is in the cache
	key, ok := c.SecKeys[secondaryKey]
	if !ok {
		return fmt.Errorf("Cache item with secondary key %s does not exist", key)
	}

	delete(c.Items[key].SecondaryKeys, secondaryKey)
	delete(c.SecKeys, secondaryKey)
	return nil
}

func (c *itemCache) flush() {
	c.secKeysLock.Lock()
	defer c.secKeysLock.Unlock()
	c.itemsLock.Lock()
	defer c.itemsLock.Unlock()

	c.SecKeys = make(map[string]string)
	c.Items = make(map[string]*Item)
}

func (c *itemCache) save(path string) error {
	c.secKeysLock.Lock()
	defer c.secKeysLock.Unlock()
	c.itemsLock.Lock()
	defer c.itemsLock.Unlock()

	file, err := os.Create(filepath.Clean(path))
	if err != nil {
		return err
	}

	cacheBytes, err := json.Marshal(c)
	if err != nil {
		file.Close()
		return err
	}
	_, err = io.Copy(file, bytes.NewReader(cacheBytes))
	file.Close()
	return err
}

func (c *itemCache) load(path string) error {
	c.secKeysLock.Lock()
	defer c.secKeysLock.Unlock()
	c.itemsLock.Lock()
	defer c.itemsLock.Unlock()

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}

	err = json.NewDecoder(file).Decode(c)
	file.Close()
	return err
}

// Get - return the object in the cache
func (c *itemCache) Get(key string) (interface{}, error) {
	item, err := c.get(key)
	if err != nil {
		return nil, err
	}

	return item.Object, nil
}

// GetItem - Return a pointer to the Item structure
func (c *itemCache) GetItem(key string) (*Item, error) {
	return c.get(key)
}

// GetBySecondaryKey - Using the secondary key return the object in the cache
func (c *itemCache) GetBySecondaryKey(secondaryKey string) (interface{}, error) {
	// Find the primary key
	key, err := c.findPrimaryKey(secondaryKey)
	if err != nil {
		return nil, err
	}

	item, err := c.get(key)
	if err != nil {
		return nil, err
	}

	return item.Object, nil
}

// GetItemBySecondaryKey - Using the secondary key return a pointer to the Item structure
func (c *itemCache) GetItemBySecondaryKey(secondaryKey string) (*Item, error) {
	// Find the primary key
	key, err := c.findPrimaryKey(secondaryKey)
	if err != nil {
		return nil, err
	}

	return c.get(key)
}

// GetKeys - Returns the keys in cache
func (c *itemCache) GetKeys() []string {
	keys := []string{}
	for key := range c.Items {
		keys = append(keys, key)
	}
	return keys
}

// HasItemChanged - Check if the item has changed
func (c *itemCache) HasItemChanged(key string, data interface{}) (bool, error) {
	return c.hasItemChanged(key, data)
}

// HasItemBySecondaryKeyChanged - Using the secondary key check if the item has changed
func (c *itemCache) HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error) {
	// Find the primary key
	key, err := c.findPrimaryKey(secondaryKey)
	if err != nil {
		return false, err
	}

	return c.hasItemChanged(key, data)
}

// Set - Create a new item, or update an existing item, in the cache with key
func (c *itemCache) Set(key string, data interface{}) error {
	return c.set(key, data)
}

// SetSecondaryKey - Create a new item in the cache with key and a secondaryKey reference
func (c *itemCache) SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error {
	err := c.set(key, data)
	if err != nil {
		return err
	}

	return c.setSecondaryKey(key, secondaryKey)
}

// SetSecondaryKey - Add the secondaryKey as a way to reference the item with key
func (c *itemCache) SetSecondaryKey(key string, secondaryKey string) error {
	return c.setSecondaryKey(key, secondaryKey)
}

// Delete - Remove the item which is found with this key
func (c *itemCache) Delete(key string) error {
	return c.delete(key)
}

// DeleteBySecondaryKey - Remove the item which is found with this secondary key
func (c *itemCache) DeleteBySecondaryKey(secondaryKey string) error {
	// Find the primary key
	key, err := c.findPrimaryKey(secondaryKey)
	if err != nil {
		return err
	}

	return c.delete(key)
}

// DeleteSecondaryKey - Remove the secondary key, preserve the item
func (c *itemCache) DeleteSecondaryKey(secondaryKey string) error {
	return c.deleteSecondaryKey(secondaryKey)
}

// Flush - Clears the entire cache
func (c *itemCache) Flush() {
	c.flush()
}

// Save - Save the data in this cache to file described by path
func (c *itemCache) Save(path string) error {
	return c.save(path)
}

// Load - Load the data from the file described by path to this cache
func (c *itemCache) Load(path string) error {
	return c.load(path)
}
