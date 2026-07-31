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
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	lowQueue     = "lowPriorityQueue"
	regularQueue = "regularQueue"

	// laneBufferSize lets a lane's dispatch send get a few events ahead of its worker, so a
	// short burst of slow handling on one lane doesn't immediately stall dispatch for every
	// other lane - there's only one dispatch goroutine feeding all of them.
	laneBufferSize = 5
)

type Listener interface {
	Listen()
	Stop()
}

type APIClient interface {
	ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error)
	GetResource(url string) (*apiv1.ResourceInstance, error)
	CreateResourceInstance(ri apiv1.Interface) (*apiv1.ResourceInstance, error)
	DeleteResourceInstance(ri apiv1.Interface) error
	GetAPIV1ResourceInstances(map[string]string, string) ([]*apiv1.ResourceInstance, error)
}

var lowPriorityHandlers map[string]bool = map[string]bool{
	management.ManagedApplicationGVK().Kind: true,
	management.AccessRequestGVK().Kind:      true,
	management.CredentialGVK().Kind:         true,
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
	kindJobs        map[string]chan handlerData // one worker per kind
	lowPriorityJobs chan handlerData            // exactly one dedicated worker
	seenManagedApps map[string]struct{}         // managed application names already dispatched to their kind's lane at least once
}

type handlerData struct {
	event   *proto.Event
	ctx     context.Context
	ri      *apiv1.ResourceInstance
	handler handler.Handler
	logger  log.FieldLogger
}

type ListenerOpts func(el *EventListener)

type NewListenerFunc func(source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler, opts ...ListenerOpts) *EventListener

func WithContextAndCancel(ctx context.Context, cancel context.CancelCauseFunc) ListenerOpts {
	return func(el *EventListener) {
		el.ctx = ctx
		el.cancel = cancel
	}
}

// NewEventListener creates a new EventListener to process events based on the provided Handlers,
// indexed by the resource Kind they should be dispatched for.
func NewEventListener(source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler, opts ...ListenerOpts) *EventListener {
	logger := log.NewFieldLogger().
		WithComponent("EventListener").
		WithPackage("sdk.agent.events")

	el := &EventListener{
		ctx:             context.Background(),
		client:          client,
		baseURL:         baseURL,
		handlersByKind:  handlersByKind,
		logger:          logger,
		sequenceManager: sequenceManager,
		source:          source,
		kindJobs:        make(map[string]chan handlerData, len(handlersByKind)-len(lowPriorityHandlers)),
		lowPriorityJobs: make(chan handlerData, laneBufferSize),
		seenManagedApps: make(map[string]struct{}),
	}
	for _, o := range opts {
		o(el)
	}
	return el
}

func (em *EventListener) Stop() {
	if em != nil && em.cancel != nil {
		em.cancel(nil)
	}
}

func (em *EventListener) closeJobs() {
	close(em.lowPriorityJobs)
	for _, ch := range em.kindJobs {
		close(ch)
	}
}

