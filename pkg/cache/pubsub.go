package cache

import (
	"fmt"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/notification"
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

// Publish - send in data to publish, if it has changed update cache and notify subscribers
func (c *cachePubSub) Publish(key, secondarykey string, data interface{}) error {
	changed, err := globalCache.HasItemChanged(key, data)
	if !changed && err != nil {
		return err
	}
	if !changed {
		return nil
	}

	// Data has changed
	globalCache.Set(key, data)
	globalCache.SetSecondaryKey(key, secondarykey)
	c.channel <- data
	return nil
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
