package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/stretchr/testify/assert"
)

var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"

func TestClient(t *testing.T) {
	tests := []struct {
		name        string
		statusErr   bool
		err         error
		hasErr      bool
		listenerErr error
	}{
		{
			name:        "should not return an error when calling HealthCheck",
			statusErr:   true,
			err:         nil,
			hasErr:      false,
			listenerErr: nil,
		},
		{
			name:        "should return an error when calling HealthCheck",
			statusErr:   false,
			err:         nil,
			hasErr:      false,
			listenerErr: nil,
		},
		{
			name:        "should handle an error from the manager",
			statusErr:   true,
			err:         fmt.Errorf("error"),
			hasErr:      true,
			listenerErr: nil,
		},
		{
			name:        "should handle an error from the listener",
			statusErr:   true,
			err:         nil,
			hasErr:      true,
			listenerErr: fmt.Errorf("error"),
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
				&mockListener{
					err: tc.listenerErr,
				},
				make(chan *proto.Event),
			)

			err := c.Start()
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			statusErr := c.Status()

			if tc.statusErr {
				assert.Nil(t, statusErr)
			} else {
				assert.NotNil(t, statusErr)
			}
		})
	}
}

func TestClientStreamJob(t *testing.T) {
	s := &mockStreamer{}
	stopCh := make(chan interface{})
	j := NewClientStreamJob(s, stopCh)

	assert.Nil(t, j.Status())
	assert.True(t, j.Ready())
	assert.Nil(t, j.Execute())
}

type mockStreamer struct {
	hcErr    error
	startErr error
}

func (m mockStreamer) Start() error {
	return m.startErr
}

func (m mockStreamer) Status() error {
	return m.hcErr
}

func (m mockStreamer) Stop() {
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

func (m mockManager) CloseAll() {
}

func (m mockManager) CloseConn() {
}

func (m mockManager) Status() bool {
	return m.status
}

type mockEventManager struct{}

func (m mockEventManager) Listen() error {
	return nil
}

type mockListener struct {
	err error
}

func (m mockListener) Listen() error {
	return m.err
}

func (m mockListener) Stop() {
}
