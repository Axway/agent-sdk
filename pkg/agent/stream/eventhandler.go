package stream

import (
	"fmt"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Starter an interface to start a process
type Starter interface {
	Start() error
}

// EventManager holds the various caches to save events into as they get written to the source channel.
type EventManager struct {
	apis        cache.Cache
	categories  cache.Cache
	source      chan *proto.Event
	getResource RiGetter
}

// NewEventManager creates a new EventManager to save events into the appropriate cache.
func NewEventManager(source chan *proto.Event, getResource RiGetter, apis cache.Cache, categories cache.Cache) *EventManager {
	return &EventManager{
		apis:        apis,
		categories:  categories,
		source:      source,
		getResource: getResource,
	}
}

// Start starts a loop that will cache events as they are sent on the channel
func (ec *EventManager) Start() error {
	for {
		err := ec.start()
		if err != nil {
			return err
		}
	}
}

// start waits for an event on the channel and then attempts to save the item.
func (ec *EventManager) start() error {
	event, ok := <-ec.source
	if !ok {
		return fmt.Errorf("event source has been closed")
	}

	return ec.handleEvent(event)
}

// handleEvent fetches the api server resource based on the event self link, and then tries to save it to the cache.
func (ec *EventManager) handleEvent(event *proto.Event) error {
	ri, err := ec.getResource.Get(event.Payload.Metadata.SelfLink)
	if err != nil {
		return err
	}

	switch v := ri.(type) {
	case *apiv1.ResourceInstance:
		return ec.handleResourceType(event.Type, v)
	case nil:
		return fmt.Errorf("received event, but the returned api server resource is nil")
	default:
		fmt.Printf("unable to convert type to *ResourceInstance. %s %+v", v.GetName(), v.GetGroupVersionKind())
	}

	return nil
}

// handleResourceType determines the resource kind to save the item to the right cache.
func (ec *EventManager) handleResourceType(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	kind := resource.GetGroupVersionKind().Kind
	switch kind {
	case "APIService":
		return ec.handleSvcInstance(action, resource)
	case "APIServiceInstance":
		return ec.handleSvcInstance(action, resource)
	case "Category":
		return ec.handleCategory(action, resource)
	default:
		logrus.Debugf("received unexpected event for kind %s. It will not be cached.", kind)
	}

	return nil
}

func (ec *EventManager) handleAPISvc(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	// id, ok := resource.Attributes[apic.AttrExternalAPIID]
	// if !ok {
	// 	return fmt.Errorf("%s not found on resource api service %s", apic.AttrExternalAPIID, resource.Name)
	// }

	// primaryKey, ok := resource.Attributes[apic.AttrExternalAPIPrimaryKey]
	// if !ok {
	//
	// }

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		// return s.apis.SetWithSecondaryKey("", "", resource)
	}

	if action == proto.Event_DELETED {
		// return s.apis.Delete(id)
	}

	return nil
}

func (ec *EventManager) handleSvcInstance(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	// instances, err := s.apis.Get(serviceInstanceCache)
	// if err != nil {
	// 	return err
	// }

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		// return s.apis.SetWithSecondaryKey("", "", resource)
	}

	if action == proto.Event_DELETED {
		// return s.apis.Delete("")
	}

	return nil
}

func (ec *EventManager) handleCategory(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		// return s.categories.SetWithSecondaryKey(resource.Name, resource.Title, resource)
	}

	if action == proto.Event_DELETED {
		// return s.categories.Delete(resource.Name)
	}

	return nil
}
