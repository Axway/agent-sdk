package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
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
	GetForeignKeys() []string
	GetItemsByForeignKey(foreignKey string) ([]*Item, error)
	GetKeys() []string
	HasItemChanged(key string, data interface{}) (bool, error)
	HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error)
	Set(key string, data interface{}) error
	SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error
	SetWithForeignKey(key string, foreignKey string, data interface{}) error
	SetSecondaryKey(key string, secondaryKey string) error
	SetForeignKey(key string, foreignKey string) error
	Delete(key string) error
	DeleteBySecondaryKey(secondaryKey string) error
	DeleteSecondaryKey(secondaryKey string) error
	DeleteForeignKey(foreignKey string) error
	DeleteItemsByForeignKey(foreignKey string) error
	Flush()
	Save(path string) error
	Load(path string) error
}

// GetItem interface for getting a single item from a cache.
type GetItem interface {
	Get(key string) (interface{}, error)
}

type action int

const (
	getAction action = iota
	setAction
	deleteAction
	findAction
	hasChangedAction
	setSecKeyAction
	setForeignKeyAction
	deleteSecKeyAction
	deleteForeignKeyAction
	flushAction
	loadAction
	getKeysAction
	getForeignKeysAction
	getItemsByForeignKeyAction
	marshalAction
)

type cacheAction struct {
	action action
	key    string
	secKey string
	forKey string
	data   interface{}
	path   string
}

type cacheReply struct {
	item    *Item
	key     string
	err     error
	changed bool
	keys    []string
	items   []*Item
	data    []byte
}

// itemCache
type itemCache struct {
	Items         map[string]*Item  `json:"cache"`
	SecKeys       map[string]string `json:"secondaryKeys"`
	startedMutex  *sync.Mutex
	saveMutex     *sync.Mutex
	started       bool
	actionChannel chan cacheAction
	replyChannel  chan cacheReply
}

func (c *itemCache) MarshalJSON() ([]byte, error) {
	marshalReply := c.runAction(cacheAction{
		action: marshalAction,
	})
	if marshalReply.err != nil {
		return nil, marshalReply.err
	}

	return marshalReply.data, nil
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
		startedMutex:  &sync.Mutex{},
		saveMutex:     &sync.Mutex{},
		actionChannel: make(chan cacheAction),
		replyChannel:  make(chan cacheReply),
	}
	go newCache.handleAction()
	return newCache
}

// Load - create a new cache object and load saved data
func Load(path string) Cache {
	newCache := &itemCache{
		Items:         make(map[string]*Item),
		SecKeys:       make(map[string]string),
		startedMutex:  &sync.Mutex{},
		saveMutex:     &sync.Mutex{},
		actionChannel: make(chan cacheAction),
		replyChannel:  make(chan cacheReply),
	}
	go newCache.handleAction()
	newCache.Load(path)
	return newCache
}

// LoadFromBuffer - create a new cache object and loads the data from buffer
func LoadFromBuffer(buffer []byte) Cache {
	newCache := &itemCache{
		Items:        make(map[string]*Item),
		SecKeys:      make(map[string]string),
		startedMutex: &sync.Mutex{},
		saveMutex:    &sync.Mutex{},
	}
	json.Unmarshal(buffer, &newCache)

	newCache.actionChannel = make(chan cacheAction)
	newCache.replyChannel = make(chan cacheReply)
	go newCache.handleAction()
	return newCache
}

func (c *itemCache) isStarted() bool {
	c.startedMutex.Lock()
	defer c.startedMutex.Unlock()
	return c.started
}

func (c *itemCache) updateIsStarted(val bool) {
	c.startedMutex.Lock()
	defer c.startedMutex.Unlock()
	c.started = val
}

