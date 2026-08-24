package events

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// newTestListener builds an EventListener with a single known kind lane already wired up, as
// Listen would do, without spawning the worker goroutines - start's dispatch logic can then be
// exercised and asserted on directly by reading from the lane channels. The source channel is
// buffered so a test can queue an event before calling start() without needing a goroutine.
func newTestListener(t *testing.T, knownKind string) *EventListener {
	t.Helper()
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	ctx, cancel := context.WithCancelCause(context.Background())
	em := NewEventListener(
		ctx, cancel,
		make(chan *proto.Event, 1),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{
			knownKind:                               {&mockHandler{}},
			management.ManagedApplicationGVK().Kind: {&mockHandler{}},
		},
	)
	em.kindJobs = map[string][]chan handlerData{
		knownKind:                               {make(chan handlerData, 1)},
		management.ManagedApplicationGVK().Kind: {make(chan handlerData, 1)},
	}
	em.provisioningJobs = []chan handlerData{make(chan handlerData, 1)}
	return em
}

func TestEventListener_start(t *testing.T) {
	const knownKind = "TestKind"
	lowKind := management.ManagedApplicationGVK().Kind

	tests := []struct {
		name        string
		kind        string
		closeSource bool
		cancelCtx   bool
		wantDone    bool
		wantErr     bool
		wantLane    string // "kind", "demoted", or "" for no dispatch at all
	}{
		{
			name:     "dispatches a known kind to its own kind lane",
			kind:     knownKind,
			wantLane: "kind",
		},
		{
			// ManagedApplication events are always keyed by their own name, so they land on the
			// app's provisioning shard alongside its access requests, rather than the kind lane
			name:     "dispatches a ManagedApplication event to its managed app's shard",
			kind:     lowKind,
			wantLane: "demoted",
		},
		{
			name: "drops an event for a kind with no registered lane instead of blocking",
			kind: "UnknownKind",
		},
		{
			name:        "returns an error when the event source is closed",
			closeSource: true,
			wantDone:    true,
			wantErr:     true,
		},
		{
			name:      "stops gracefully when the context is cancelled",
			cancelCtx: true,
			wantDone:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := newTestListener(t, knownKind)

			var event *proto.Event
			switch {
			case tc.closeSource:
				close(em.source)
			case tc.cancelCtx:
				em.cancel(nil)
			default:
				event = newTestEvent(1)
				event.Payload.Kind = tc.kind
				event.Payload.Name = "app-1"
				em.source <- event
			}

			done, err := em.start()
			assert.Equal(t, tc.wantDone, done)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			var lane chan handlerData
			switch tc.wantLane {
			case "kind":
				lane = em.kindJobs[tc.kind][0]
			case "demoted":
				lane = em.provisioningJobs[0]
			}
			if lane == nil {
				return
			}
			select {
			case got := <-lane:
				assert.Same(t, event, got.event)
			default:
				t.Fatal("event was not dispatched to the expected lane")
			}
		})
	}
}

// Should call Listen and handle a graceful stop, and an error
func TestEventListener_Listen(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	events := make(chan *proto.Event)
	ctx, cancel := context.WithCancelCause(context.Background())
	listener := NewEventListener(ctx, cancel, events, &mockAPIClient{}, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {&mockHandler{}}})
	listener.Listen()
	listener.Stop()
	err := ctx.Err()
	assert.NotNil(t, err)

	ctx, cancel = context.WithCancelCause(context.Background())
	listener = NewEventListener(ctx, cancel, events, &mockAPIClient{}, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {&mockHandler{}}})
	listener.Listen()
	close(events)
	err = ctx.Err()
	assert.Nil(t, err)
}

