package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/apic"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	APIService         = "APIService"
	APIServiceInstance = "APIServiceInstance"
	Category           = "Category"
)

// Starter an interface to start a process
type Starter interface {
	Start() error
}

type callback func(action proto.Event_Type, resource *apiv1.ResourceInstance)

// EventManager holds the various caches to save events into as they get written to the source channel.
type EventManager struct {
	apis        cache.Cache
	categories  cache.Cache
	instances   cache.Cache
	source      chan *proto.Event
	getResource ResourceGetter
	cbs         []callback
}

// TODO: add option to pass in a list of callbacks for additional event processing.

// NewEventManager creates a new EventManager to save events into the appropriate cache.
func NewEventManager(source chan *proto.Event, ri ResourceGetter, apis, categories, instances cache.Cache, cbs ...callback) *EventManager {
	return &EventManager{
		apis:        apis,
		categories:  categories,
		source:      source,
		getResource: ri,
		instances:   instances,
		cbs:         cbs,
	}
}

// Start starts a loop that will cache events as they are sent on the channel
func (em *EventManager) Start() error {
	for {
		err := em.start()
		if err != nil {
			return err
		}
	}
}

// start waits for an event on the channel and then attempts to save the item.
func (em *EventManager) start() error {
	event, ok := <-em.source
	if !ok {
		return fmt.Errorf("event source has been closed")
	}

	err := em.handleEvent(event)
	if err != nil {
		log.Error(err)
	}

	return nil
}

// handleEvent fetches the api server resource based on the event self link, and then tries to save it to the cache.
func (em *EventManager) handleEvent(event *proto.Event) error {
	if event.Type == proto.Event_DELETED {
		return em.handleResourceType(event.Type, nil)
	}

	ri, err := em.getResource.Get(event.Payload.Metadata.SelfLink)
	if err != nil {
		return err
	}

	return em.handleResourceType(event.Type, ri)

}

// handleResourceType determines the resource kind to save the item to the right cache.
func (em *EventManager) handleResourceType(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	var err error
	kind := resource.GetGroupVersionKind().Kind
	switch kind {
	case APIService:
		err = em.handleAPISvc(action, resource)
	case APIServiceInstance:
		err = em.handleSvcInstance(action, resource)
	case Category:
		err = em.handleCategory(action, resource)
	default:
		logrus.Debugf("cache not provided for resource %s", kind)
	}

	for _, cb := range em.cbs {
		cb(action, resource)
	}

	return err
}

func (em *EventManager) handleAPISvc(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	id, ok := resource.Attributes[apic.AttrExternalAPIID]
	if !ok {
		return fmt.Errorf("%s not found on resource api service %s", apic.AttrExternalAPIID, resource.Name)
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		externalAPIName := resource.Attributes[apic.AttrExternalAPIName]
		primaryKey, ok := resource.Attributes[apic.AttrExternalAPIPrimaryKey]
		if !ok {
			return em.apis.SetWithSecondaryKey(id, externalAPIName, resource)
		}

		return em.apis.SetWithSecondaryKey(primaryKey, externalAPIName, resource)
	}

	if action == proto.Event_DELETED {
		return em.apis.Delete(id)
	}

	return nil
}

func (em *EventManager) handleSvcInstance(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	key := resource.Metadata.ID
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return em.instances.Set(key, resource)
	}

	if action == proto.Event_DELETED {
		return em.instances.Delete(key)
	}

	return nil
}

func (em *EventManager) handleCategory(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		// return s.categories.SetWithSecondaryKey(resource.Name, resource.Title, resource)
	}

	if action == proto.Event_DELETED {
		// return s.categories.Delete(resource.Name)
	}

	return nil
}