// handleAction - handles all calls to the cache to prevent locking issues
func (c *itemCache) handleAction() {
	// make sure only one handleAction loop is running
	if c.isStarted() {
		return
	}
	c.updateIsStarted(true)
	defer c.updateIsStarted(false)

	actionMap := map[action]func(cacheAction) cacheReply{
		getAction:                  c.get,
		getKeysAction:              c.getKeys,
		getForeignKeysAction:       c.getForeignKeys,
		setAction:                  c.set,
		deleteAction:               c.delete,
		findAction:                 c.findPrimaryKey,
		hasChangedAction:           c.hasItemChanged,
		setSecKeyAction:            c.setSecondaryKey,
		setForeignKeyAction:        c.setForeignKey,
		getItemsByForeignKeyAction: c.getItemsByForeignKeys,
		deleteSecKeyAction:         c.deleteSecondaryKey,
		deleteForeignKeyAction:     c.deleteForeignKey,
		flushAction:                c.flush,
		marshalAction:              c.marshal,
		loadAction:                 c.load,
	}

	for {
		thisAction := <-c.actionChannel
		reply := actionMap[thisAction.action](thisAction)
		c.replyChannel <- reply
	}
}

func (c *itemCache) marshal(thisAction cacheAction) (thisReply cacheReply) {
	thisReply = cacheReply{
		err: nil,
	}

	itemBytes, err := json.Marshal(c.Items)
	if err != nil {
		thisReply.err = err
		return
	}

	secKeysBytes, err := json.Marshal(c.SecKeys)
	if err != nil {
		thisReply.err = err
		return
	}

	type alias struct {
		Items   json.RawMessage `json:"cache"`
		SecKeys json.RawMessage `json:"secondaryKeys"`
	}

	a := &alias{
		Items:   json.RawMessage(itemBytes),
		SecKeys: json.RawMessage(secKeysBytes),
	}

	cachedBytes, err := json.Marshal(a)
	if err != nil {
		thisReply.err = err
		return
	}
	thisReply.data = cachedBytes
	return
}

// check the current hash vs the newHash, return true if it has changed
func (c *itemCache) hasItemChanged(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key
	data := thisAction.data

	thisReply = cacheReply{
		changed: true,
		err:     nil,
	}

	// Get the current item by key=
	item, ok := c.Items[key]
	if !ok {
		thisReply.err = fmt.Errorf("could not find item with key: %s", key)
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
	}
	return
}

// returns the entire item, if found
func (c *itemCache) get(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key

	thisReply = cacheReply{
		item: nil,
		err:  fmt.Errorf("could not find item with key: %s", key),
	}
	if item, ok := c.Items[key]; ok {
		replyItem := &Item{
			UpdateTime:    item.UpdateTime,
			Hash:          item.Hash,
			SecondaryKeys: item.SecondaryKeys,
			ForeignKey:    item.ForeignKey,
			Object:        item.Object,
		}
		if item.Object != nil && item.ContainsPointer && reflect.ValueOf(item.Object).Type().Kind() == reflect.Ptr {
			pOriginal := reflect.ValueOf(item.Object).Elem().Interface()
			rf := reflect.ValueOf(pOriginal)
			p := reflect.New(rf.Type())
			p.Elem().Set(rf)
			replyItem.Object = p.Interface()
		}

		thisReply = cacheReply{
			item: replyItem,
			err:  nil,
		}
	}
	return
}

// getKeys - Returns the keys in cache
func (c *itemCache) getKeys(thisAction cacheAction) (thisReply cacheReply) {
	keys := []string{}
	for key := range c.Items {
		keys = append(keys, key)
	}
	thisReply = cacheReply{
		keys: keys,
		err:  nil,
	}
	return
}

// getForeignKeys - Returns the Foreign keys in cache
func (c *itemCache) getForeignKeys(thisAction cacheAction) (thisReply cacheReply) {
	keys := []string{}
	for key := range c.Items {
		if c.Items[key].ForeignKey != "" {
			keys = append(keys, c.Items[key].ForeignKey)
		}
	}

	thisReply = cacheReply{
		keys: keys,
		err:  nil,
	}
	return
}

// getItemsByForeignKeys - Returns the Items with a particular Foreign key in cache
func (c *itemCache) getItemsByForeignKeys(thisAction cacheAction) (thisReply cacheReply) {
	var keys []string
	var items []*Item

	for key, item := range c.Items {
		if item.ForeignKey == thisAction.forKey {
			keys = append(keys, key)
			items = append(items, item)
		}
	}

	thisReply = cacheReply{
		items: items,
		keys:  keys,
		err:   nil,
	}
	return
}