func TestEventListener_handleEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    proto.Event_Type
		hasError bool
		client   APIClient
		handler  handler.Handler
	}{
		{
			name:     "should process a delete event with no error",
			event:    proto.Event_DELETED,
			hasError: false,
			client:   &mockAPIClient{},
			handler:  &mockHandler{},
		},
		{
			name:    "should return when the request to get a ResourceClient fails",
			event:   proto.Event_CREATED,
			client:  &mockAPIClient{getErr: fmt.Errorf("err")},
			handler: &mockHandler{},
		},
		{
			name:     "should get a ResourceClient, and process a create event",
			event:    proto.Event_CREATED,
			hasError: false,
			client:   &mockAPIClient{},
			handler:  &mockHandler{},
		},
		{
			name:     "should get a ResourceClient, and process an update event",
			event:    proto.Event_UPDATED,
			hasError: false,
			client:   &mockAPIClient{},
			handler:  &mockHandler{},
		},
	}
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &proto.Event{
				Type: tc.event,
				Payload: &proto.ResourceInstance{
					Metadata: &proto.Metadata{
						SelfLink: "/management/v1/watchtopics/mock-watch-topic",
						Scope: &proto.Metadata_ScopeKind{
							Kind:     "Kind",
							Name:     "Name",
							SelfLink: "/self/link",
						},
					},
				},
				Metadata: &proto.EventMeta{
					SequenceID: 1,
				},
			}

			ctx, cancel := context.WithCancelCause(context.Background())
			listener := NewEventListener(ctx, cancel, make(chan *proto.Event), tc.client, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {tc.handler}})
			listener.kindJobs[""] = []chan handlerData{make(chan handlerData, 1)}

			err := listener.handleEvent(event)

			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func Test_sequenceTracker(t *testing.T) {
	tr := newSequenceTracker()

	if _, advanced := tr.register(1, 1); advanced {
		t.Fatal("should not advance yet - nothing has completed")
	}
	if _, advanced := tr.register(2, 1); advanced {
		t.Fatal("should not advance yet - nothing has completed")
	}
	if _, advanced := tr.register(3, 1); advanced {
		t.Fatal("should not advance yet - nothing has completed")
	}

	if _, advanced := tr.complete(2); advanced {
		t.Fatal("completing 2 must not advance the watermark while 1 is still outstanding")
	}

	watermark, advanced := tr.complete(1)
	if !advanced || watermark != 2 {
		t.Fatalf("expected completing 1 to advance to 2 (already-done 2 pops along with it), got %d (advanced=%v)", watermark, advanced)
	}

	watermark, advanced = tr.complete(3)
	if !advanced || watermark != 3 {
		t.Fatalf("expected watermark 3, got %d (advanced=%v)", watermark, advanced)
	}
}

type blockingHandler struct {
	release chan struct{}
}

func (h *blockingHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *apiv1.ResourceInstance) error {
	<-h.release
	return nil
}

func (h *blockingHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool {
	return true
}

// TestEventListener_handleEvent_sequenceWaitsForHandleCompletion proves the fix for the
// crash-safety gap: the persisted sequence must not advance just because an event's handlerData
// was handed off to a worker - only once the handler has actually finished running.
func TestEventListener_handleEvent_sequenceWaitsForHandleCompletion(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	const kind = "BlockingKind"

	release := make(chan struct{})
	blocking := &blockingHandler{release: release}

	ctx, cancel := context.WithCancelCause(context.Background())
	listener := NewEventListener(
		ctx, cancel,
		make(chan *proto.Event),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{kind: {blocking}},
	)
	ch := make(chan handlerData, 1)
	listener.kindJobs[kind] = []chan handlerData{ch}
	go worker(ch)

	event := newTestEvent(5)
	event.Payload.Kind = kind

	done := make(chan error, 1)
	go func() {
		done <- listener.handleEvent(event)
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("handleEvent should return once the handlerData is handed off, without waiting for Handle to finish")
	}

	// give the worker a moment to pick up the handlerData and start blocking inside Handle
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int64(0), sequenceManager.GetSequence(), "sequence must not advance before Handle has completed")

	close(release)

	assert.Eventually(t, func() bool {
		return sequenceManager.GetSequence() == 5
	}, time.Second, 10*time.Millisecond, "sequence should advance once Handle completes")
}

