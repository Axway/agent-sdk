package chimera

import "github.com/streadway/amqp"

type SubscribeBinding []byte

// Structure of a Chimera subscription.
type Subscribe struct {
	Binding  SubscribeBinding `json:"binding"`
	LifeTime int              `json:"lifetime"`
}

func (s *SubscribeBinding) MarshalJSON() ([]byte, error) {
	return []byte(*s), nil
}

func (s *SubscribeBinding) UnmarshalJSON(data []byte) error {
	*s = data
	return nil
}

// Subscription information returned by Chimera after subscription call.
type SubscriptionMeta struct {
	ID      string                 `json:"id"`
	URI     string                 `json:"uri"`
	Proxy   string                 `json:"proxy"`
	Queues  []string               `json:"queues"`
	Options map[string]interface{} `json:"options"`
}

type Subscription struct {
	uri        string
	queue      string
	connection *amqp.Connection
	channel    *amqp.Channel
}

func (s Subscription) Close() error {
	if s.connection == nil {
		return nil
	}
	return s.connection.Close()
}
