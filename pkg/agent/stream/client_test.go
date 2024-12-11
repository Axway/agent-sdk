package stream

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/apic/mock"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/stretchr/testify/assert"
)

func NewConfig() *config.CentralConfiguration {
	return &config.CentralConfiguration{
		AgentType:     1,
		TenantID:      "12345",
		Environment:   "stream-test",
		EnvironmentID: "123",
		AgentName:     "discoveryagents",
		URL:           "http://abc.com",
		TLS:           &config.TLSConfiguration{},
		SingleURL:     "https://abc.com",
	}
}

// should create a new streamer and call Start
func TestNewStreamer(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &management.WatchTopic{}
	httpClient := &mockAPIClient{}
	cfg := NewConfig()
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)

	streamer, err := NewStreamerClient(
		httpClient,
		cfg,
		getToken,
		nil,
		WithOnStreamConnection(),
		WithCacheManager(cacheManager),
		WithWatchTopic(wt),
	)
	assert.NotNil(t, streamer)
	assert.Nil(t, err)

	manager := &mockManager{
		status:  true,
		readyCh: make(chan struct{}),
	}

	streamer.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.Nil(t, streamer.Status())

	errCh := make(chan error)
	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	<-manager.readyCh

	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err = <-errCh
	assert.Nil(t, err)

	assert.Equal(t, hc.OK, hc.RunChecks())
	streamer.manager = nil
	streamer.listener = nil

	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	<-manager.readyCh

	assert.Nil(t, streamer.Status())
	stop(t, streamer, errCh)
	manager.status = false

	assert.NotNil(t, streamer.Status())
	assert.Equal(t, hc.FAIL, hc.RunChecks())
}

func TestClientOptions(t *testing.T) {
	sequence := &mockSequence{}
	sequence.SetSequence(1)
	sc, _ := NewStreamerClient(
		&mock.Client{},
		NewConfig(),
		&mockTokenGetter{},
		nil,
		WithHarvester(&mockHarvester{}, sequence),
		WithEventSyncError(func() {
		}),
		WithOnStreamConnection(),
	)
	assert.NotNil(t, sc.harvester)
	assert.NotNil(t, sc.sequence)
	assert.NotNil(t, sc.onEventSyncError)
	assert.NotNil(t, sc.onStreamConnection)
}

func TestStatusUpdates(t *testing.T) {
	cases := map[string]struct {
		queueActive  bool
		state        string
		prevState    string
		stateMessage string
		expectedReq  *proto.Request
		expectedErr  bool
	}{
		"error on status update with inactive queue": {
			state:       "unhealthy",
			expectedErr: true,
		},
		"no stopped state update with stream client": {
			state:       "stopped",
			queueActive: true,
		},
		"no status update with same state change": {
			state:       "unhealthy",
			prevState:   "unhealthy",
			queueActive: true,
		},
		"status update from running to unhealthy": {
			state:       "unhealthy",
			prevState:   "running",
			queueActive: true,
			expectedReq: &proto.Request{
				RequestType: proto.RequestType_AGENT_STATUS.Enum(),
				AgentStatus: &proto.AgentStatus{
					State: "unhealthy",
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streamer, err := NewStreamerClient(
				&mockAPIClient{},
				NewConfig(),
				&mockTokenGetter{},
				nil,
			)

			assert.NotNil(t, streamer)
			assert.Nil(t, err)

			manager := &mockManager{
				status:  true,
				readyCh: make(chan struct{}),
			}
			requestQueue := &mockRequestQueue{active: tc.queueActive}

			streamer.newRequestQueue = func(requestCh chan *proto.Request) events.RequestQueue {
				return requestQueue
			}
			streamer.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
				return manager, nil
			}

			assert.Nil(t, streamer.Status())
			errCh := make(chan error)
			go func() {
				err := streamer.Start()
				errCh <- err
			}()
			defer func() {
				streamer.Stop()
				err = <-errCh
				assert.Nil(t, err)
			}()

			// wait for stream ready
			<-manager.readyCh

			err = streamer.UpdateAgentStatus(tc.state, tc.prevState, tc.stateMessage)
			if tc.expectedErr {
				assert.NotNil(t, err)
				return
			}

			if tc.expectedReq != nil {
				assert.NotNil(t, requestQueue.request)
				assert.Equal(t, tc.expectedReq.RequestType.Enum(), requestQueue.request.RequestType.Enum())
				assert.Equal(t, tc.expectedReq.AgentStatus.State, requestQueue.request.AgentStatus.State)
				return
			}
			assert.Nil(t, requestQueue.request)
		})
	}
}

func stop(t *testing.T, streamer *StreamerClient, errCh chan error) {
	t.Helper()
	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err := <-errCh
	assert.Nil(t, err)
}

type mockManager struct {
	status  bool
	readyCh chan struct{}
}

func (m *mockManager) RegisterWatch(_ string, _ chan *proto.Event, _ chan error) (string, error) {
	if m.readyCh != nil {
		m.readyCh <- struct{}{}
	}
	return "", nil
}

func (m *mockManager) CloseWatch(_ string) error {
	return nil
}

func (m *mockManager) CloseConn() {
}

func (m *mockManager) Status() bool {
	return m.status
}

type mockAPIClient struct {
	resource    *apiv1.ResourceInstance
	getErr      error
	createErr   error
	deleteErr   error
	paged       []*apiv1.ResourceInstance
	pagedCalled bool
	pagedErr    error
}

func (m mockAPIClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return m.resource, m.getErr
}

func (m mockAPIClient) CreateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return nil, m.createErr
}

func (m mockAPIClient) DeleteResourceInstance(_ apiv1.Interface) error {
	return m.deleteErr
}

func (m *mockAPIClient) GetAPIV1ResourceInstances(map[string]string, string) ([]*apiv1.ResourceInstance, error) {
	m.pagedCalled = true
	return m.paged, m.pagedErr
}

type mockTokenGetter struct {
	token string
	err   error
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return m.token, m.err
}

type mockHarvester struct{}

func (m mockHarvester) EventCatchUp(link string, events chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 0, nil
}

type mockSequence struct{}

func (m mockSequence) GetSequence() int64 { return 0 }

func (m mockSequence) SetSequence(sequenceID int64) {}

type mockRequestQueue struct {
	active  bool
	request *proto.Request
}

func (m *mockRequestQueue) Start() {

}

func (m *mockRequestQueue) Write(request *proto.Request) error {
	m.request = request
	return nil
}

func (m *mockRequestQueue) Stop() {

}

func (m *mockRequestQueue) IsActive() bool {
	return m.active
}
