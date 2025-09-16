package events

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

func TestEventListener_start(t *testing.T) {
	tests := []struct {
		name      string
		hasError  bool
		events    chan *proto.Event
		client    APIClient
		handler   handler.Handler
		writeStop bool
	}{
		{
			name:     "should start without an error",
			hasError: false,
			events:   make(chan *proto.Event),
			client:   &mockAPIClient{},
			handler:  &mockHandler{},
		},
		{
			name:     "should return an error when the event channel is closed",
			hasError: true,
			events:   make(chan *proto.Event),
			client:   &mockAPIClient{},
			handler:  &mockHandler{},
		},

		{
			name:     "should return an error, when the request for a ResourceClient fails",
			hasError: true,
			events:   make(chan *proto.Event),
			client:   &mockAPIClient{getErr: fmt.Errorf("failed")},
			handler:  &mockHandler{},
		},
		{
			name:     "should not return an error, even if a handler fails to process an event",
			hasError: false,
			events:   make(chan *proto.Event),
			client:   &mockAPIClient{},
			handler:  &mockHandler{err: fmt.Errorf("failed")},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			listener := NewEventListener(ctx, cancel, tc.events, tc.client, sequenceManager, tc.handler)

			errCh := make(chan error)
			go func() {
				_, err := listener.start()
				errCh <- err
			}()

			if tc.hasError == false {
				tc.events <- &proto.Event{
					Type: proto.Event_CREATED,
					Payload: &proto.ResourceInstance{
						Metadata: &proto.Metadata{
							SelfLink: "/management/v1alpha1/watchtopics/mock-watch-topic",
						},
					},
					Metadata: &proto.EventMeta{
						SequenceID: 1,
					},
				}
			} else {
				close(tc.events)
			}

			err := <-errCh
			if tc.hasError == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}

// Should call Listen and handle a graceful stop, and an error
func TestEventListener_Listen(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := NewSequenceProvider(cacheManager, "testWatch")
	events := make(chan *proto.Event)
	ctx, cancel := context.WithCancel(context.Background())
	listener := NewEventListener(ctx, cancel, events, &mockAPIClient{}, sequenceManager, &mockHandler{})
	listener.Listen()
	listener.Stop()
	err := ctx.Err()
	assert.NotNil(t, err)

	ctx, cancel = context.WithCancel(context.Background())
	listener = NewEventListener(ctx, cancel, events, &mockAPIClient{}, sequenceManager, &mockHandler{})
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
						SelfLink: "/management/v1alpha1/watchtopics/mock-watch-topic",
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

			ctx, cancel := context.WithCancel(context.Background())
			listener := NewEventListener(ctx, cancel, make(chan *proto.Event), tc.client, sequenceManager, tc.handler)

			err := listener.handleEvent(event)

			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

type mockHandler struct {
	err error
}

func (m *mockHandler) Handle(_ context.Context, _ *proto.EventMeta, _ *apiv1.ResourceInstance) error {
	return m.err
}
