package events

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/util/log"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	regularQueue     = "regularQueue"
	workerBufferSize = 5
	// amount of shards (each with its own dedicated worker) per kind lane. Events are hashed by
	// resource ID onto a shard (see kindLane), so a single resource's events always land on the
	// same worker and stay ordered, while different resources spread across all shards.
	workerCount = 3
	// provisioningLaneCount spreads demoted AccessRequest/Credential events across multiple worker
	// shards, keyed by managed application, instead of a single shared worker
	provisioningWorkerCount = 8
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

var provisioningHandlers map[string]bool = map[string]bool{
	management.ManagedApplicationGVK().Kind: true,
	management.AccessRequestGVK().Kind:      true,
	management.CredentialGVK().Kind:         true,
}

// EventListener holds the various caches to save events into as they get written to the source channel.
type EventListener struct {
	ctx                        context.Context
	cancel                     context.CancelCauseFunc
	client                     APIClient
	baseURL                    string
	handlersByKind             map[string][]handler.Handler
	logger                     log.FieldLogger
	sequenceManager            SequenceProvider
	source                     chan *proto.Event
	kindJobs                   map[string][]chan handlerData // regularWorkerCount shards per kind, one worker each, sharded by resource ID - see kindLane
	provisioningJobs           []chan handlerData            // provisioningLaneCount shards, one worker each, sharded by managed application key
	provisioningLaneAssignment map[string]int                // managed application key -> shard index in provisioningJobs, assigned on first demotion
	regularWorkerCount         int
	provisioningWorkerCount    int
	workerBuffer               int
	seqTracker                 *sequenceTracker
}

type handlerData struct {
	event            *proto.Event
	ctx              context.Context
	handler          handler.Handler
	logger           log.FieldLogger
	apiServerFields  []string
	getEventResource func(event *proto.Event, apiServerFields []string) (*apiv1.ResourceInstance, error)
	onComplete       func()
}

// sequenceTracker tracks, per dispatched event, how many handlerData items are still outstanding,
// so the persisted watch sequence only advances past events whose handlers have actually finished
// running - not merely been handed off to a lane. Events are registered in the strictly increasing
// order they're read from the source, but different lanes complete them at different speeds, so
// completions only move the watermark once they form a contiguous prefix from the front of order.
type sequenceTracker struct {
	mu        sync.Mutex
	remaining map[int64]int
	order     []int64
}

func newSequenceTracker() *sequenceTracker {
	return &sequenceTracker{remaining: make(map[int64]int)}
}

// register records that count handlerData items were dispatched for sequenceID. count may be zero
// for an event that matched no handler - it still holds its place in order so later sequences
// don't get persisted ahead of it. Returns the new watermark and true if registering it let the
// front of order advance immediately (only possible when count is zero).
func (t *sequenceTracker) register(sequenceID int64, count int) (int64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if count > 0 {
		t.remaining[sequenceID] = count
	}
	t.order = append(t.order, sequenceID)
	return t.advanceLocked()
}

// complete marks one handlerData item for sequenceID as done. Returns the new watermark and true
// if this completion let the front of order advance; otherwise (0, false).
func (t *sequenceTracker) complete(sequenceID int64) (int64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.remaining[sequenceID] > 0 {
		t.remaining[sequenceID]--
	}
	if t.remaining[sequenceID] <= 0 {
		delete(t.remaining, sequenceID)
	}
	return t.advanceLocked()
}

// advanceLocked pops sequences off the front of order for as long as they have no outstanding
// handlerData items left, returning the highest one popped, if any. Callers must hold t.mu.
func (t *sequenceTracker) advanceLocked() (int64, bool) {
	watermark, advanced := int64(0), false
	for len(t.order) > 0 {
		if _, pending := t.remaining[t.order[0]]; pending {
			break
		}
		watermark = t.order[0]
		advanced = true
		t.order = t.order[1:]
	}
	return watermark, advanced
}

type ListenerOpts func(el *EventListener)

type NewListenerFunc func(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler, opts ...ListenerOpts) *EventListener