// TestEventListener_handleEvent_managedAppDemotion exercises the managed-app provisioning lifecycle
// through handleEvent: ManagedApplication events always dispatch to their own app's provisioning
// shard, AccessRequest/Credential events do the same when they carry a reference to that app (and
// otherwise fall back to their kind lane), and deleting the ManagedApplication clears its shard
// assignment.
func TestEventListener_handleEvent_managedAppDemotion(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")

	maKind := management.ManagedApplicationGVK().Kind
	arKind := management.AccessRequestGVK().Kind

	ctx, cancel := context.WithCancelCause(context.Background())
	listener := NewEventListener(
		ctx, cancel,
		make(chan *proto.Event),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{
			maKind: {&mockHandler{}},
			arKind: {&mockHandler{}},
		},
	)
	listener.kindJobs[maKind] = []chan handlerData{make(chan handlerData, 1)}
	listener.kindJobs[arKind] = []chan handlerData{make(chan handlerData, 1)}
	listener.provisioningJobs = []chan handlerData{make(chan handlerData, 1)}

	maEvent := newTestEvent(1)
	maEvent.Payload.Kind = maKind
	maEvent.Payload.Name = "app-1"
	assert.NoError(t, listener.handleEvent(maEvent))
	select {
	case <-listener.provisioningJobs[0]:
	default:
		t.Fatal("expected the ManagedApplication event to use its app's provisioning shard")
	}

	arEventNoRef := newTestEvent(2)
	arEventNoRef.Payload.Kind = arKind
	assert.NoError(t, listener.handleEvent(arEventNoRef))
	select {
	case <-listener.kindJobs[arKind][0]:
	default:
		t.Fatal("expected an AccessRequest without a ManagedApplication reference to fall back to its kind lane")
	}

	arEventWithRef := newTestEvent(3)
	arEventWithRef.Payload.Kind = arKind
	arEventWithRef.Payload.Metadata.References = []*proto.Reference{{Kind: maKind, Name: "app-1"}}
	assert.NoError(t, listener.handleEvent(arEventWithRef))
	select {
	case <-listener.provisioningJobs[0]:
	default:
		t.Fatal("expected an AccessRequest referencing the ManagedApplication to use the app's provisioning shard")
	}

	deleteEvent := newTestEvent(4)
	deleteEvent.Payload.Kind = maKind
	deleteEvent.Payload.Name = "app-1"
	deleteEvent.Type = proto.Event_DELETED
	assert.NoError(t, listener.handleEvent(deleteEvent))
	select {
	case <-listener.provisioningJobs[0]:
	default:
		t.Fatal("expected the ManagedApplication delete to use the app's provisioning shard")
	}
	if _, stillAssigned := listener.provisioningLaneAssignment["app-1"]; stillAssigned {
		t.Fatal("expected the delete to clear the shard assignment")
	}
}

// TestEventListener_dispatchLane_appSharesLaneWithItsAccessRequests proves the ordering guarantee
// the app status guard in accessRequestHandler.onPending relies on: a managed app's own events and
// the events of the AccessRequests/Credentials referencing it resolve to the same shard. A shard has
// exactly one worker, so the app's provisioning - and the status write marking it Success - completes
// before an access request reads that status. When these were split across lanes, the access request
// could observe a not-yet-successful app and fail permanently.
func TestEventListener_dispatchLane_appSharesLaneWithItsAccessRequests(t *testing.T) {
	maKind := management.ManagedApplicationGVK().Kind
	arKind := management.AccessRequestGVK().Kind
	credKind := management.CredentialGVK().Kind

	em := &EventListener{
		provisioningJobs:           make([]chan handlerData, 8),
		provisioningLaneAssignment: make(map[string]int),
	}
	for i := range em.provisioningJobs {
		em.provisioningJobs[i] = make(chan handlerData, 1)
	}

	maEvent := newTestEvent(1)
	maEvent.Payload.Kind = maKind
	maEvent.Payload.Name = "app-1"
	appLane, appLabel := em.dispatchLane(maEvent)

	for _, kind := range []string{arKind, credKind} {
		event := newTestEvent(2)
		event.Payload.Kind = kind
		event.Payload.Name = "resource-for-app-1"
		event.Payload.Metadata.References = []*proto.Reference{{Kind: maKind, Name: "app-1"}}

		lane, label := em.dispatchLane(event)
		assert.True(t, appLane == lane, "%s must share the managed app's shard", kind)
		assert.Equal(t, appLabel, label)
	}

	// other apps still spread across shards - sharing is per app, not global
	spread := false
	for i := 2; i < 27; i++ {
		name := fmt.Sprintf("app-%d", i)

		otherEvent := newTestEvent(int64(i))
		otherEvent.Payload.Kind = maKind
		otherEvent.Payload.Name = name

		lane, _ := em.dispatchLane(otherEvent)
		if lane != appLane {
			spread = true
			break
		}
	}
	assert.True(t, spread, "expected other managed apps to land on other shards")
}

