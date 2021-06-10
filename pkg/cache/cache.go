package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	util "github.com/Axway/agent-sdk/pkg/util"
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

type action int

const (
	getAction action = iota
	setAction
	deleteAction
	findAction
	hasChangedAction
	setSecKeyAction
	deleteSecKeyAction
	flushAction
	saveAction
	loadAction
)

type cacheAction struct {
	action action
	key    string
	secKey string
	data   interface{}
	path   string
}

type reply struct {
	item    *Item
	key     string
	err     error
	changed bool
}

// itemCache
type itemCache struct {
	Items         map[string]*Item  `json:"cache"`
	SecKeys       map[string]string `json:"secondaryKeys"`
	actionChannel chan cacheAction
	replyChannel  chan reply
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
		Items:         make(map[string]*Item),
		SecKeys:       make(map[string]string),
		actionChannel: make(chan cacheAction),
		replyChannel:  make(chan reply),
	}
	go newCache.handleAction()
	return newCache
}

// Load - create a new cache object and load saved data
func Load(path string) Cache {
	newCache := &itemCache{
		Items:         make(map[string]*Item),
		SecKeys:       make(map[string]string),
		actionChannel: make(chan cacheAction),
		replyChannel:  make(chan reply),
	}
	go newCache.handleAction()
	newCache.Load(path)
	return newCache
}

// handleAction - handles all calls to the cache to prevent locking issues
func (c *itemCache) handleAction() {
	for {
		thisAction := <-c.actionChannel
		switch thisAction.action {
		case getAction:
			c.get(thisAction.key)
		case hasChangedAction:
			c.hasItemChanged(thisAction.key, thisAction.data)
		case findAction:
			c.findPrimaryKey(thisAction.secKey)
		case setAction:
			c.set(thisAction.key, thisAction.data)
		case setSecKeyAction:
			c.setSecondaryKey(thisAction.key, thisAction.secKey)
		case deleteAction:
			c.delete(thisAction.key)
		case deleteSecKeyAction:
			c.deleteSecondaryKey(thisAction.secKey)
		case flushAction:
			c.flush()
		case saveAction:
			c.save(thisAction.path)
		case loadAction:
			c.load(thisAction.path)
		}
	}
}