func WithRegularWorkerCount(rwc int) ListenerOpts {
	return func(el *EventListener) {
		el.regularWorkerCount = rwc
	}
}

func WithProvisioningWorkerCount(plc int) ListenerOpts {
	return func(el *EventListener) {
		el.provisioningWorkerCount = plc
	}
}

func WithWorkerBuffer(wb int) ListenerOpts {
	return func(el *EventListener) {
		el.workerBuffer = wb
	}
}

// NewEventListener creates a new EventListener to process events based on the provided Handlers,
// indexed by the resource Kind they should be dispatched for.
func NewEventListener(ctx context.Context, cancel context.CancelCauseFunc, source chan *proto.Event, client APIClient, baseURL string, sequenceManager SequenceProvider, handlersByKind map[string][]handler.Handler, opts ...ListenerOpts) *EventListener {
	logger := log.NewFieldLogger().
		WithComponent("EventListener").
		WithPackage("sdk.agent.events")

	el := &EventListener{
		ctx:                        ctx,
		cancel:                     cancel,
		client:                     client,
		baseURL:                    baseURL,
		handlersByKind:             handlersByKind,
		logger:                     logger,
		sequenceManager:            sequenceManager,
		source:                     source,
		kindJobs:                   make(map[string][]chan handlerData),
		provisioningLaneAssignment: make(map[string]int),
		seqTracker:                 newSequenceTracker(),
		regularWorkerCount:         workerCount,
		provisioningWorkerCount:    provisioningWorkerCount,
		workerBuffer:               workerBufferSize,
	}
	for _, o := range opts {
		o(el)
	}
	el.provisioningJobs = make([]chan handlerData, el.provisioningWorkerCount)
	for i := range el.provisioningJobs {
		el.provisioningJobs[i] = make(chan handlerData, el.workerBuffer)
	}

	return el
}

func (em *EventListener) Stop() {
	if em != nil && em.cancel != nil {
		em.cancel(nil)
	}
}

func (em *EventListener) closeJobs() {
	for _, ch := range em.provisioningJobs {
		close(ch)
	}
	for _, lanes := range em.kindJobs {
		for _, ch := range lanes {
			close(ch)
		}
	}
}

// Listen starts a loop that will process events as they are sent on the channel
func (em *EventListener) Listen() {
	for _, ch := range em.provisioningJobs {
		go worker(ch) // exactly one worker per shard, order preserved within a shard (i.e. within a managed app)
	}

	for kind := range em.handlersByKind {
		lanes := make([]chan handlerData, em.regularWorkerCount)
		for i := range lanes {
			lanes[i] = make(chan handlerData, em.workerBuffer)
			go worker(lanes[i]) // exactly one worker per shard, order preserved per resource ID - see kindLane
		}
		em.kindJobs[kind] = lanes
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
		workerProcess(handlerData)
		if handlerData.onComplete != nil {
			handlerData.onComplete()
		}
	}
}

