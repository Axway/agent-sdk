package poller

import (
	"fmt"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestPollerRegisterWatch(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	wt := management.NewWatchTopic("mocktopic")
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	seq.SetSequence(1)
	mockH := &mockHarvester{}

	poller := newPollExecutor(cfg.PollInterval, withHarvester(harvesterConfig{
		sequence:      seq,
		topicSelfLink: wt.GetSelfLink(),
		hClient:       mockH,
	}))

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	h := &mockHarvester{
		eventCh: eventCh,
	}

	poller.harvester = h
	poller.RegisterWatch(eventCh, errCh)

	evt := <-h.eventCh
	assert.NotNil(t, evt)
}

func TestPollerRegisterWatchError(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	wt := management.NewWatchTopic("mocktopic")
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	mockH := &mockHarvester{}

	poller := newPollExecutor(cfg.PollInterval, withHarvester(harvesterConfig{
		sequence:      seq,
		topicSelfLink: wt.GetSelfLink(),
		hClient:       mockH,
	}))

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	poller.harvester = &mockHarvester{
		err: fmt.Errorf("harvester error"),
	}

	poller.RegisterWatch(eventCh, errCh)

	err := <-errCh
	assert.NotNil(t, err)
}

func TestPollClientOptions(t *testing.T) {
	cfg := config.NewCentralConfig(config.DiscoveryAgent)
	pc, _ := NewPollClient(
		&mock.Client{}, cfg, nil,
		WithHarvester(&mockHarvester{}, &mockSequence{}, "/self/link"),
		WithOnClientStop(func() {}),
		WithHealthCheckRegister(func(string, string, hc.CheckStatus) (string, error) {
			return "", nil
		}),
	)

	assert.NotNil(t, pc.harvesterConfig.hClient)
	assert.NotNil(t, pc.harvesterConfig.sequence)
	assert.NotNil(t, pc.harvesterConfig.topicSelfLink)
	assert.NotNil(t, pc.onClientStop)
	assert.NotNil(t, pc.hcRegister)
}

type mockSequence struct{}

func (m mockSequence) GetSequence() int64 {
	return 0
}

func (m mockSequence) SetSequence(_ int64) {

}
