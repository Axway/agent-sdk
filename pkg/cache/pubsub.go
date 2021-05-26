package cache

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/notification"
	util "github.com/Axway/agent-sdk/pkg/util"
)

var topics map[string]*cachePubSub

type cachePubSub struct {
	notification.PubSub
	topic    string
	notifier notification.Notifier
	channel  chan interface{}
}

func init() {
	topics = make(map[string]*cachePubSub)
}

// CreateTopic - create a new PubSub for a cache item
func CreateTopic(topic string) (notification.PubSub, error) {
	return CreateTopicWithInitData(topic, nil)
}

// CreateTopicWithInitData - create a new PubSub with an item that will be cached, initialize it as well
func CreateTopicWithInitData(topic string, initData interface{}) (notification.PubSub, error) {
	_, err := globalCache.Get(topic)
	if err == nil {
		return nil, fmt.Errorf("Could not create a PubSub topic, name in cache already used")
	}
	globalCache.Set(topic, initData)
	channel := make(chan interface{})
	notifier, err := notification.RegisterNotifier(topic, channel)
	if err != nil {
		return nil, fmt.Errorf("Could not create a PubSub: %s", err.Error())
	}
	notifier.Start() // Start the notifier to listen for data on the channel
	newCachePubSub := &cachePubSub{
		topic:    topic,
		notifier: notifier,
		channel:  channel,
	}
	topics[topic] = newCachePubSub
	return newCachePubSub, nil
}

// RemoveTopic - removes a PubSub topic and cleans up it's cache
func RemoveTopic(topic string) error {
	if _, ok := topics[topic]; !ok {
		return fmt.Errorf("Can't remove topic %s, this topic is unknown", topic)
	}

	// Clean the cache
	var cacheErr error
	if err := globalCache.Delete(topic); err != nil {
		cacheErr = fmt.Errorf("Hit error deleting the cache item for topic %s: %s", topic, err.Error())
	}

	// Stop the notifier and remove the topic from the topics map
	topics[topic].notifier.Stop()
	delete(topics, topic)
	return cacheErr
}

// GetPubSub - find a PubSub by topic name
func GetPubSub(topic string) (notification.PubSub, error) {
	cPubSub, ok := topics[topic]
	if !ok {
		return nil, fmt.Errorf("Could not find topic: %s", topic)
	}

	return cPubSub, nil
}

func (c *cachePubSub) hasChanged(key string, data interface{}) (bool, error) {
	changed, err := globalCache.HasItemChanged(key, data)
	if !changed && err != nil {
		return false, err
	}
	return changed, nil
}

func (c *cachePubSub) updateCache(key, secondaryKey string, data interface{}) {
	globalCache.Set(key, data)
	if secondaryKey != "" {
		globalCache.SetSecondaryKey(key, secondaryKey)
	}
}

func (c *cachePubSub) setAndSend(key string, data interface{}) error {
	return c.setAndSendWithSecondaryKey(key, "", data)
}

func (c *cachePubSub) setAndSendWithSecondaryKey(key, secondaryKey string, data interface{}) error {
	c.updateCache(key, secondaryKey, data)
	c.channel <- data
	return nil
}

func (c *cachePubSub) setHashAndSend(key string, data interface{}, hash uint64) error {
	return c.setHashAndSendWithSecondaryKey(key, "", data, hash)
}

func (c *cachePubSub) setHashAndSendWithSecondaryKey(key, secondaryKey string, data interface{}, hash uint64) error {
	c.updateCache(key, secondaryKey, hash)
	c.channel <- data
	return nil
}

// Publish - send in data to publish, if it has changed update cache and notify subscribers
func (c *cachePubSub) Publish(key, secondaryKey string, data interface{}) error {
	changed, err := c.hasChanged(key, data)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setAndSendWithSecondaryKey(key, secondaryKey, data)
}

// Publish - send in data to publish, if it has changed update cache and notify subscribers
func (c *cachePubSub) PublishToTopic(data interface{}) error {
	changed, err := c.hasChanged(c.topic, data)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setAndSend(c.topic, data)
}

// Publish - send in data to publish, if it has changed update cache and notify subscribers
func (c *cachePubSub) PublishToTopicWithSecondaryKey(secondaryKey string, data interface{}) error {
	changed, err := c.hasChanged(c.topic, data)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setAndSendWithSecondaryKey(c.topic, secondaryKey, data)
}

// PublishCacheHash - send in data to publish, if it has changed update cache, storing only the hash, notify subscribers
func (c *cachePubSub) PublishCacheHash(key, secondaryKey string, data interface{}) error {
	hash, err := util.ComputeHash(data)
	if err != nil {
		return err
	}

	changed, err := c.hasChanged(key, hash)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setHashAndSendWithSecondaryKey(key, secondaryKey, data, hash)
}

// PublishCacheHash - send in data to publish, if it has changed update cache, storing only the hash, notify subscribers
func (c *cachePubSub) PublishCacheHashToTopic(data interface{}) error {
	hash, err := util.ComputeHash(data)
	if err != nil {
		return err
	}

	changed, err := c.hasChanged(c.topic, hash)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setHashAndSend(c.topic, data, hash)
}

// PublishCacheHash - send in data to publish, if it has changed update cache, storing only the hash, notify subscribers
func (c *cachePubSub) PublishCacheHashToTopicWithSecondaryKey(secondaryKey string, data interface{}) error {
	hash, err := util.ComputeHash(data)
	if err != nil {
		return err
	}

	changed, err := c.hasChanged(c.topic, hash)
	if !changed || err != nil {
		return err
	}

	// Data has changed
	return c.setHashAndSendWithSecondaryKey(c.topic, secondaryKey, data, hash)
}

// Unsubscribe - stop subscriber identified by id
func (c *cachePubSub) Unsubscribe(id string) error {
	return c.notifier.Unsubscribe(id)
}

// Subscribe - creates a subscriber to this cache topic
func (c *cachePubSub) Subscribe() (chan interface{}, string) {
	channel := make(chan interface{})
	subscriber, _ := notification.Subscribe(c.topic, channel)
	return channel, subscriber.GetID()
}

// SubscribeWithCallback - creates a subscriber to this cache topic, when data is received the callback is called
func (c *cachePubSub) SubscribeWithCallback(callback func(data interface{})) string {
	channel := make(chan interface{})
	subscriber, _ := notification.Subscribe(c.topic, channel)

	// Start a go function looping for data, when received send to callback
	go func() {
		for {
			msg, ok := <-channel
			if ok {
				callback(msg)
			}
		}
	}()

	return subscriber.GetID()
}
