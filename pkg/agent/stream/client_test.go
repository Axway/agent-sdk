package stream

import (
	"context"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"strings"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/stretchr/testify/assert"
)

var noop = func() {}

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
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		watchTopic: ri,
	}
	cfg := NewConfig()
	cacheManager := agentcache.NewAgentCacheManager(&cfg, false)
	onStreamConnection := func(s *StreamerClient) {
		hc.RegisterHealthcheck(util.AmplifyCentral, "central", s.Healthcheck)
	}
	streamer, err := NewStreamerClient(httpClient, &cfg, getToken, cacheManager, onStreamConnection, nil)
	assert.NotNil(t, streamer)
	assert.Nil(t, err)

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

	assert.Equal(t, hc.OK, hc.RunChecks())
	streamer.manager = nil
	streamer.listener = nil

	go func() {
		err := streamer.Start()
		errCh <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)

	go func() {
		for streamer.manager == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cancel()
	}()
	<-ctx.Done()
	if !assert.Equal(t, context.Canceled, ctx.Err(), "Init was never completed") {
		assert.FailNow(t, "no need to go further minimal state not reached")
	}

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
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		watchTopic: ri,
		paged: []*apiv1.ResourceInstance{
			createRI("123", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	initDone := false
	onStreamConnection := func(s *StreamerClient) {
		initDone = true
	}
	tHandler := mockHandler{}
	underTest, err := NewStreamerClient(httpClient, &cfg, getToken, cacheManager, onStreamConnection, noop, &tHandler)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{status: true}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.NotNil(t, underTest.Status())

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)

	go func() {
		for !initDone {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cancel()
	}()
	<-ctx.Done()
	if !assert.Equal(t, context.Canceled, ctx.Err(), "Values were never added") {
		assert.FailNow(t, "no need to go further minimal state not reached")
	}

	assert.True(t, httpClient.pagedCalled)

	res := underTest.cacheManager.GetAllFetchOnStartupResources()
	assert.Len(t, res, 2)

	// make sure handler are called
	underTest.HandleFetchOnStartupResources()
	assert.Len(t, tHandler.received, 2)

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
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		watchTopic: ri,
		paged: []*apiv1.ResourceInstance{
			createRI("123", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()
	cfg.GRPCCfg.FetchOnStartup.Retention = time.Millisecond

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	initDone := false
	onStreamConnection := func(s *StreamerClient) {
		initDone = true
	}
	tHandler := mockHandler{}
	underTest, err := NewStreamerClient(httpClient, &cfg, getToken, cacheManager, onStreamConnection, noop, &tHandler)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{status: true}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	assert.NotNil(t, underTest.Status())

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)

	go func() {
		for !initDone {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cancel()
	}()
	<-ctx.Done()
	if !assert.Equal(t, context.Canceled, ctx.Err(), "Values were never added") {
		assert.FailNow(t, "no need to go further minimal state not reached")
	}

	res := underTest.cacheManager.GetAllFetchOnStartupResources()
	assert.Len(t, res, 0)

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
					Type: []string{events.WatchTopicFilterTypeDeleted}, // deleted => hence nothing to load
				},
			},
		},
	}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		watchTopic: ri,
		paged: []*apiv1.ResourceInstance{
			createRI("132", "foo"),
			createRI("456", "bar"),
		},
	}
	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	initDone := false
	onStreamConnection := func(s *StreamerClient) {
		initDone = true
	}

	tHandler := mockHandler{}
	underTest, err := NewStreamerClient(httpClient, &cfg, getToken, cacheManager, onStreamConnection, noop, &tHandler)
	assert.NotNil(t, underTest)
	assert.NoError(t, err)

	manager := &mockManager{status: true}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	go func() {
		for !initDone {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cancel()
	}()
	<-ctx.Done()
	if !assert.Equal(t, ctx.Err(), context.Canceled, "Init of streamer not finished") {
		underTest.listener.Stop()
		assert.FailNow(t, "no need to go further minimal state not reached")
	}

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
					Type: []string{events.WatchTopicFilterTypeCreated},
				},
			},
		},
	}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		watchTopic: ri,
		resource:   createRI("123", "foo"),
	}

	cfg := NewConfigWithLoadOnStartup()

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	initDone := false
	onStreamConnection := func(s *StreamerClient) {
		initDone = true
	}

	tHandler := mockHandler{}
	underTest, err := NewStreamerClient(httpClient, &cfg, getToken, cacheManager, onStreamConnection, noop, &tHandler)
	assert.NoError(t, err)

	manager := &mockManager{status: true}
	underTest.newManager = func(cfg *wm.Config, opts ...wm.Option) (wm.Manager, error) {
		return manager, nil
	}

	errCh := make(chan error)
	go func() {
		err := underTest.Start()
		errCh <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	go func() {
		for !initDone {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		cancel()
	}()
	<-ctx.Done()
	if !assert.Equal(t, ctx.Err(), context.Canceled, "Init of streamer not finished") {
		underTest.listener.Stop()
		assert.FailNow(t, "no need to go further minimal state not reached")
	}

	assert.False(t, httpClient.pagedCalled)

	res := underTest.cacheManager.GetAllFetchOnStartupResources()
	assert.Len(t, res, 1)

	// make sure handler are called
	underTest.HandleFetchOnStartupResources()
	assert.Len(t, tHandler.received, 1)
	assert.Equal(t, "foo", tHandler.received[0].Name)
	assert.Equal(t, "123", tHandler.received[0].Metadata.ID)

	// and won't be anymore
	assert.Empty(t, underTest.cacheManager.GetAllFetchOnStartupResources())

	// should stop the listener and write nil to the listener's error channel
	stop(t, underTest, errCh)

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
	status bool
}

func (m *mockManager) RegisterWatch(_ string, _ chan *proto.Event, _ chan error) (string, error) {
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
	watchTopic  *apiv1.ResourceInstance
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
	if strings.Contains(url, mv1.WatchTopicResourceName) {
		return m.watchTopic, m.getErr
	}
	return m.resource, m.getErr
}

func (m mockAPIClient) CreateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.watchTopic, m.createErr
}

func (m mockAPIClient) UpdateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.watchTopic, m.updateErr
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
