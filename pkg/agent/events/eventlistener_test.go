package events

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/config"

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
		make(chan *proto.Event, 1),
		&mockAPIClient{},
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{
			knownKind:                               {&mockHandler{}},
			management.ManagedApplicationGVK().Kind: {&mockHandler{}},
		},
		WithContextAndCancel(ctx, cancel),
	)
	em.kindJobs = map[string]chan handlerData{
		knownKind:                               make(chan handlerData, 1),
		management.ManagedApplicationGVK().Kind: make(chan handlerData, 1),
	}
	em.lowPriorityJobs = make(chan handlerData, 1)
	return em
}

func TestEventListener_start(t *testing.T) {
	const knownKind = "TestKind"
	lowKind := management.ManagedApplicationGVK().Kind

	tests := []struct {
		name        string
		kind        string
		alreadySeen bool // pre-seed seenManagedApps for the managed app before dispatching
		closeSource bool
		cancelCtx   bool
		wantDone    bool
		wantErr     bool
		wantLane    string // "kind", "lowPriority", or "" for no dispatch at all
	}{
		{
			name:     "dispatches a known kind to its own kind lane",
			kind:     knownKind,
			wantLane: "kind",
		},
		{
			name:     "dispatches the first event for a managed app to its own kind lane",
			kind:     lowKind,
			wantLane: "kind",
		},
		{
			name:        "a later ManagedApplication event for an already-seen app still uses its own kind lane",
			kind:        lowKind,
			alreadySeen: true,
			wantLane:    "kind",
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
				if tc.alreadySeen {
					em.seenManagedApps["app-1"] = struct{}{}
				}
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
				lane = em.kindJobs[tc.kind]
			case "lowPriority":
				lane = em.lowPriorityJobs
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
	listener := NewEventListener(events, &mockAPIClient{}, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {&mockHandler{}}}, WithContextAndCancel(ctx, cancel))
	listener.Listen()
	listener.Stop()
	err := ctx.Err()
	assert.NotNil(t, err)

	ctx, cancel = context.WithCancelCause(context.Background())
	listener = NewEventListener(events, &mockAPIClient{}, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {&mockHandler{}}}, WithContextAndCancel(ctx, cancel))
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
			name:     "should return an error when the request to get a ResourceClient fails",
			event:    proto.Event_CREATED,
			hasError: true,
			client:   &mockAPIClient{getErr: fmt.Errorf("err")},
			handler:  &mockHandler{},
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
			listener := NewEventListener(make(chan *proto.Event), tc.client, "https://apicentral.example.com", sequenceManager, map[string][]handler.Handler{"": {tc.handler}}, WithContextAndCancel(ctx, cancel))
			listener.kindJobs[""] = make(chan handlerData, 1)

			err := listener.handleEvent(event)

			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func Test_managedApplicationKey(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		appName string
		spec    map[string]interface{}
		wantKey string
		wantOK  bool
	}{
		{
			name:    "managed application uses its own name",
			kind:    management.ManagedApplicationGVK().Kind,
			appName: "app-1",
			wantKey: "app-1",
			wantOK:  true,
		},
		{
			name:    "access request reads managedApplication from spec",
			kind:    management.AccessRequestGVK().Kind,
			spec:    map[string]interface{}{"managedApplication": "app-2"},
			wantKey: "app-2",
			wantOK:  true,
		},
		{
			name:    "credential reads managedApplication from spec",
			kind:    management.CredentialGVK().Kind,
			spec:    map[string]interface{}{"managedApplication": "app-3"},
			wantKey: "app-3",
			wantOK:  true,
		},
		{
			name: "access request with no spec (e.g. a delete) is undeterminable",
			kind: management.AccessRequestGVK().Kind,
		},
		{
			name: "credential with an empty managedApplication is undeterminable",
			kind: management.CredentialGVK().Kind,
			spec: map[string]interface{}{"managedApplication": ""},
		},
		{
			name: "unrelated kind is not applicable",
			kind: "SomeOtherKind",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := newTestEvent(1)
			event.Payload.Kind = tc.kind
			event.Payload.Name = tc.appName

			var ri *apiv1.ResourceInstance
			if tc.spec != nil {
				ri = &apiv1.ResourceInstance{Spec: tc.spec}
			}

			key, ok := managedApplicationKey(event, ri)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantKey, key)
		})
	}
}

