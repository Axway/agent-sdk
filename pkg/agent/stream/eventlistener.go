package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/util/log"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Handler interface used by the EventListener to process events.
type Handler interface {
	// handle receives the type of the event (add, update, delete), and the ResourceClient on API Server, if it exists.
	handle(action proto.Event_Type, resource *apiv1.ResourceInstance) error
}

// Listener starts the EventListener
type Listener interface {
	Listen() error
	Stop()
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	getResource ResourceClient
	handlers    []Handler
	source      chan *proto.Event
	stop        chan interface{}
	isRunning   bool
}

// NewEventListener creates a new EventListener to process events based on the provided Handlers.
func NewEventListener(source chan *proto.Event, ri ResourceClient, cbs ...Handler) *EventListener {
	return &EventListener{
		getResource: ri,
		handlers:    cbs,
		source:      source,
		stop:        make(chan interface{}),
	}
}

// Stop stops the listener
func (em *EventListener) Stop() {
	em.stop <- nil
}

// Listen starts a loop that will process events as they are sent on the channel
func (em *EventListener) Listen() error {
	if em.isRunning {
		return fmt.Errorf("event listener is already running")
	}

	var listenErr error
	em.isRunning = true

	for {
		done, err := em.start()
		if err != nil {
			listenErr = err
			break
		}

		if done == true {
			listenErr = nil
			break
		}
	}

	em.isRunning = false
	return listenErr
}

// start waits for an event on the channel and then attempts to pass the item to the handlers.
// Return true if processing should end, and false for it to continue.
func (em *EventListener) start() (done bool, err error) {
	select {
	case event, ok := <-em.source:
		if !ok {
			return true, fmt.Errorf("stream event source has been closed")
		}

		err := em.handleEvent(event)
		if err != nil {
			log.Errorf("stream event listener error: %s", err)
		}

		return false, nil
	case <-em.stop:
		log.Tracef("stream event listener has been gracefully stopped")
		return true, nil
	}
}

// handleEvent fetches the api server ResourceClient based on the event self link, and then tries to save it to the cache.
func (em *EventListener) handleEvent(event *proto.Event) error {
	var ri *apiv1.ResourceInstance
	var err error

	log.Debugf(
		"processing received watch event[sequenceID: %d, action: %s, type: %s, name: %s]",
		event.Metadata.SequenceID,
		proto.Event_Type_name[int32(event.Type)],
		event.Payload.Kind,
		event.Payload.Name,
	)

	if event.Type == proto.Event_CREATED || event.Type == proto.Event_UPDATED {
		ri, err = em.getResource.Get(event.Payload.Metadata.SelfLink)
		if err != nil {
			return err
		}
	}

	if event.Type == proto.Event_DELETED {
		ri = em.convertEventPayload(event)
	}

	em.handleResource(event.Type, ri)

	return nil
}

// handleResource loops through all the handlers and passes the event to each one for processing.
func (em *EventListener) handleResource(action proto.Event_Type, resource *apiv1.ResourceInstance) {
	for _, h := range em.handlers {
		err := h.handle(action, resource)
		if err != nil {
			log.Error(err)
		}
	}
}

func (em *EventListener) convertEventPayload(event *proto.Event) *apiv1.ResourceInstance {
	ri := &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: event.Payload.Group,
					Kind:  event.Payload.Kind,
				},
			},
			Name: event.Payload.Name,
			Metadata: apiv1.Metadata{
				ID:       event.Payload.Metadata.Id,
				SelfLink: event.Payload.Metadata.SelfLink,
			},
			Attributes: event.Payload.Attributes,
		},
	}
	if event.Payload.Metadata.Scope != nil {
		ri.Metadata.Scope = apiv1.MetadataScope{
			ID:       event.Payload.Metadata.Scope.Id,
			Kind:     event.Payload.Metadata.Scope.Kind,
			Name:     event.Payload.Metadata.Scope.Name,
			SelfLink: event.Payload.Metadata.Scope.SelfLink,
		}
	}
	return ri
}
