package stream

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

var url = "https://tjohnson.dev.ampc.axwaytest.net/apis"
var tenantID = "426937327920148"

func TestEventManager_start(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		events   chan *proto.Event
		ri       ResourceGetter
		handler  Handler
	}{
		{
			name:     "should start without an error",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  mockHandler{},
		},
		{
			name:     "should return an error when the event channel is closed",
			hasError: true,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  mockHandler{},
		},
		{
			name:     "should not return an error, even if the request for a resource fails",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{err: fmt.Errorf("failed")},
			handler:  mockHandler{},
		},
		{
			name:     "should not return an error, even if a handler fails to process an event",
			hasError: false,
			events:   make(chan *proto.Event),
			ri:       &mockRI{},
			handler:  mockHandler{err: fmt.Errorf("failed")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEventManager(tc.events, tc.ri, tc.handler)

			errCh := make(chan error)
			go func() {
				err := em.start()
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

	events := make(chan *proto.Event)
	em := NewEventManager(events, &mockRI{}, &mockHandler{})

	errCh := make(chan error)
	go func() {
		err := em.start()
		errCh <- err
	}()

	events <- &proto.Event{
		Type: proto.Event_CREATED,
		Payload: &proto.ResourceInstance{
			Metadata: &proto.Metadata{
				SelfLink: "/management/v1alpha1/watchtopics/mock-watch-topic",
			},
		},
	}
	err := <-errCh
	assert.Nil(t, err)
}

func TestEventManager_handleEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    proto.Event_Type
		hasError bool
		ri       ResourceGetter
		handler  Handler
	}{
		{
			name:     "should process a delete event with no error",
			event:    proto.Event_DELETED,
			hasError: false,
			ri:       &mockRI{},
			handler:  mockHandler{},
		},
		{
			name:     "should return an error when the request to get a resource fails",
			event:    proto.Event_CREATED,
			hasError: true,
			ri:       &mockRI{err: fmt.Errorf("err")},
			handler:  mockHandler{},
		},
		{
			name:     "should get a resource, and process a create event",
			event:    proto.Event_CREATED,
			hasError: false,
			ri:       &mockRI{},
			handler:  mockHandler{},
		},
		{
			name:     "should get a resource, and process an update event",
			event:    proto.Event_UPDATED,
			hasError: false,
			ri:       &mockRI{},
			handler:  mockHandler{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := &proto.Event{
				Type: tc.event,
				Payload: &proto.ResourceInstance{
					Metadata: &proto.Metadata{
						SelfLink: "/management/v1alpha1/watchtopics/mock-watch-topic",
					},
				},
			}

			em := NewEventManager(make(chan *proto.Event), tc.ri, tc.handler)
			err := em.handleEvent(event)

			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestEventHandlerOld(t *testing.T) {
	t.Skip()
	events := make(chan *proto.Event)
	apis, categories, instances := cache.New(), cache.New(), cache.New()

	client := api.NewClient(nil, "")
	c := NewResourceClient(url, client, &mockTokenGetter{}, tenantID)

	sc := NewEventManager(
		events,
		c,
		NewAPISvcHandler(apis),
		NewCategoryHandler(categories),
		NewInstanceHandler(instances),
	)

	go func() {
		err := sc.Start()
		if err != nil {
			logrus.Errorf("stream cache error: %s", err)
			os.Exit(1)
		}
	}()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	filePath := path.Join(dir, "event.json")
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	event := &proto.Event{}
	err = json.Unmarshal(data, event)
	if err != nil {
		t.Fatal(err)
	}

	events <- event
	time.Sleep(60 * time.Minute)
}

type mockTokenGetter struct {
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzY3NTk1MDUsImlhdCI6MTYzNjc1NTkwNSwiYXV0aF90aW1lIjoxNjM2NzUxNjYwLCJqdGkiOiI0NjBkMjU0ZC0yOTE1LTRhYzUtYTRmOS1jOTgwYjY5NTgwM2UiLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiJjYmEyNzVmMC01MjZjLTQxZGEtODc3Yy1hMWY1NWJlOGIzYTciLCJzZXNzaW9uX3N0YXRlIjoiMTJmOTQ5NzEtMzhiMC00MTgwLTkzZGMtNDVmY2M0NzY1MzZmIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM2NzU1OTA1MzQyRTEyLCJuYW1lIjoiVHJldm9yIEpvaG5zb24iLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0am9obnNvbkBheHdheS5jb20iLCJnaXZlbl9uYW1lIjoiVHJldm9yIiwiZmFtaWx5X25hbWUiOiJKb2huc29uIiwiZW1haWwiOiJ0am9obnNvbkBheHdheS5jb20ifQ.OsFvBvp4YNiWszGnY34C8dJN7gQE0FQjdYOXv7q57j2QT19kMIjpEHB4OYvkCVYLSfWT0BXFZoKJj1CQUYhpv-72dnBot_kC-3l5TVbMNvAfsfK5s6E_mxgD7lR15qK9spp8kjMRyjwsk0QJ2W1jGg5FbZyidTu0-m-QOFIeJYfZaCMHyTQ17OZxpYZfVaPT7nMwZofaXqzzL3iX9P8xPoYHIX5Z486F-dHPSGv2Kp_Wa7rLM5Z7YTsAZMJB-tvsE9J0FQSv_rsBUVWCcsa69xGwm1cC5z8EiDQDPb_I9CRJXKg-CXFwepoK8H90WCPakB2LoE9MUV-Wab5wf-phLA", nil
}

type mockRI struct {
	err error
}

func (m mockRI) Get(_ string) (*apiv1.ResourceInstance, error) {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{},
		Owner:        nil,
		Spec:         nil,
	}, m.err
}

type mockHandler struct {
	err error
}

func (m mockHandler) callback(action proto.Event_Type, resource *apiv1.ResourceInstance) error {
	return m.err
}
