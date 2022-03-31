package stream

import (
	"testing"

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
	wt := &mv1.WatchTopic{}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}

	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	onStreamConnection := func(s Streamer) {
		hc.RegisterHealthcheck(util.AmplifyCentral, "central", s.Healthcheck)
	}
	c, err := NewStreamer(httpClient, cfg, getToken, cacheManager, onStreamConnection)
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

	assert.Equal(t, hc.OK, hc.RunChecks())
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
	streamer.Stop()
	err = <-errCh
	assert.Nil(t, err)
	manager.status = false

	assert.NotNil(t, streamer.Status())
	assert.Equal(t, hc.FAIL, hc.RunChecks())
}

func TestClientStreamJob(t *testing.T) {
	s := &mockStreamer{}
	j := NewClientStreamJob(s)

	assert.Nil(t, j.Status())
	assert.True(t, j.Ready())
	assert.Nil(t, j.Execute())
}

func Test_getAgentSequenceManager(t *testing.T) {
	wtName := "fake"
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	sm := newAgentSequenceManager(cacheManager, wtName)
	assert.Equal(t, sm.GetSequence(), int64(0))

	sm = newAgentSequenceManager(cacheManager, "")
	assert.Equal(t, sm.GetSequence(), int64(0))
}

func Test_getWatchTopic(t *testing.T) {
	wt := &mv1.WatchTopic{}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}
	wt, err := getWatchTopic(cfg, httpClient)
	assert.NotNil(t, wt)
	assert.Nil(t, err)

	wt, err = getWatchTopic(cfg, httpClient)
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

func (m mockStreamer) Healthcheck(_ string) *hc.Status {
	return &hc.Status{
		Result: hc.OK,
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
	ri        *apiv1.ResourceInstance
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m mockAPIClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return m.ri, m.getErr
}

func (m mockAPIClient) CreateResource(url string, bts []byte) (*apiv1.ResourceInstance, error) {
	return m.ri, m.createErr
}

func (m mockAPIClient) UpdateResource(url string, bts []byte) (*apiv1.ResourceInstance, error) {
	return m.ri, m.updateErr
}

func (m mockAPIClient) DeleteResourceInstance(*apiv1.ResourceInstance) error {
	return m.deleteErr
}