func TestEventListener_dispatchLane(t *testing.T) {
	const knownKind = "TestKind"
	em := newTestListener(t, knownKind)

	appRI := &apiv1.ResourceInstance{Spec: map[string]interface{}{"managedApplication": "app-1"}}

	event := newTestEvent(1)
	event.Payload.Kind = knownKind
	kindJob, _ := em.dispatchLane(event, nil)
	assert.True(t, em.kindJobs[knownKind] == kindJob)

	maEvent := newTestEvent(2)
	maEvent.Payload.Kind = management.ManagedApplicationGVK().Kind
	maEvent.Payload.Name = "app-1"
	kindJob, _ = em.dispatchLane(maEvent, nil)
	assert.True(t, em.kindJobs[maEvent.Payload.Kind] == kindJob)
	if _, seen := em.seenManagedApps["app-1"]; !seen {
		t.Fatal("expected the first event for app-1 to mark it as seen")
	}

	arEvent := newTestEvent(3)
	arEvent.Payload.Kind = management.AccessRequestGVK().Kind
	kindJob, _ = em.dispatchLane(arEvent, appRI)
	assert.True(t, em.lowPriorityJobs == kindJob)

	credEvent := newTestEvent(4)
	credEvent.Payload.Kind = management.CredentialGVK().Kind
	kindJob, _ = em.dispatchLane(credEvent, &apiv1.ResourceInstance{})
	assert.True(t, em.lowPriorityJobs == kindJob)
}

// TestEventListener_handleEvent_managedAppDemotion exercises the full managed-app demotion
// lifecycle through handleEvent: the first event for an app uses its kind lane, later events
// for the same app are demoted to the low priority lane, and deleting the ManagedApplication
// resets it so the next related event is treated as first-seen again.
func TestEventListener_handleEvent_managedAppDemotion(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")

	maKind := management.ManagedApplicationGVK().Kind
	arKind := management.AccessRequestGVK().Kind

	client := &mockAPIClient{ri: &apiv1.ResourceInstance{
		Spec: map[string]interface{}{"managedApplication": "app-1"},
	}}

	ctx, cancel := context.WithCancelCause(context.Background())
	listener := NewEventListener(
		make(chan *proto.Event),
		client,
		"https://apicentral.example.com",
		sequenceManager,
		map[string][]handler.Handler{
			maKind: {&mockHandler{}},
			arKind: {&mockHandler{}},
		},
		WithContextAndCancel(ctx, cancel),
	)
	listener.kindJobs[maKind] = make(chan handlerData, 1)
	listener.kindJobs[arKind] = make(chan handlerData, 1)
	listener.lowPriorityJobs = make(chan handlerData, 1)

	maEvent := newTestEvent(1)
	maEvent.Payload.Kind = maKind
	maEvent.Payload.Name = "app-1"
	assert.NoError(t, listener.handleEvent(maEvent))
	select {
	case <-listener.kindJobs[maKind]:
	default:
		t.Fatal("expected the first ManagedApplication event to use its kind lane")
	}

	arEvent := newTestEvent(2)
	arEvent.Payload.Kind = arKind
	assert.NoError(t, listener.handleEvent(arEvent))
	select {
	case <-listener.lowPriorityJobs:
	default:
		t.Fatal("expected the AccessRequest for an already-seen app to use the low priority lane")
	}

	deleteEvent := newTestEvent(3)
	deleteEvent.Payload.Kind = maKind
	deleteEvent.Payload.Name = "app-1"
	deleteEvent.Type = proto.Event_DELETED
	assert.NoError(t, listener.handleEvent(deleteEvent))
	select {
	case <-listener.kindJobs[maKind]:
	default:
		t.Fatal("expected the ManagedApplication delete to still use its own kind lane - only AccessRequest/Credential get demoted")
	}
	if _, stillSeen := listener.seenManagedApps["app-1"]; stillSeen {
		t.Fatal("expected the delete to clear the seen entry")
	}

	arEvent2 := newTestEvent(4)
	arEvent2.Payload.Kind = arKind
	assert.NoError(t, listener.handleEvent(arEvent2))
	select {
	case <-listener.kindJobs[arKind]:
	default:
		t.Fatal("expected the AccessRequest after the app was deleted to use the kind lane again")
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