// TestEventListener_provisioningLane proves the two properties the shard-by-key design depends on:
// the same managed app key always lands on the same shard (so its AccessRequest/Credential
// events stay ordered relative to each other, same as when they shared one worker), and
// different keys spread across more than one shard (so different managed apps stop contending
// for the same worker).
func TestEventListener_provisioningLane(t *testing.T) {
	em := &EventListener{
		provisioningJobs:           make([]chan handlerData, 8),
		provisioningLaneAssignment: make(map[string]int),
	}
	for i := range em.provisioningJobs {
		em.provisioningJobs[i] = make(chan handlerData, 1)
	}

	ch, label := em.provisioningLane("app-1")
	for i := 0; i < 5; i++ {
		gotCh, gotLabel := em.provisioningLane("app-1")
		assert.True(t, ch == gotCh, "the same key must always land on the same shard")
		assert.Equal(t, label, gotLabel)
	}

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		_, label := em.provisioningLane(fmt.Sprintf("app-%d", i+2))
		seen[label] = true
	}
	assert.Greater(t, len(seen), 1, "expected demoted keys to spread across more than one shard")
}

// TestEventListener_provisioningShards_runConcurrently proves that once two managed apps are demoted
// to different shards, a slow handler for one app does not delay the other - the reason
// AccessRequest/Credential events were sharded instead of sharing a single worker.
func TestEventListener_provisioningShards_runConcurrently(t *testing.T) {
	slow := &slowHandler{delay: 200 * time.Millisecond}

	shardA := make(chan handlerData, 1)
	shardB := make(chan handlerData, 1)
	go worker(shardA)
	go worker(shardB)

	start := time.Now()
	for _, ch := range []chan handlerData{shardA, shardB} {
		ch <- handlerData{
			event:   newTestEvent(1),
			ctx:     context.Background(),
			handler: slow,
			logger:  log.NewFieldLogger(),
			getEventResource: func(_ *proto.Event, _ []string) (*apiv1.ResourceInstance, error) {
				return &apiv1.ResourceInstance{}, nil
			},
		}
	}

	assert.Eventually(t, func() bool {
		return slow.callCount.Load() == 2
	}, time.Second, 10*time.Millisecond, "both shards should have run their handler")
	assert.Less(t, time.Since(start), 350*time.Millisecond,
		"both shards' handlers should run concurrently, not serialize behind one worker")
}

// TestKindLane_ShardsByResourceID proves kindLane's routing property: a given resource ID always
// maps to the same worker channel (so a lane behaves as if single-threaded for that resource), and
// different resource IDs spread across more than one worker when there's more than one shard.
func TestKindLane_ShardsByResourceID(t *testing.T) {
	const kind = "TestKind"
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	em := NewEventListener(
		ctx, cancel,
		make(chan *proto.Event),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{kind: {&mockHandler{}}},
	)
	lanes := make([]chan handlerData, 4)
	for i := range lanes {
		lanes[i] = make(chan handlerData, 1)
	}
	em.kindJobs[kind] = lanes

	lane := em.kindLane(kind, "resource-a")
	for i := 0; i < 50; i++ {
		assert.True(t, lane == em.kindLane(kind, "resource-a"), "the same resource ID must always land on the same worker")
	}

	seen := map[chan handlerData]bool{}
	for i := 0; i < 50; i++ {
		seen[em.kindLane(kind, fmt.Sprintf("resource-%d", i))] = true
	}
	assert.Greater(t, len(seen), 1, "different resource IDs should spread across more than one worker")
}