func workerProcess(handlerData handlerData) {
	ri, err := handlerData.getEventResource(handlerData.event, handlerData.apiServerFields)
	if err != nil {
		handlerData.logger.WithError(err).Error("failed to get event resource")
		return
	}

	if err := handlerData.handler.Handle(handlerData.ctx, handlerData.event.GetMetadata(), ri); err != nil {
		handlerData.logger.WithError(err).Error("failed to handle event data")
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

	queueType := ""
	msg := "skipped handling event"
	apiServerFields := requiredAPIServerFields(ctx, event, em.handlersByKind[event.Payload.Kind])

	defer func() {
		logger.WithField("apiServerFields", apiServerFields).WithField("queueType", queueType).Trace(msg)
	}()

	var toDispatch []handler.Handler
	for _, h := range em.handlersByKind[event.Payload.Kind] {
		if !h.ShouldHandle(ctx, event) {
			continue
		}
		toDispatch = append(toDispatch, h)
	}

	var jobs chan handlerData
	if len(toDispatch) > 0 {
		msg = "passed event to handlers"
		jobs, queueType = em.dispatchLane(event)
	}

	seq := event.Metadata.SequenceID
	if watermark, advanced := em.seqTracker.register(seq, len(toDispatch)); advanced {
		em.advanceSequence(watermark)
	}

	for _, h := range toDispatch {
		select {
		case jobs <- handlerData{
			event:            event,
			ctx:              ctx,
			handler:          h,
			apiServerFields:  apiServerFields,
			getEventResource: em.getEventResource,
			logger:           logger,
			onComplete: func() {
				if watermark, advanced := em.seqTracker.complete(seq); advanced {
					em.advanceSequence(watermark)
				}
			},
		}:
		case <-em.ctx.Done():
			msg = "stopped dispatching event: listener is shutting down"
			return nil
		}
	}

	// TODO: right now, we have no delete events because the WatchTopic is not configured to listen for delete calls
	// for ManagedApp/AccessRequest/Credential. maybe simplest way to fix this is to have a separate goroutine that
	// checks in the cache(every 10 mins, let's say) for the existing managedAppNames and removes them if not found
	if event.Payload.Kind == management.ManagedApplicationGVK().Kind && event.Type == proto.Event_DELETED {
		delete(em.provisioningLaneAssignment, event.Payload.Name)
	}

	return nil
}

func (em *EventListener) advanceSequence(sequenceID int64) {
	if sequenceID > em.sequenceManager.GetSequence() {
		em.sequenceManager.SetSequence(sequenceID)
	}
}

// For AccessRequest/Credential, the first event seen for a given ManagedApplication name uses that kind's lane;
// every later event tied to the same name goes to that managed app's demoted shard (see provisioningLane).
func (em *EventListener) dispatchLane(event *proto.Event) (chan handlerData, string) {
	kind := event.Payload.Kind
	resourceID := event.Payload.Metadata.Id
	if !provisioningHandlers[kind] {
		return em.kindLane(kind, resourceID), regularQueue
	}

	key, ok := managedApplicationKey(event)
	if ok {
		return em.provisioningLane(key)
	}

	return em.kindLane(kind, resourceID), regularQueue
}

// kindLane picks a resource's worker within its kind lane by hashing the resource ID, so every
// event for that resource always lands on the same worker - a lane behaves as single-threaded per
// resource, ruling out two events for the same resource ever running concurrently (the race that
// let a subresource merge handler's SetSubResource call race CreateHashes on the shared
// SubResources map - see pkg/agent/cache, ResourceMeta.CreateHashes). Different resources still
// spread across the kind's shards for parallelism.
func (em *EventListener) kindLane(kind, resourceID string) chan handlerData {
	lanes := em.kindJobs[kind]
	if len(lanes) == 0 {
		return nil
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(resourceID))
	return lanes[h.Sum32()%uint32(len(lanes))]
}

// uses the same jobIndex for a previously seem managedApp so provisioning jobs tied to a previously seen
// managedApp use the same lane
func (em *EventListener) provisioningLane(key string) (chan handlerData, string) {
	idx, ok := em.provisioningLaneAssignment[key]
	if !ok {
		idx = rand.Intn(len(em.provisioningJobs))
		em.provisioningLaneAssignment[key] = idx
	}
	return em.provisioningJobs[idx], fmt.Sprintf("provisioningQueue-%d", idx)
}

func managedApplicationKey(event *proto.Event) (key string, ok bool) {
	switch event.Payload.Kind {
	case management.ManagedApplicationGVK().Kind:
		return event.Payload.Name, true
	case management.AccessRequestGVK().Kind, management.CredentialGVK().Kind:
		for _, ref := range event.Payload.Metadata.References {
			if ref.Kind == management.ManagedApplicationGVK().Kind {
				return ref.Name, true
			}
		}
	}
	return "", false
}

// Joins the APIServerFields if multiple handlers implement the interface. In case either does not
// implement it, an empty slice is returned to signal "no restriction".
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
	if event.Payload.Kind == management.AccessRequestGVK().Kind {
		queryParams = map[string]string{"embed": "metadata.references"}
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
