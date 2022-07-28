package agent

import (
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestEventSync_pollMode(t *testing.T) {
	cfg := createCentralCfg("https://abc.com", "mockenv")
	err := Initialize(cfg)

	mc := &mock.Client{}
	mc.GetResourceMock = func(url string) (*apiv1.ResourceInstance, error) {
		wt := management.NewWatchTopic("mock-wt")
		ri, err := wt.AsInstance()
		return ri, err
	}
	InitializeForTest(mc)
	assert.Nil(t, err)

	es, err := NewEventSync()
	assert.Nil(t, err)
	assert.NotNil(t, es.watchTopic)
	assert.NotNil(t, es.discoveryCache)
	assert.NotNil(t, es.sequence)
	assert.NotNil(t, es.harvester)

	es.harvester = &mockHarvester{}
	err = es.SyncCache()
	assert.Nil(t, err)
}

func TestEventSync_streamMode(t *testing.T) {
	cfg := createCentralCfg("https://abc.com", "mockenv")
	cfg.GRPCCfg = config.GRPCConfig{
		Enabled:  true,
		Insecure: true,
	}
	err := Initialize(cfg)
	agent.agentFeaturesCfg = &config.AgentFeaturesConfiguration{
		ConnectToCentral:        true,
		ProcessSystemSignals:    true,
		VersionChecker:          false,
		PersistCache:            false,
		MarketplaceProvisioning: true,
	}

	mc := &mock.Client{}
	mc.GetResourceMock = func(url string) (*apiv1.ResourceInstance, error) {
		wt := management.NewWatchTopic("mock-wt")
		ri, err := wt.AsInstance()
		return ri, err
	}
	InitializeForTest(mc)
	assert.Nil(t, err)

	es, err := NewEventSync()
	assert.Nil(t, err)
	assert.NotNil(t, es.watchTopic)
	assert.NotNil(t, es.discoveryCache)
	assert.NotNil(t, es.sequence)
	assert.NotNil(t, es.harvester)

	es.harvester = &mockHarvester{}
	err = es.SyncCache()
	assert.Nil(t, err)
}

type mockHarvester struct{}

func (m mockHarvester) EventCatchUp(link string, events chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 1, nil
}