// returns the primary key based on the secondary key
func (c *itemCache) findPrimaryKey(thisAction cacheAction) (thisReply cacheReply) {
	secondaryKey := thisAction.secKey

	thisReply = cacheReply{
		key: "",
		err: fmt.Errorf("could not find secondary key: %s", secondaryKey),
	}

	if key, ok := c.SecKeys[secondaryKey]; ok {
		thisReply = cacheReply{
			key: key,
			err: nil,
		}
	}
	return
}

// set the Item object to the key specified, updates the hash
func (c *itemCache) set(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key
	data := thisAction.data

	thisReply = cacheReply{
		err: nil,
	}

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

	if data != nil && reflect.ValueOf(data).Type().Kind() == reflect.Ptr {
		c.Items[key].ContainsPointer = true
	}
	return
}

// set the secondaryKey for the key given
func (c *itemCache) setSecondaryKey(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key
	secondaryKey := thisAction.secKey

	thisReply = cacheReply{
		err: nil,
	}

	// check that the secondary key given is not used as primary
	if _, ok := c.Items[secondaryKey]; ok {
		thisReply.err = fmt.Errorf("can't use %s as a secondary key, it is already a primary key", secondaryKey)
		return
	}

	item, ok := c.Items[key]
	// Check that the key given is in the cache
	if !ok {
		thisReply.err = fmt.Errorf("can't set secondary key, %s, for a key, %s, as %s is not a known key", secondaryKey, key, key)
		return
	}

	c.SecKeys[secondaryKey] = key
	item.SecondaryKeys[secondaryKey] = true
	return
}

// set the ForeignKey for the key given
func (c *itemCache) setForeignKey(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key
	foreignKey := thisAction.forKey

	thisReply = cacheReply{
		err: nil,
	}

	item, ok := c.Items[key]
	// Check that the key given is in the cache
	if !ok {
		thisReply.err = fmt.Errorf("can't set foreign key, %s, for a key, %s, as %s is not a known key", foreignKey, key, key)
		return
	}

	// check that the foreign key given is not already a foreign key
	if foreignKey == item.ForeignKey {
		thisReply.err = fmt.Errorf("can't use %s as a foreign key, it is already a foreign key for the item", foreignKey)
		return
	}

	item.ForeignKey = foreignKey
	return
}

// delete an item from the cache
func (c *itemCache) delete(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key

	thisReply = cacheReply{
		err: nil,
	}

	// Check that the key given is in the cache
	if _, ok := c.Items[key]; !ok {
		thisReply.err = fmt.Errorf("cache item with key %s does not exist", key)
		return
	}

	// Remove all secondary keys
	for secKey := range c.Items[key].SecondaryKeys {
		c.removeSecondaryKey(secKey)
	}

	delete(c.Items, key)
	return
}

// deleteSecondaryKey - removes a secondary key reference in the cache, but locks the items before doing so
func (c *itemCache) deleteSecondaryKey(thisAction cacheAction) (thisReply cacheReply) {
	secondaryKey := thisAction.secKey

	thisReply = cacheReply{
		err: c.removeSecondaryKey(secondaryKey),
	}
	return
}

// removeSecondaryKey - removes a secondary key reference in the cache
func (c *itemCache) removeSecondaryKey(secondaryKey string) error {
	// Check that the secondaryKey given is in the cache
	key, ok := c.SecKeys[secondaryKey]
	if !ok {
		return fmt.Errorf("cache item with secondary key %s does not exist", key)
	}

	delete(c.Items[key].SecondaryKeys, secondaryKey)
	delete(c.SecKeys, secondaryKey)
	return nil
}

// deleteForeignKey - removes a foreign key reference in the cache, but locks the items before doing so
func (c *itemCache) deleteForeignKey(thisAction cacheAction) (thisReply cacheReply) {
	key := thisAction.key

	item, ok := c.Items[key]
	// Check that the key given is in the cache
	if !ok {
		thisReply.err = fmt.Errorf("cache item with key %s does not exist", key)
		return
	}

	item.ForeignKey = ""
	return
}

func (c *itemCache) flush(thisAction cacheAction) (thisReply cacheReply) {
	thisReply = cacheReply{}

	c.SecKeys = make(map[string]string)
	c.Items = make(map[string]*Item)
	return
}

