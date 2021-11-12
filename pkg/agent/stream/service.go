package stream

import (
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Service struct for processing events from a grpc stream
type Service struct {
	Manager wm.Manager
	ric     ResourceGetter
}

// NewStreamService creates a NewStreamService
func NewStreamService(manager wm.Manager, ric ResourceGetter) *Service {
	return &Service{
		Manager: manager,
		ric:     ric,
	}
}

// Watch registers a WatchClient, and creates an EventManager to process the events
func (s *Service) Watch(topic string, events chan *proto.Event, errors chan error, cbs ...Handler) error {
	id, err := s.Manager.RegisterWatch(topic, events, errors)
	if err != nil {
		return err
	}

	log.Debugf("watch-controller subscription-id: %s", id)

	em := NewEventManager(events, s.ric, cbs...)

	return em.Start()
}