// orderTrackingHandler records the sequence ID handled for each resource ID and fails the test if
// a later call for the same resource ID reports a sequence that isn't strictly increasing - i.e. if
// two events for the same resource were ever handled out of order.
type orderTrackingHandler struct {
	t       *testing.T
	wg      *sync.WaitGroup
	mu      sync.Mutex
	lastSeq map[string]int64
}

func (h *orderTrackingHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool { return true }

func (h *orderTrackingHandler) Handle(_ context.Context, meta *proto.EventMeta, ri *apiv1.ResourceInstance) error {
	defer h.wg.Done()
	// jitter simulates handlers taking varying amounts of time, e.g. one blocked on a network
	// call while another for the same resource is dispatched to a different worker - the
	// scenario that let events for the same resource complete out of order pre-sharding.
	time.Sleep(time.Duration(rand.Intn(300)) * time.Microsecond)

	id := ri.Metadata.ID
	seq := meta.SequenceID

	h.mu.Lock()
	defer h.mu.Unlock()
	if prev, ok := h.lastSeq[id]; ok && seq <= prev {
		h.t.Errorf("resource %s: handled sequence %d out of order after %d", id, seq, prev)
	}
	h.lastSeq[id] = seq
	return nil
}

// TestEventListener_kindLanePreservesPerResourceOrder is the integration-level proof for the
// sharding fix: with multiple workers per kind lane, events for the same resource ID are still
// always handled by the same worker and therefore in dispatch order, ruling out both the
// concurrent-map-write crash and silent out-of-order overwrites for a single resource - while
// different resources are still free to run in parallel across the other workers.
func TestEventListener_kindLanePreservesPerResourceOrder(t *testing.T) {
	const kind = "TestKind"
	const resourceCount = 8
	const eventsPerResource = 300

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	var wg sync.WaitGroup
	wg.Add(resourceCount * eventsPerResource)
	tracker := &orderTrackingHandler{t: t, wg: &wg, lastSeq: map[string]int64{}}

	em := NewEventListener(
		ctx, cancel,
		make(chan *proto.Event),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{kind: {tracker}},
		WithRegularWorkerCount(4),
	)
	em.Listen()
	defer em.Stop()

	seq := int64(0)
	for round := 0; round < eventsPerResource; round++ {
		for r := 0; r < resourceCount; r++ {
			seq++
			event := newTestEvent(seq)
			event.Type = proto.Event_DELETED
			event.Payload.Kind = kind
			event.Payload.Metadata.Id = fmt.Sprintf("resource-%d", r)
			assert.NoError(t, em.handleEvent(event))
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("handlers did not finish in time")
	}
}

type mockHandler struct {
	err error
}

func (m *mockHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *apiv1.ResourceInstance) error {
	return m.err
}

func (m *mockHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool {
	return true
}

// slowHandler blocks for the given duration each time Handle is called,
// and atomically increments callCount.
type slowHandler struct {
	delay     time.Duration
	callCount atomic.Int32
}

func (h *slowHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *apiv1.ResourceInstance) error {
	time.Sleep(h.delay)
	h.callCount.Add(1)
	return nil
}

func (h *slowHandler) ShouldHandle(_ context.Context, _ *proto.Event) bool {
	return true
}

func newTestEvent(seqID int64) *proto.Event {
	return &proto.Event{
		Type: proto.Event_CREATED,
		Payload: &proto.ResourceInstance{
			Metadata: &proto.Metadata{
				SelfLink: "/management/v1/watchtopics/mock-watch-topic",
			},
		},
		Metadata: &proto.EventMeta{
			SequenceID: seqID,
		},
	}
}
