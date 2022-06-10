package stream

import (
	"context"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/apic/mock"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/stretchr/testify/assert"
)

func NewConfig() config.CentralConfiguration {
	return config.CentralConfiguration{
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

func NewConfigWithLoadOnStartup() config.CentralConfiguration {
	cfg := NewConfig()
	cfg.GRPCCfg.FetchOnStartup = config.FetchOnStartup{
		Enabled:   true,
		PageSize:  10,
		Retention: 300 * time.Millisecond,
	}
	return cfg
}

// should create a new streamer and call Start
func TestNewStreamer(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{}
	httpClient := &mockAPIClient{}
	cfg := NewConfig()
	cacheManager := agentcache.NewAgentCacheManager(&cfg, false)

	streamer, err := NewStreamerClient(
		httpClient,
		&cfg,
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

func TestNewStreamerWithFetchOnStartup(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{
		Spec: mv1.WatchTopicSpec{
			Filters: []mv1.WatchTopicSpecFilters{
				{
					Name: "*",
					Kind: mv1.AccessRequestGVK().Kind,
					Scope: &mv1.WatchTopicSpecScope{
						Kind: mv1.EnvironmentGVK().Kind,
						Name: "mock",
					},
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	httpClient := &mockAPIClient{
		paged: []*apiv1.ResourceInstance{
			createRI("123", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	tHandler := &mockHandler{}
	underTest, err := NewStreamerClient(
		httpClient,
		&cfg,
		getToken,
		[]handler.Handler{tHandler},
		WithCacheManager(cacheManager),
		WithWatchTopic(wt),
	)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{
		status:  true,
		readyCh: make(chan struct{}),
	}

	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.NotNil(t, underTest.Status())

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	<-manager.readyCh

	assert.True(t, httpClient.pagedCalled)

	res := underTest.cacheManager.GetAllFetchOnStartupResources()
	assert.Equal(t, 2, len(res))

	// make sure handler are called
	underTest.HandleFetchOnStartupResources()
	assert.Equal(t, 2, len(tHandler.received))

	// and won't be anymore
	assert.Empty(t, underTest.cacheManager.GetAllFetchOnStartupResources())

	stop(t, underTest, errCh)
}

func TestNewStreamerWithFetchOnStartupRetentionToZeroEmptiesCache(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{
		Spec: mv1.WatchTopicSpec{
			Filters: []mv1.WatchTopicSpecFilters{
				{
					Name: "*",
					Kind: mv1.AccessRequestGVK().Kind,
					Scope: &mv1.WatchTopicSpecScope{
						Kind: mv1.EnvironmentGVK().Kind,
						Name: "mock",
					},
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	httpClient := &mockAPIClient{
		paged: []*apiv1.ResourceInstance{
			createRI("123", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()
	cfg.GRPCCfg.FetchOnStartup.Retention = time.Millisecond

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	tHandler := &mockHandler{}

	underTest, err := NewStreamerClient(httpClient,
		&cfg,
		getToken,
		[]handler.Handler{tHandler},
		WithCacheManager(cacheManager),
		WithWatchTopic(wt),
	)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{
		status:  true,
		readyCh: make(chan struct{}),
	}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.NotNil(t, underTest.Status())

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	<-manager.readyCh

	res := underTest.cacheManager.GetAllFetchOnStartupResources()

	assert.Equal(t, 2, len(res))

	stop(t, underTest, errCh)

}

func TestNewStreamerWithFetchOnStartupButNothingToLoad(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{
		Spec: mv1.WatchTopicSpec{
			Filters: []mv1.WatchTopicSpecFilters{
				{
					Name: "*",
					Kind: mv1.AccessRequestGVK().Kind,
					Scope: &mv1.WatchTopicSpecScope{
						Kind: mv1.EnvironmentGVK().Kind,
						Name: "mock",
					},
					Type: []string{events.WatchTopicFilterTypeDeleted}, // deleted => hence nothing to load
				},
			},
		},
	}
	httpClient := &mockAPIClient{
		paged: []*apiv1.ResourceInstance{
			createRI("132", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	tHandler := &mockHandler{}
	underTest, err := NewStreamerClient(httpClient,
		&cfg,
		getToken,
		[]handler.Handler{tHandler},
		WithCacheManager(cacheManager),
		WithWatchTopic(wt),
	)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{
		status:  true,
		readyCh: make(chan struct{}),
	}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	<-manager.readyCh

	// at this stage we should have resources loaded... but here nothing to load (all deleted)
	underTest.HandleFetchOnStartupResources()
	assert.Nil(t, tHandler.received)

	stop(t, underTest, errCh)
}

func TestNewStreamerWithFetchOnStartupWithNamedTopic(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{
		Spec: mv1.WatchTopicSpec{
			Filters: []mv1.WatchTopicSpecFilters{
				{
					Name: "foo",
					Kind: mv1.AccessRequestGVK().Kind,
					Scope: &mv1.WatchTopicSpecScope{
						Kind: mv1.EnvironmentGVK().Kind,
						Name: "mock",
					},
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	httpClient := &mockAPIClient{
		resource: createRI("123", "foo"),
	}

	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)

	tHandler := &mockHandler{}
	underTest, err := NewStreamerClient(
		httpClient,
		&cfg, getToken,
		[]handler.Handler{tHandler},
		WithCacheManager(cacheManager),
		WithWatchTopic(wt),
	)
	assert.NoError(t, err)

	manager := &mockManager{
		status:  true,
		readyCh: make(chan struct{}),
	}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	<-manager.readyCh

	assert.False(t, httpClient.pagedCalled)

	res := underTest.cacheManager.GetAllFetchOnStartupResources()
	assert.Equal(t, 1, len(res))

	// make sure handler are called
	underTest.HandleFetchOnStartupResources()
	assert.Equal(t, 1, len(tHandler.received))
	assert.Equal(t, "foo", tHandler.received[0].Name)
	assert.Equal(t, "123", tHandler.received[0].Metadata.ID)

	// and won't be anymore
	assert.Empty(t, underTest.cacheManager.GetAllFetchOnStartupResources())

	// should stop the listener and write nil to the listener's error channel
	stop(t, underTest, errCh)

}

func TestClientOptions(t *testing.T) {
	sc, _ := NewStreamerClient(
		&mock.Client{},
		config.NewCentralConfig(config.DiscoveryAgent),
		&mockTokenGetter{},
		nil,
		WithHarvester(&mockHarvester{}, &mockSequence{}),
		WithEventSyncError(func() {
		}),
		WithOnStreamConnection(),
	)
	assert.NotNil(t, sc.harvester)
	assert.NotNil(t, sc.sequence)
	assert.NotNil(t, sc.onEventSyncError)
	assert.NotNil(t, sc.onStreamConnection)
}

func stop(t *testing.T, streamer *StreamerClient, errCh chan error) {
	t.Helper()
	// should stop the listener and write nil to the listener's error channel
	streamer.listener.Stop()

	err := <-errCh
	assert.Nil(t, err)
}

func createRI(id, name string) *apiv1.ResourceInstance {
	return &apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ID: id,
			},
			Name: name,
		},
	}
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
	updateErr   error
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

func (m *mockAPIClient) GetAPIV1ResourceInstancesWithPageSize(map[string]string, string, int) ([]*apiv1.ResourceInstance, error) {
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

type mockHandler struct {
	err      error
	received []*apiv1.ResourceInstance
}

func (m *mockHandler) Handle(_ context.Context, _ *proto.EventMeta, ri *apiv1.ResourceInstance) error {
	if m.received == nil {
		m.received = make([]*apiv1.ResourceInstance, 0, 1)
	}
	m.received = append(m.received, ri)
	return m.err
}

type mockHarvester struct{}

func (m mockHarvester) EventCatchUp(link string, events chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 0, nil
}

type mockSequence struct{}

func (m mockSequence) GetSequence() int64 {
	return 0
}

func (m mockSequence) SetSequence(sequenceID int64) {
	return
}
