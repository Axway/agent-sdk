package stream

import (
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/cache"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Service struct for processing events from a grpc stream
type Service struct {
	Manager    wm.Manager
	ric        ResourceGetter
	apis       cache.Cache
	categories cache.Cache
	instances  cache.Cache
}

// NewStreamService creates a NewStreamService
func NewStreamService(manager wm.Manager, ric ResourceGetter, apis, categories, instances cache.Cache) *Service {
	return &Service{
		Manager:    manager,
		ric:        ric,
		apis:       apis,
		categories: categories,
		instances:  instances,
	}
}

// Watch registers a WatchClient, and creates an EventManager to process the events
func (s *Service) Watch(topic string, events chan *proto.Event, errors chan error, cbs ...callback) error {
	id, err := s.Manager.RegisterWatch(topic, events, errors)
	if err != nil {
		return err
	}

	log.Debugf("watch-controller subscription-id: %s", id)

	em := NewEventManager(events, s.ric, s.apis, s.categories, s.instances, cbs...)

	return em.Start()
}