func (c *itemCache) load(thisAction cacheAction) (thisReply cacheReply) {
	path := thisAction.path

	thisReply = cacheReply{
		err: nil,
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		thisReply.err = err
		return
	}

	thisReply.err = json.NewDecoder(file).Decode(c)
	file.Close()
	return
}

func (c *itemCache) runAction(thisAction cacheAction) cacheReply {
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

// GetItemsByForeignKey - Using the foreign key return an array of pointers to the Items which have that particular foreign key
func (c *itemCache) GetItemsByForeignKey(foreignKey string) ([]*Item, error) {

	getItemsForeignKeyReply := c.runAction(cacheAction{
		action: getItemsByForeignKeyAction,
		forKey: foreignKey,
	})

	return getItemsForeignKeyReply.items, getItemsForeignKeyReply.err
}

// GetKeys - Returns the keys in cache
func (c *itemCache) GetKeys() []string {
	getKeysReply := c.runAction(cacheAction{
		action: getKeysAction,
	})
	if getKeysReply.err != nil {
		return []string{}
	}

	return getKeysReply.keys
}

// GetForeignKeys - Returns the Foreign keys in cache
func (c *itemCache) GetForeignKeys() []string {
	getKeysReply := c.runAction(cacheAction{
		action: getForeignKeysAction,
	})
	if getKeysReply.err != nil {
		return []string{}
	}

	return getKeysReply.keys
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

// SetWithForeignKey - Create a new item in the cache with key and a ForeignKey reference
func (c *itemCache) SetWithForeignKey(key string, foreignKey string, data interface{}) error {
	err := c.Set(key, data)
	if err != nil {
		return err
	}

	return c.SetForeignKey(key, foreignKey)
}

// SetForeignKey - Add the ForeignKey as a way to reference the item with key
func (c *itemCache) SetForeignKey(key string, foreignKey string) error {
	setForeignKeyReply := c.runAction(cacheAction{
		action: setForeignKeyAction,
		key:    key,
		forKey: foreignKey,
	})
	return setForeignKeyReply.err
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

// DeleteItemsByForeignKey - Remove all the items which is found with this foreign key
func (c *itemCache) DeleteItemsByForeignKey(foreignKey string) error {
	getItemsForeignKeyReply := c.runAction(cacheAction{
		action: getItemsByForeignKeyAction,
		forKey: foreignKey,
	})
	if len(getItemsForeignKeyReply.keys) == 0 {
		return fmt.Errorf("no items found with foreign key: %s", foreignKey)
	}

	var lastErr error
	for _, key := range getItemsForeignKeyReply.keys {
		deleteReply := c.runAction(cacheAction{
			action: deleteAction,
			key:    key,
		})

		if deleteReply.err != nil {
			lastErr = deleteReply.err
		}
	}

	return lastErr

}

// DeleteSecondaryKey - Remove the secondary key, preserve the item
func (c *itemCache) DeleteSecondaryKey(secondaryKey string) error {
	deleteSecKeyReply := c.runAction(cacheAction{
		action: deleteSecKeyAction,
		secKey: secondaryKey,
	})
	return deleteSecKeyReply.err
}

// DeleteForeignKey - Remove the foreign key, preserve the item
func (c *itemCache) DeleteForeignKey(key string) error {
	deleteForeignKeyReply := c.runAction(cacheAction{
		action: deleteForeignKeyAction,
		key:    key,
	})
	return deleteForeignKeyReply.err
}

// Flush - Clears the entire cache
func (c *itemCache) Flush() {
	c.runAction(cacheAction{
		action: flushAction,
	})
}

// Save - Save the data in this cache to file described by path
func (c *itemCache) Save(path string) error {
	c.saveMutex.Lock()
	defer c.saveMutex.Unlock()

	file, err := os.Create(filepath.Clean(path))
	defer func() {
		file.Close()
	}()

	if err != nil {
		return err
	}

	cacheBytes, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, bytes.NewReader(cacheBytes))
	return err
}

// Load - Load the data from the file described by path to this cache
func (c *itemCache) Load(path string) error {
	loadReply := c.runAction(cacheAction{
		action: loadAction,
		path:   path,
	})
	return loadReply.err
}
