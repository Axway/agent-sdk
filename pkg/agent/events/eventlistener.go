package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	GetResource(url string) (*apiv1.ResourceInstance, error)
	CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	DeleteResourceInstance(ri apiv1.Interface) error
	GetAPIV1ResourceInstances(map[string]string, string) ([]*apiv1.ResourceInstance, error)
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	ctx             context.Context
	cancel          context.CancelCauseFunc
	client          APIClient
	baseURL         string
	handlersByKind  map[string][]handler.Handler
	logger          log.FieldLogger
	sequenceManager SequenceProvider
	source          chan *proto.Event
}

// NewListenerFunc type for creating a new listener
type NewListenerFunc func(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler) *EventListener

// NewEventListener creates a new EventListener to process events based on the provided Handlers,
// indexed by the resource Kind they should be dispatched for.
func NewEventListener(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler) *EventListener {
	logger := log.NewFieldLogger().
		WithComponent("EventListener").
		WithPackage("sdk.agent.events")

	return &EventListener{
		ctx:             ctx,
		cancel:          cancel,
		client:          client,
		baseURL:         baseURL,
		handlersByKind:  handlersByKind,
		logger:          logger,
		sequenceManager: sequenceManager,
		source:          source,
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
		WithField("subResource", event.Metadata.Subresource).
		Debug("processing watch event")

	var ri *apiv1.ResourceInstance
	var err error
	apiServerFields := requiredAPIServerFields(ctx, event, em.handlersByKind[event.Payload.Kind])
	for _, h := range em.handlersByKind[event.Payload.Kind] {
		if !h.ShouldHandle(ctx, event) {
			continue
		}

		if ri == nil {
			ri, err = em.getEventResource(event, apiServerFields)
			if err != nil {
				return err
			}
		}
		if err := h.Handle(ctx, event.Metadata, ri); err != nil {
			em.logger.Error(err)
		}
	}

	em.sequenceManager.SetSequence(event.Metadata.SequenceID)
	return nil
}

// requiredAPIServerFields returns the union of the fields declared as required by the given
// handlers, preserving first-seen order. If any handler does not restrict itself to specific
// fields - either by not implementing RequiredFieldsHandler, or by declaring none - the full
// resource is required, so an empty slice is returned to signal "no restriction".
func requiredAPIServerFields(ctx context.Context, event *proto.Event, handlers []handler.Handler) []string {
	seen := map[string]struct{}{}
	fields := []string{}
	for _, h := range handlers {
		rfh, ok := h.(handler.RequiredFieldsHandler)
		if !ok {
			return nil
		}

		hFields := rfh.GetAPIServerFields(ctx, event)
		if len(hFields) == 0 {
			return nil
		}

		for _, f := range hFields {
			if _, ok := seen[f]; !ok {
				seen[f] = struct{}{}
				fields = append(fields, f)
			}
		}
	}
	return fields
}

func (em *EventListener) getEventResource(event *proto.Event, apiServerFields []string) (*apiv1.ResourceInstance, error) {
	if event.Type == proto.Event_DELETED {
		return em.convertEventPayload(event), nil
	}

	queryParams := map[string]string{}
	if len(apiServerFields) > 0 {
		queryParams["fields"] = strings.Join(apiServerFields, ",")
	}

	url := fmt.Sprintf("%s/apis%s", em.baseURL, event.Payload.Metadata.SelfLink)
	resp, err := em.client.ExecuteAPI(http.MethodGet, url, queryParams, nil)
	if err != nil {
		return nil, err
	}

	ri := &apiv1.ResourceInstance{}
	if err := json.Unmarshal(resp, ri); err != nil {
		return nil, err
	}
	return ri, nil
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
