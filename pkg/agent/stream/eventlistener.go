package stream

import (
	"context"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/util/log"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// Listener starts the EventListener
type Listener interface {
	Listen() chan error
	Stop()
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	cancel          context.CancelFunc
	ctx             context.Context
	client          apiClient
	handlers        []handler.Handler
	source          chan *proto.Event
	sequenceManager *agentSequenceManager
}

type newListenerFunc func(source chan *proto.Event, ri apiClient, sequenceManager *agentSequenceManager, cbs ...handler.Handler) *EventListener

// NewEventListener creates a new EventListener to process events based on the provided Handlers.
func NewEventListener(source chan *proto.Event, ri apiClient, sequenceManager *agentSequenceManager, cbs ...handler.Handler) *EventListener {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventListener{
		cancel:          cancel,
		ctx:             ctx,
		client:          ri,
		handlers:        cbs,
		source:          source,
		sequenceManager: sequenceManager,
	}
}

// Stop stops the listener
func (em *EventListener) Stop() {
	if em != nil {
		em.cancel()
	}
}

// Listen starts a loop that will process events as they are sent on the channel
func (em *EventListener) Listen() chan error {
	errCh := make(chan error)
	go func() {
		for {
			done, err := em.start()
			if done && err == nil {
				errCh <- nil
				break
			}

			if err != nil {
				errCh <- err
				break
			}
		}
	}()

	return errCh
}

func (em *EventListener) start() (done bool, err error) {
	select {
	case event, ok := <-em.source:
		if !ok {
			done = true
			err = fmt.Errorf("stream event source has been closed")
			break
		}

		err := em.handleEvent(event)
		if err != nil {
			log.Errorf("stream event listener error: %s", err)
		}
	case <-em.ctx.Done():
		log.Tracef("stream event listener has been gracefully stopped")
		done = true
		err = nil
		break
	}

	return done, err
}

// handleEvent fetches the api server ResourceClient based on the event self link, and then tries to save it to the cache.
func (em *EventListener) handleEvent(event *proto.Event) error {
	log.Debugf(
		"processing received watch event[sequenceID: %d, action: %s, type: %s, name: %s]",
		event.Metadata.SequenceID,
		proto.Event_Type_name[int32(event.Type)],
		event.Payload.Kind,
		event.Payload.Name,
	)

	ri, err := em.getEventResource(event)
	if err != nil {
		return err
	}

	em.handleResource(event.Type, event.Metadata, ri)
	em.sequenceManager.SetSequence(event.Metadata.SequenceID)
	return nil
}

func (em *EventListener) getEventResource(event *proto.Event) (*apiv1.ResourceInstance, error) {
	if event.Type == proto.Event_DELETED {
		return em.convertEventPayload(event), nil
	}
	return em.client.GetResource(event.Payload.Metadata.SelfLink)
}

// handleResource loops through all the handlers and passes the event to each one for processing.
func (em *EventListener) handleResource(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *apiv1.ResourceInstance) {
	for _, h := range em.handlers {
		err := h.Handle(action, eventMetadata, resource)
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
