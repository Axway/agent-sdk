package notification

// PubSub - interface for creating a PubSub library
type PubSub interface {
	Publish(key, secondarykey string, data interface{}) error
	PublishToTopic(data interface{}) error
	PublishToTopicWithSecondaryKey(secondarykey string, data interface{}) error
	PublishCacheHash(key, secondarykey string, data interface{}) error
	PublishCacheHashToTopic(data interface{}) error
	PublishCacheHashToTopicWithSecondaryKey(secondarykey string, data interface{}) error
	Subscribe() (msgChan chan interface{}, id string)
	SubscribeWithCallback(callback func(data interface{})) (id string)
	Unsubscribe(id string) error
}
