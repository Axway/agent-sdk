package stream

import (
	"context"
	"fmt"
	"testing"
	"time"

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
		TLS:           config.NewTLSConfig(),
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

	assert.NotNil(t, streamer.Status())

	errCh := make(chan error)
	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	<-manager.readyCh
	assert.Equal(t, hc.OK, hc.RunChecks())
	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err = <-errCh
	assert.NotNil(t, err)

	assert.Equal(t, hc.FAIL, hc.RunChecks())
	streamer.manager = nil
	streamer.listener = nil

	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	<-manager.readyCh

	// wait for isInitialized to be set after requestQueue.Start()
	assert.Eventually(t, func() bool { return streamer.Status() == nil }, time.Second, 10*time.Millisecond)
	stop(t, streamer, errCh)
	manager.status = false

	assert.NotNil(t, streamer.Status())
	assert.Equal(t, hc.FAIL, hc.RunChecks())
}

func TestStreamerWaitForReady(t *testing.T) {
	managerErr := fmt.Errorf("connection refused")

	cases := map[string]struct {
		newManager  func(*wm.Config, ...wm.Option) (wm.Manager, error)
		ctxTimeout  time.Duration
		expectedErr error
	}{
		"connects successfully": {
			newManager: func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
				return &mockManager{status: true}, nil // nil readyCh: RegisterWatch returns immediately
			},
			ctxTimeout:  2 * time.Second,
			expectedErr: nil,
		},
		"context deadline exceeded before connection": {
			newManager: func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
				return &blockingManager{block: make(chan struct{})}, nil
			},
			ctxTimeout:  50 * time.Millisecond,
			expectedErr: context.DeadlineExceeded,
		},
		"start fails before connecting": {
			newManager: func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
				return nil, managerErr
			},
			ctxTimeout:  2 * time.Second,
			expectedErr: managerErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streamer, err := NewStreamerClient(&mockAPIClient{}, NewConfig(), &mockTokenGetter{}, nil)
			assert.Nil(t, err)
			streamer.newManager = tc.newManager

			go func() { _ = streamer.Start() }()
			defer streamer.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()

			err = streamer.WaitForReady(ctx)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
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

			streamer.newRequestQueue = func(ctx context.Context, cancel context.CancelCauseFunc, requestCh chan *proto.Request) events.RequestQueue {
				return requestQueue
			}
			streamer.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
				return manager, nil
			}

			assert.NotNil(t, streamer.Status())
			errCh := make(chan error)
			go func() {
				err := streamer.Start()
				errCh <- err
			}()
			defer func() {
				streamer.Stop()
				err = <-errCh
				assert.NotNil(t, err)
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

func TestStreamerClient_PauseResumeListener(t *testing.T) {
	tests := map[string]struct {
		setupListener bool // whether to start the client so listener is live
		callPause     bool
		callResume    bool
	}{
		"no-op when listener is nil (before Start)": {
			setupListener: false,
			callPause:     true,
			callResume:    true,
		},
		"delegates pause to live listener": {
			setupListener: true,
			callPause:     true,
		},
		"delegates resume to live listener": {
			setupListener: true,
			callPause:     true,
			callResume:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			streamer, err := NewStreamerClient(
				&mockAPIClient{},
				NewConfig(),
				&mockTokenGetter{},
				nil,
				WithCacheManager(agentcache.NewAgentCacheManager(NewConfig(), false)),
				WithWatchTopic(&management.WatchTopic{}),
			)
			assert.Nil(t, err)

			if tc.setupListener {
				manager := &mockManager{status: true, readyCh: make(chan struct{})}
				streamer.newManager = func(_ *wm.Config, _ ...wm.Option) (wm.Manager, error) {
					return manager, nil
				}
				errCh := make(chan error, 1)
				go func() { errCh <- streamer.Start() }()
				<-manager.readyCh
				defer func() {
					streamer.listener.Stop()
					<-errCh
				}()
			}

			// Neither call should panic regardless of listener state.
			if tc.callPause {
				assert.NotPanics(t, func() { streamer.PauseListener() })
			}
			if tc.callResume {
				assert.NotPanics(t, func() { streamer.ResumeListener() })
			}
		})
	}
}

func stop(t *testing.T, streamer *StreamerClient, errCh chan error) {
	t.Helper()
	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err := <-errCh
	assert.NotNil(t, err)
}

type mockManager struct {
	status  bool
	readyCh chan struct{}
}

func (m *mockManager) RegisterWatch(_ string, _ chan *proto.Event) (string, error) {
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

// blockingManager blocks RegisterWatch until block is closed, simulating a slow/unresponsive Central.
type blockingManager struct {
	block chan struct{}
}

func (m *blockingManager) RegisterWatch(_ string, _ chan *proto.Event) (string, error) {
	<-m.block
	return "", nil
}

func (m *blockingManager) CloseWatch(_ string) error { return nil }
func (m *blockingManager) CloseConn()                {}
func (m *blockingManager) Status() bool              { return true }

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

func (m mockHarvester) EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
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
