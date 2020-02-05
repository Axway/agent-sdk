package notification

// PubSub - interface for creating a PubSub library
type PubSub interface {
	Publish(data interface{}) error
	Subscribe() (msgChan chan interface{}, id string)
	SubscribeWithCallback(callback func(data interface{})) (id string)
	Unsubscribe(id string) error
}
