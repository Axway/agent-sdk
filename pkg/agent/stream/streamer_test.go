package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"

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
