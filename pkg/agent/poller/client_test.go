package poller

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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
	SingleURL:     "https://abc.com",
}

func TestPollClient(t *testing.T) {
	getToken := &mockTokenGetter{}
	wt := &mv1.WatchTopic{}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}

	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	streamer, err := NewPollClient(httpClient, cfg, getToken, cacheManager)
	assert.NotNil(t, streamer)
	assert.Nil(t, err)
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
