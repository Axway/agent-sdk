package poller

import (
	"fmt"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestPollerRegisterWatch(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	wt := mv1.NewWatchTopic("mocktopic")
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, &mockTokenGetter{}, seq)
	poller := newPollManager(hcfg, cfg.GetPollInterval(), nil)

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	h := &mockHarvester{
		eventCh: eventCh,
	}

	poller.harvester = h
	poller.RegisterWatch(wt.GetSelfLink(), eventCh, errCh)

	evt := <-h.eventCh
	assert.NotNil(t, evt)
}

func TestPollerRegisterWatchError(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	wt := mv1.NewWatchTopic("mocktopic")
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, &mockTokenGetter{}, seq)
	poller := newPollManager(hcfg, cfg.GetPollInterval(), nil)

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	poller.harvester = &mockHarvester{
		err: fmt.Errorf("harvester error"),
	}

	poller.RegisterWatch(wt.GetSelfLink(), eventCh, errCh)

	err := <-errCh
	assert.NotNil(t, err)
}
