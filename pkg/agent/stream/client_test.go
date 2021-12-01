package stream

import (
	"fmt"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/stretchr/testify/assert"
)

var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"

func TestClient(t *testing.T) {
	tests := []struct {
		name      string
		statusErr bool
		err       error
		hasErr    bool
	}{
		{
			name:      "should return an OK status on the healthcheck",
			statusErr: true,
			err:       nil,
			hasErr:    false,
		},
		{
			name:      "should return a FAIL status on the healthcheck",
			statusErr: false,
			err:       nil,
			hasErr:    false,
		},
		{
			name:      "should handle an error from the manager",
			statusErr: true,
			err:       fmt.Errorf("error"),
			hasErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(
				topic,
				&mockManager{
					err:    tc.err,
					status: tc.statusErr,
				},
				&mockListener{},
				make(chan *proto.Event),
			)

			err := c.Start()
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			statusErr := c.HealthCheck()

			if tc.statusErr {
				assert.Nil(t, statusErr)
			} else {
				assert.NotNil(t, statusErr)
			}
		})
	}
}

type mockManager struct {
	err    error
	status bool
}

func (m mockManager) RegisterWatch(_ string, _ chan *proto.Event, _ chan error) (string, error) {
	return "", m.err
}

func (m mockManager) CloseWatch(_ string) error {
	return nil
}

func (m mockManager) Close() {
}

func (m mockManager) Status() bool {
	return m.status
}

type mockEventManager struct{}

func (m mockEventManager) Listen() error {
	return nil
}

func mockNewEventManager(_ chan *proto.Event, _ chan interface{}, _ ResourceClient, _ ...Handler) Listener {
	return &mockEventManager{}
}

type mockResourceClient struct {
}

func (m mockResourceClient) Get(_ string) (*apiv1.ResourceInstance, error) {
	return nil, nil
}

type mockListener struct {
}

func (m mockListener) Listen() error {
	return nil
}
