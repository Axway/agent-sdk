package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/util/log"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Handler interface used by the EventManager to process events.
type Handler interface {
	// callback receives the type of the event (add, update, delete), and the resource on API Server, if it exists.
	callback(action proto.Event_Type, resource *apiv1.ResourceInstance) error
}

// Starter starts the EventManager
type Starter interface {
	Start() error
}

// EventManager holds the various caches to save events into as they get written to the source channel.
type EventManager struct {
	source      chan *proto.Event
	getResource ResourceGetter
	handlers    []Handler
}

// NewEventManager creates a new EventManager to process events based on the provided Handlers.
func NewEventManager(source chan *proto.Event, ri ResourceGetter, cbs ...Handler) *EventManager {
	return &EventManager{
		source:      source,
		getResource: ri,
		handlers:    cbs,
	}
}

// Start starts a loop that will process events as they are sent on the channel
func (em *EventManager) Start() error {
	for {
		err := em.start()
		if err != nil {
			return err
		}
	}
}

// start waits for an event on the channel and then attempts to pass the item to the handlers.
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
	var ri *apiv1.ResourceInstance
	var err error
	if event.Type == proto.Event_CREATED || event.Type == proto.Event_UPDATED {
		ri, err = em.getResource.Get(event.Payload.Metadata.SelfLink)
		if err != nil {
			return err
		}
	}

	em.handleResource(event.Type, ri)

	return nil
}

// handleResource loops through all the handlers and passes the event to each one for processing.
func (em *EventManager) handleResource(action proto.Event_Type, resource *apiv1.ResourceInstance) {
	for _, cb := range em.handlers {
		err := cb.callback(action, resource)
		if err != nil {
			log.Error(err)
		}
	}
}
