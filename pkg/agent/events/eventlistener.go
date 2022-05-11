package events

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

// APIClient client interface for handling resources
type APIClient interface {
	GetResource(url string) (*apiv1.ResourceInstance, error)
	CreateResource(url string, bts []byte) (*apiv1.ResourceInstance, error)
	UpdateResource(url string, bts []byte) (*apiv1.ResourceInstance, error)
	DeleteResourceInstance(ri *apiv1.ResourceInstance) error
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	cancel          context.CancelFunc
	client          APIClient
	ctx             context.Context
	errCh           chan error
	handlers        []handler.Handler
	logger          log.FieldLogger
	sequenceManager SequenceProvider
	source          chan *proto.Event
}

// NewListenerFunc type for creating a new listener
type NewListenerFunc func(
	source chan *proto.Event, client APIClient, sequenceManager SequenceProvider, cbs ...handler.Handler,
) *EventListener

// NewEventListener creates a new EventListener to process events based on the provided Handlers.
func NewEventListener(
	source chan *proto.Event, client APIClient, sequenceManager SequenceProvider, cbs ...handler.Handler,
) *EventListener {
	ctx, cancel := context.WithCancel(context.Background())
	logger := log.NewFieldLogger().
		WithComponent("EventListener").
		WithPackage("sdk.agent.events")

	return &EventListener{
		cancel:          cancel,
		client:          client,
		ctx:             ctx,
		errCh:           make(chan error),
		handlers:        cbs,
		logger:          logger,
		sequenceManager: sequenceManager,
		source:          source,
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
	go em.start()
	return em.errCh
}

func (em *EventListener) start() {
	for {
		select {
		case event, ok := <-em.source:
			if !ok {
				em.errCh <- fmt.Errorf("harvester event source has been closed")
				em.Stop()
				return
			}

			go func(evt *proto.Event) {
				err := em.handleEvent(evt)
				if err != nil {
					em.logger.WithError(err).Error("harvester event listener error")
				}
			}(event)

		case <-em.ctx.Done():
			em.logger.Trace("harvester event listener has been gracefully stopped")
			em.errCh <- nil
		}
	}
}

// handleEvent fetches the api server ResourceClient based on the event self link, and then tries to save it to the cache.
func (em *EventListener) handleEvent(event *proto.Event) error {
	ctx := handler.NewEventContext(event.Type, event.Metadata, event.Payload.Name, event.Payload.Kind)
	em.logger.WithField("sequence", event.Metadata.SequenceID).Trace("processing watch event")

	ri, err := em.getEventResource(event)
	if err != nil {
		return err
	}

	em.handleResource(ctx, event.Metadata, ri)
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
func (em *EventListener) handleResource(
	ctx context.Context,
	eventMetadata *proto.EventMeta,
	resource *apiv1.ResourceInstance,
) {
	for _, h := range em.handlers {
		err := h.Handle(ctx, eventMetadata, resource)
		if err != nil {
			em.logger.Error(err)
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
