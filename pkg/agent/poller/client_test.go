package poller

import (
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
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
	SingleURL:     "https://abc.com",
	PollInterval:  1 * time.Second,
}

func TestPollClientStart(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}

	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	poller, err := NewPollClient(httpClient, cfg, getToken, cacheManager, nil, nil)
	poller.poller.harvester = &mockHarvester{}
	assert.NotNil(t, poller)
	assert.Nil(t, err)

	assert.NotNil(t, poller.Status())

	errCh := make(chan error)
	go func() {
		err := poller.Start()
		errCh <- err
	}()

	for poller.listener == nil {
		continue
	}

	// assert the poller is healthy
	assert.Nil(t, poller.Status())
	assert.Equal(t, hc.OK, poller.Healthcheck("").Result)

	// should stop the poller and write nil to the error channel
	poller.Stop()

	err = <-errCh
	assert.Nil(t, err)

	assert.Equal(t, hc.FAIL, poller.Healthcheck("").Result)
	assert.NotNil(t, poller.Status())
	poller.poller = nil
	poller.listener = nil
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

type mockTokenGetter struct {
	token string
	err   error
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return m.token, m.err
}

type mockHarvester struct {
	eventCh chan *proto.Event
	err     error
}

func (m mockHarvester) EventCatchUp(_ string, _ chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(_ string, _ int64, _ chan *proto.Event) (int64, error) {
	if m.eventCh != nil {
		m.eventCh <- &proto.Event{
			Id: "1",
		}
	}
	return 0, m.err
}
