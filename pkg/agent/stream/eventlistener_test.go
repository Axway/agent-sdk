package stream

import (
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
		ri        ResourceClient
		handler   handler.Handler
		writeStop bool
	}{
		{
			name:     "should start without an error",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  &mockHandler{},
		},
		{
			name:     "should return an error when the event channel is closed",
			hasError: true,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  &mockHandler{},
		},

		{
			name:     "should not return an error, even if the request for a ResourceClient fails",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{err: fmt.Errorf("failed")},
			handler:  &mockHandler{},
		},
		{
			name:     "should not return an error, even if a handler fails to process an event",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  &mockHandler{err: fmt.Errorf("failed")},
		},
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := newAgentSequenceManager(cacheManager, "testWatch")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listener := NewEventListener(tc.events, tc.ri, sequenceManager, tc.handler)

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
	sequenceManager := newAgentSequenceManager(cacheManager, "testWatch")
	events := make(chan *proto.Event)
	listener := NewEventListener(events, &mockRI{}, sequenceManager, &mockHandler{})
	errCh := listener.Listen()
	go listener.Stop()
	err := <-errCh
	assert.Nil(t, err)

	listener = NewEventListener(events, &mockRI{}, sequenceManager, &mockHandler{})
	errCh = listener.Listen()
	close(events)
	err = <-errCh
	assert.NotNil(t, err)
}

func TestEventListener_handleEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    proto.Event_Type
		hasError bool
		ri       ResourceClient
		handler  handler.Handler
	}{
		{
			name:     "should process a delete event with no error",
			event:    proto.Event_DELETED,
			hasError: false,
			ri:       &mockRI{},
			handler:  &mockHandler{},
		},
		{
			name:     "should return an error when the request to get a ResourceClient fails",
			event:    proto.Event_CREATED,
			hasError: true,
			ri:       &mockRI{err: fmt.Errorf("err")},
			handler:  &mockHandler{},
		},
		{
			name:     "should get a ResourceClient, and process a create event",
			event:    proto.Event_CREATED,
			hasError: false,
			ri:       &mockRI{},
			handler:  &mockHandler{},
		},
		{
			name:     "should get a ResourceClient, and process an update event",
			event:    proto.Event_UPDATED,
			hasError: false,
			ri:       &mockRI{},
			handler:  &mockHandler{},
		},
	}
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sequenceManager := newAgentSequenceManager(cacheManager, "testWatch")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &proto.Event{
				Type: tc.event,
				Payload: &proto.ResourceInstance{
					Metadata: &proto.Metadata{
						SelfLink: "/management/v1alpha1/watchtopics/mock-watch-topic",
					},
				},
				Metadata: &proto.EventMeta{
					SequenceID: 1,
				},
			}

			listener := NewEventListener(make(chan *proto.Event), tc.ri, sequenceManager, tc.handler)

			err := listener.handleEvent(event)

			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

type mockTokenGetter struct {
	token string
	err   error
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return m.token, m.err
}

type mockRI struct {
	err error
}

func (m mockRI) Create(_ string, _ []byte) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

func (m mockRI) Get(_ string) (*apiv1.ResourceInstance, error) {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Kind: "kind",
				},
			},
			Name:  "name",
			Title: "title",
		},
		Owner: nil,
		Spec:  nil,
	}, m.err
}

type mockHandler struct {
	err error
}

func (m *mockHandler) Handle(_ proto.Event_Type, _ *proto.EventMeta, _ *apiv1.ResourceInstance) error {
	return m.err
}
