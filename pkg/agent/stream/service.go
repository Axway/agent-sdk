package stream

import (
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/cache"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

type Service struct {
	Manager    wm.Manager
	ric        RiGetter
	apis       cache.Cache
	categories cache.Cache
	instances  cache.Cache
}

func NewStreamService(manager wm.Manager, ric RiGetter, apis, categories, instances cache.Cache) *Service {
	return &Service{
		Manager:    manager,
		ric:        ric,
		apis:       apis,
		categories: categories,
		instances:  instances,
	}
}

func (s *Service) Watch(topic string, events chan *proto.Event, errors chan error, cbs ...callback) error {
	id, err := s.Manager.RegisterWatch(topic, events, errors)
	if err != nil {
		return err
	}

	log.Debug("watch-controller subscription-id: %s", id)

	em := NewEventManager(events, s.ric, s.apis, s.categories, s.instances, cbs...)

	return em.Start()
}
