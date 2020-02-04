package notification

// Subscriber - subscribes to a notifier
type Subscriber interface {
	GetID() string
	SendMsg(interface{})
	Close() // Unsubscribes and closes channel
	close() // closes channel
}

type notifierSubscriber struct {
	Subscriber
	id       string
	notifier string
	output   chan interface{}
}

// GetID - return the id of this subscriber
func (s *notifierSubscriber) GetID() string {
	return s.id
}

// SendMsg - the notifier calls this on the subscriber to send data
func (s *notifierSubscriber) SendMsg(data interface{}) {
	select {
	case <-s.output:
	default:
		s.output <- data
	}
}

// Close - used to unsubscribe this Subscriber from its notifier and close the channel
func (s *notifierSubscriber) Close() {
	Unsubscribe(s.notifier, s.id)
}

// Close - used to unsubscribe this Subscriber from its notifier and close the channel
func (s *notifierSubscriber) close() {
	// Close the channel
	close(s.output)
}
