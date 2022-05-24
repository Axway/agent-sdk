package poller

import (
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
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
	pollClient, err := NewPollClient(httpClient, cfg, getToken, cacheManager, nil, nil)
	assert.NotNil(t, pollClient)
	assert.Nil(t, err)

	mockH := &mockHarvester{
		readyCh: make(chan struct{}),
	}

	pollClient.newPollManager = func(cfg *harvester.Config, interval time.Duration, onStop onClientStopCb) *manager {
		p := newPollManager(cfg, interval, onStop)
		p.harvester = mockH
		return p
	}

	errCh := make(chan error)
	go func() {
		err := pollClient.Start()
		errCh <- err
	}()

	<-mockH.readyCh

	// assert the poller is healthy
	assert.Nil(t, pollClient.Status())
	assert.Equal(t, hc.OK, pollClient.Healthcheck("").Result)

	// should stop the poller and write nil to the error channel
	pollClient.Stop()

	err = <-errCh
	assert.Nil(t, err)

	assert.Equal(t, hc.FAIL, pollClient.Healthcheck("").Result)
	assert.NotNil(t, pollClient.Status())
	pollClient.poller = nil
	pollClient.listener = nil
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

func (m mockAPIClient) CreateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.ri, m.createErr
}

func (m mockAPIClient) UpdateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.ri, m.updateErr
}

func (m mockAPIClient) DeleteResourceInstance(_ apiv1.Interface) error {
	return m.deleteErr
}

func (m mockAPIClient) GetAPIV1ResourceInstancesWithPageSize(map[string]string, string, int) ([]*apiv1.ResourceInstance, error) {
	return nil, nil
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
	readyCh chan struct{}
}

func (m mockHarvester) EventCatchUp(_ string, _ chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(_ string, _ int64, _ chan *proto.Event) (int64, error) {
	if m.readyCh != nil {
		m.readyCh <- struct{}{}
	}

	if m.eventCh != nil {
		m.eventCh <- &proto.Event{
			Id: "1",
		}
	}
	return 0, m.err
}