// check the current hash vs the newHash, return true if it has changed
func (c *itemCache) hasItemChanged(key string, data interface{}) {
	thisReply := reply{
		changed: true,
		err:     nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	// Get the current item by key=
	item, ok := c.Items[key]
	if !ok {
		thisReply.err = fmt.Errorf("Could not find item with key: %s", key)
		return
	}

	// Get the hash of the new data
	newHash, err := util.ComputeHash(data)
	if err != nil {
		thisReply.changed = false
		thisReply.err = err
		return
	}

	// Check the hash
	if item.Hash == newHash {
		thisReply.changed = false
		thisReply.err = nil
		return
	}
}

// returns the entire item, if found
func (c *itemCache) get(key string) {
	thisReply := reply{
		item: nil,
		err:  fmt.Errorf("Could not find item with key: %s", key),
	}
	if item, ok := c.Items[key]; ok {
		thisReply = reply{
			item: item,
			err:  nil,
		}
	}
	c.replyChannel <- thisReply
}

// returns the primary key based on the secondary key
func (c *itemCache) findPrimaryKey(secondaryKey string) {
	thisReply := reply{
		key: "",
		err: fmt.Errorf("Could not find secondary key: %s", secondaryKey),
	}
	if key, ok := c.SecKeys[secondaryKey]; ok {
		thisReply = reply{
			key: key,
			err: nil,
		}
	}
	c.replyChannel <- thisReply
}

// set the Item object to the key specified, updates the hash
func (c *itemCache) set(key string, data interface{}) {
	thisReply := reply{
		err: nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	hash, err := util.ComputeHash(data)
	if err != nil {
		thisReply.err = err
		return
	}

	secKeys := make(map[string]bool)
	if _, ok := c.Items[key]; ok {
		secKeys = c.Items[key].SecondaryKeys
	}
	c.Items[key] = &Item{
		Object:        data,
		UpdateTime:    time.Now().Unix(),
		Hash:          hash,
		SecondaryKeys: secKeys,
	}
	return
}

// set the secondaryKey for the key given
func (c *itemCache) setSecondaryKey(key string, secondaryKey string) {
	thisReply := reply{
		err: nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	// check that the secondary key given is not used as primary
	if _, ok := c.Items[secondaryKey]; ok {
		thisReply.err = fmt.Errorf("Can't use %s as a secondary key, it is already a primary key", secondaryKey)
		return
	}

	// check that the secondary key given is not already a secondary key
	if _, ok := c.SecKeys[secondaryKey]; ok {
		thisReply.err = fmt.Errorf("Can't use %s as a secondary key, it is already a secondary key", secondaryKey)
		return
	}

	item, ok := c.Items[key]
	// Check that the key given is in the cache
	if !ok {
		thisReply.err = fmt.Errorf("Can't set secondary key, %s, for a key, %s, as %s is not a known key", secondaryKey, key, key)
		return
	}

	c.SecKeys[secondaryKey] = key
	item.SecondaryKeys[secondaryKey] = true
}

// delete an item from the cache
func (c *itemCache) delete(key string) {
	thisReply := reply{
		err: nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	// Check that the key given is in the cache
	if _, ok := c.Items[key]; !ok {
		thisReply.err = fmt.Errorf("Cache item with key %s does not exist", key)
		return
	}

	// Remove all secondary keys
	for secKey := range c.Items[key].SecondaryKeys {
		c.removeSecondaryKey(secKey)
	}

	delete(c.Items, key)
}

//deleteSecondaryKey - removes a secondary key reference in the cache, but locks the items before doing so
func (c *itemCache) deleteSecondaryKey(secondaryKey string) {
	thisReply := reply{
		err: c.removeSecondaryKey(secondaryKey),
	}
	c.replyChannel <- thisReply
}

//removeSecondaryKey - removes a secondary key reference in the cache
func (c *itemCache) removeSecondaryKey(secondaryKey string) error {
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
	defer func() {
		c.replyChannel <- reply{}
	}()

	c.SecKeys = make(map[string]string)
	c.Items = make(map[string]*Item)
}

func (c *itemCache) save(path string) {
	thisReply := reply{
		err: nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	file, err := os.Create(filepath.Clean(path))
	if err != nil {
		thisReply.err = err
		return
	}

	cacheBytes, err := json.Marshal(c)
	if err != nil {
		file.Close()
		thisReply.err = err
		return
	}
	_, err = io.Copy(file, bytes.NewReader(cacheBytes))
	file.Close()
	thisReply.err = err
	return
}

func (c *itemCache) load(path string) {
	thisReply := reply{
		err: nil,
	}
	defer func() {
		c.replyChannel <- thisReply
	}()

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		thisReply.err = err
		return
	}

	thisReply.err = json.NewDecoder(file).Decode(c)
	file.Close()
	return
}

func (c *itemCache) runAction(thisAction cacheAction) reply {
	c.actionChannel <- thisAction
	thisReply := <-c.replyChannel
	return thisReply
}

func (c *itemCache) runFindPrimaryKey(secondaryKey string) (string, error) {
	findReply := c.runAction(cacheAction{
		action: findAction,
		secKey: secondaryKey,
	})
	if findReply.err != nil {
		return "", findReply.err
	}
	return findReply.key, nil
}

// Get - return the object in the cache
func (c *itemCache) Get(key string) (interface{}, error) {
	item, err := c.GetItem(key)
	if item != nil {
		return item.Object, nil
	}
	return nil, err
}

// GetItem - Return a pointer to the Item structure
func (c *itemCache) GetItem(key string) (*Item, error) {
	getReply := c.runAction(cacheAction{
		action: getAction,
		key:    key,
	})
	if getReply.err != nil {
		return nil, getReply.err
	}

	return getReply.item, nil
}

// GetBySecondaryKey - Using the secondary key return the object in the cache
func (c *itemCache) GetBySecondaryKey(secondaryKey string) (interface{}, error) {
	item, err := c.GetItemBySecondaryKey(secondaryKey)
	if item != nil {
		return item.Object, nil
	}
	return nil, err
}

// GetItemBySecondaryKey - Using the secondary key return a pointer to the Item structure
func (c *itemCache) GetItemBySecondaryKey(secondaryKey string) (*Item, error) {
	// Find the primary key
	key, err := c.runFindPrimaryKey(secondaryKey)
	if err != nil {
		return nil, err
	}
	getReply := c.runAction(cacheAction{
		action: getAction,
		key:    key,
	})

	return getReply.item, getReply.err
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
	changedReply := c.runAction(cacheAction{
		action: hasChangedAction,
		key:    key,
		data:   data,
	})
	return changedReply.changed, changedReply.err
}

// HasItemBySecondaryKeyChanged - Using the secondary key check if the item has changed
func (c *itemCache) HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error) {
	// Find the primary key
	key, err := c.runFindPrimaryKey(secondaryKey)
	if err != nil {
		return false, err
	}

	return c.HasItemChanged(key, data)
}

// Set - Create a new item, or update an existing item, in the cache with key
func (c *itemCache) Set(key string, data interface{}) error {
	// Find the primary key
	setReply := c.runAction(cacheAction{
		action: setAction,
		key:    key,
		data:   data,
	})
	return setReply.err
}

// SetSecondaryKey - Create a new item in the cache with key and a secondaryKey reference
func (c *itemCache) SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error {
	err := c.Set(key, data)
	if err != nil {
		return err
	}

	return c.SetSecondaryKey(key, secondaryKey)
}

// SetSecondaryKey - Add the secondaryKey as a way to reference the item with key
func (c *itemCache) SetSecondaryKey(key string, secondaryKey string) error {
	setSecKeyReply := c.runAction(cacheAction{
		action: setSecKeyAction,
		key:    key,
		secKey: secondaryKey,
	})
	return setSecKeyReply.err
}

// Delete - Remove the item which is found with this key
func (c *itemCache) Delete(key string) error {
	deleteReply := c.runAction(cacheAction{
		action: deleteAction,
		key:    key,
	})
	return deleteReply.err
}

// DeleteBySecondaryKey - Remove the item which is found with this secondary key
func (c *itemCache) DeleteBySecondaryKey(secondaryKey string) error {
	// Find the primary key
	key, err := c.runFindPrimaryKey(secondaryKey)
	if err != nil {
		return err
	}

	return c.Delete(key)
}

// DeleteSecondaryKey - Remove the secondary key, preserve the item
func (c *itemCache) DeleteSecondaryKey(secondaryKey string) error {
	deleteSecKeyReply := c.runAction(cacheAction{
		action: deleteSecKeyAction,
		secKey: secondaryKey,
	})
	return deleteSecKeyReply.err
}

// Flush - Clears the entire cache
func (c *itemCache) Flush() {
	c.runAction(cacheAction{
		action: flushAction,
	})
}

// Save - Save the data in this cache to file described by path
func (c *itemCache) Save(path string) error {
	saveReply := c.runAction(cacheAction{
		action: saveAction,
		path:   path,
	})
	return saveReply.err
}

// Load - Load the data from the file described by path to this cache
func (c *itemCache) Load(path string) error {
	loadReply := c.runAction(cacheAction{
		action: loadAction,
		path:   path,
	})
	return loadReply.err
}
