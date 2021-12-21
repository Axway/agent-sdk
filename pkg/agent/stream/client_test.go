package stream

import (
	"encoding/json"
	"fmt"
	"testing"

	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/stretchr/testify/assert"
)

var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"

var cfg = &config.CentralConfiguration{
	AgentType:     1,
	TenantID:      "12345",
	Environment:   "stream-test",
	EnvironmentID: "123",
	AgentName:     "discoveryagents",
	URL:           "http://abc.com",
	TLS:           &config.TLSConfiguration{},
}

// should create a new streamer and call Start
func TestNewStreamer(t *testing.T) {
	getToken := &mockTokenGetter{}
	httpClient := &api.MockHTTPClient{}
	wt := &mv1.WatchTopic{}
	bts, _ := json.Marshal(wt)
	httpClient.Response = &api.Response{
		Code:    200,
		Body:    bts,
		Headers: nil,
	}
	c, err := NewStreamer(httpClient, cfg, getToken)
	assert.NotNil(t, c)
	assert.Nil(t, err)

	streamer := c.(*streamer)
	manager := &mockManager{status: true}
	streamer.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.NotNil(t, streamer.Status())

	errCh := make(chan error)
	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	for streamer.listener == nil {
		continue
	}
	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err = <-errCh
	assert.Nil(t, err)

	streamer.manager = nil
	streamer.listener = nil

	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	for streamer.manager == nil {
		continue
	}

	assert.Nil(t, streamer.Status())

	// should stop the listener and write an error from the manager to the error channel
	go streamer.Stop()
	err = <-errCh
	assert.NotNil(t, err)

	manager.status = false

	assert.NotNil(t, streamer.Status())
}

func TestClientStreamJob(t *testing.T) {
	s := &mockStreamer{}
	stopCh := make(chan interface{})
	j := NewClientStreamJob(s, stopCh)

	assert.Nil(t, j.Status())
	assert.True(t, j.Ready())
	assert.Nil(t, j.Execute())
}

func Test_getAgentSequenceManager(t *testing.T) {
	wtName := "fake"
	sm := getAgentSequenceManager(wtName)
	assert.Equal(t, sm.GetSequence(), int64(0))

	sm = getAgentSequenceManager("")
	assert.Equal(t, sm.GetSequence(), int64(0))
}

func Test_getWatchTopic(t *testing.T) {
	wt, err := getWatchTopic(cfg, &mockRI{})
	assert.NotNil(t, wt)
	assert.Nil(t, err)

	wt, err = getWatchTopic(cfg, &mockRI{err: fmt.Errorf("error")})
	assert.NotNil(t, wt)
	assert.Nil(t, err)
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
	status bool
	errCh  chan error
}

func (m *mockManager) RegisterWatch(_ string, _ chan *proto.Event, errCh chan error) (string, error) {
	m.errCh = errCh
	return "", nil
}

func (m *mockManager) CloseWatch(_ string) error {
	m.errCh <- fmt.Errorf("manager error")
	return nil
}

func (m *mockManager) CloseConn() {
	m.errCh <- fmt.Errorf("manager error")
}

func (m *mockManager) Status() bool {
	return m.status
}

type mockListener struct {
	errCh chan error
}

func (m *mockListener) Listen() chan error {
	return m.errCh
}

func (m *mockListener) Stop() {
}
