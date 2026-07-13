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
	Listen()
	Stop()
}

// APIClient -
type APIClient interface {
	GetResource(url string) (*apiv1.ResourceInstance, error)
	CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	DeleteResourceInstance(ri apiv1.Interface) error
	GetAPIV1ResourceInstances(map[string]string, string) ([]*apiv1.ResourceInstance, error)
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	ctx              context.Context
	cancel           context.CancelCauseFunc
	client           APIClient
	handlersByKind   map[string][]handler.Handler
	wildcardHandlers []handler.Handler
	logger           log.FieldLogger
	sequenceManager  SequenceProvider
	source           chan *proto.Event
}

// NewListenerFunc type for creating a new listener
type NewListenerFunc func(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, sequenceManager SequenceProvider, cbs ...handler.Handler) *EventListener

// NewEventListener creates a new EventListener to process events based on the provided Handlers.
func NewEventListener(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, sequenceManager SequenceProvider, cbs ...handler.Handler) *EventListener {
	logger := log.NewFieldLogger().
		WithComponent("EventListener").
		WithPackage("sdk.agent.events")

	handlersByKind := map[string][]handler.Handler{}
	var wildcardHandlers []handler.Handler
	for _, h := range cbs {
		kinds := h.Kinds()
		if len(kinds) == 0 {
			wildcardHandlers = append(wildcardHandlers, h)
			continue
		}
		for _, kind := range kinds {
			handlersByKind[kind] = append(handlersByKind[kind], h)
		}
	}

	return &EventListener{
		ctx:              ctx,
		cancel:           cancel,
		client:           client,
		handlersByKind:   handlersByKind,
		wildcardHandlers: wildcardHandlers,
		logger:           logger,
		sequenceManager:  sequenceManager,
		source:           source,
	}
}

// Stop stops the listener
func (em *EventListener) Stop() {
	if em != nil {
		em.cancel(nil)
	}
}

// Listen starts a loop that will process events as they are sent on the channel
func (em *EventListener) Listen() {
	go func() {
		defer em.Stop()
		for {
			done, err := em.start()
			if done && err == nil {
				em.logger.Trace("stream event listener has been gracefully stopped")
				break
			}

			if err != nil {
				em.logger.WithError(err).Error("stream event listener error")
				break
			}
		}
	}()
}

func (em *EventListener) start() (done bool, err error) {
	select {
	case event, ok := <-em.source:
		if !ok {
			done = true
			err = fmt.Errorf("stream event source has been closed")
			break
		}

		if handleErr := em.handleEvent(event); handleErr != nil {
			em.logger.WithError(handleErr).Error("stream event listener error handling event")
		}
	case <-em.ctx.Done():
		em.logger.Trace("stream event listener context is done")
		done = true
		err = nil
		break
	}

	return done, err
}

// handleEvent fetches the api server ResourceClient based on the event self link, and then tries to save it to the cache.
func (em *EventListener) handleEvent(event *proto.Event) error {
	ctx := handler.NewEventContext(event.Type, event.Metadata, event.Payload.Kind, event.Payload.Name)
	em.logger.
		WithField("sequence", event.Metadata.SequenceID).
		WithField("kind", event.Payload.Kind).
		WithField("name", event.Payload.Name).
		WithField("type", event.Type.String()).
		Debug("processing watch event")

	var ri *apiv1.ResourceInstance
	var err error
	process := func(h handler.Handler) error {
		if !h.ShouldHandle(ctx, event) {
			return nil
		}
		if ri == nil {
			ri, err = em.getEventResource(event)
			if err != nil {
				return err
			}
		}
		if err := h.Handle(ctx, event.Metadata, ri); err != nil {
			em.logger.Error(err)
		}
		return nil
	}

	for _, h := range em.handlersByKind[event.Payload.Kind] {
		if err := process(h); err != nil {
			return err
		}
	}
	for _, h := range em.wildcardHandlers {
		if err := process(h); err != nil {
			return err
		}
	}

	em.sequenceManager.SetSequence(event.Metadata.SequenceID)
	return nil
}

func (em *EventListener) getEventResource(event *proto.Event) (*apiv1.ResourceInstance, error) {
	if event.Type == proto.Event_DELETED {
		return em.convertEventPayload(event), nil
	}
	return em.client.GetResource(event.Payload.Metadata.SelfLink)
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