// Listen starts a loop that will process events as they are sent on the channel
func (em *EventListener) Listen() {
	go worker(em.lowPriorityJobs) // exactly one goroutine, shared by already-known managed apps

	for kind := range em.handlersByKind {
		ch := make(chan handlerData, laneBufferSize)
		em.kindJobs[kind] = ch
		go worker(ch) // exactly one worker, order preserved for this kind
	}

	go func() {
		defer em.Stop()
		defer em.closeJobs()
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

func worker(jobs <-chan handlerData) {
	for handlerData := range jobs {
		if err := handlerData.handler.Handle(handlerData.ctx, handlerData.event.GetMetadata(), handlerData.ri); err != nil {
			handlerData.logger.WithError(err).Error("failed to handle event data")
		}
	}
}

// handleEvent fetches the api server ResourceClient based on the event self link, and then tries to save it to the cache.
func (em *EventListener) handleEvent(event *proto.Event) error {
	ctx := handler.NewEventContext(event.Type, event.Metadata, event.Payload.Kind, event.Payload.Name)
	logger := em.logger.
		WithField("sequence", event.Metadata.SequenceID).
		WithField("kind", event.Payload.Kind).
		WithField("name", event.Payload.Name).
		WithField("type", event.Type.String()).
		WithField("subResource", event.Metadata.Subresource)

	logger.Debug("processing watch event")

	var ri *apiv1.ResourceInstance
	var err error
	var jobs chan handlerData
	queueType := ""
	msg := "skipped handling event"
	apiServerFields := requiredAPIServerFields(ctx, event, em.handlersByKind[event.Payload.Kind])

	defer func() {
		logger.WithField("apiServerFields", apiServerFields).WithField("queueType", queueType).Trace(msg)
	}()

	for _, h := range em.handlersByKind[event.Payload.Kind] {
		if !h.ShouldHandle(ctx, event) {
			continue
		}
		msg = "passed event to handlers"
		if ri == nil {
			ri, err = em.getEventResource(event, apiServerFields)
			if err != nil {
				logger.WithError(err).Error("failed to get event resource")
				return err
			}
		}
		if jobs == nil {
			jobs, queueType = em.dispatchLane(event, ri)
		}
		select {
		case jobs <- handlerData{
			event:   event,
			ctx:     ctx,
			ri:      ri,
			handler: h,
			logger:  logger,
		}:
		case <-em.ctx.Done():
			// a lane's worker is stuck or the process is shutting down - stop dispatching this
			// event rather than block forever, and skip the sequence update below so the event
			// gets redelivered and retried in full after a restart
			msg = "stopped dispatching event: listener is shutting down"
			return nil
		}
	}

	if event.Payload.Kind == management.ManagedApplicationGVK().Kind && event.Type == proto.Event_DELETED {
		delete(em.seenManagedApps, event.Payload.Name)
	}

	if event.Metadata.SequenceID > em.sequenceManager.GetSequence() {
		em.sequenceManager.SetSequence(event.Metadata.SequenceID)
	}
	return nil
}

// dispatchLane picks which worker channel an event's handlerData should be sent to.
// Kinds outside the managed-app provisioning chain always use their own kind lane.
// For AccessRequest/Credential, the first event seen for a given ManagedApplication name uses that kind's lane;
// every later event tied to the same name goes to the shared low-priority lane, until the ManagedApplication is deleted (see handleEvent).
func (em *EventListener) dispatchLane(event *proto.Event, ri *apiv1.ResourceInstance) (chan handlerData, string) {
	kind := event.Payload.Kind
	if !lowPriorityHandlers[kind] {
		return em.kindJobs[kind], regularQueue
	}

	key, ok := managedApplicationKey(event, ri)
	if !ok {
		// can't determine which ManagedApplication this belongs to - e.g. a delete for an
		// AccessRequest/Credential, whose spec is no longer available - so assume it's for an
		// already-known app rather than risk promoting it to the fast lane
		return em.lowPriorityJobs, lowQueue
	}

	// We add the keys, but we don't put the ManagedApplication in the low priority queue.
	if _, seen := em.seenManagedApps[key]; seen && kind != management.ManagedApplicationGVK().Kind {
		return em.lowPriorityJobs, lowQueue
	}
	em.seenManagedApps[key] = struct{}{}
	return em.kindJobs[kind], regularQueue
}

// managedApplicationKey returns the ManagedApplication name an event relates to, for the three
// kinds that participate in managed-app provisioning. ok is false for any other kind, or when
// the reference can't be determined from ri (e.g. AccessRequest/Credential delete events, whose
// spec is empty).
func managedApplicationKey(event *proto.Event, ri *apiv1.ResourceInstance) (key string, ok bool) {
	switch event.Payload.Kind {
	case management.ManagedApplicationGVK().Kind:
		return event.Payload.Name, true
	case management.AccessRequestGVK().Kind, management.CredentialGVK().Kind:
		if ri == nil || ri.Spec == nil {
			return "", false
		}
		key, ok = ri.Spec["managedApplication"].(string)
		if !ok || key == "" {
			return "", false
		}
		return key, true
	default:
		return "", false
	}
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
