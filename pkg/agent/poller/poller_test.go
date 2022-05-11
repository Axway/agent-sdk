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
	poller := newPollManager(hcfg, cfg.GetPollInterval())

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	poller.harvester = &mockHarvester{
		seqID:   234,
		eventCh: eventCh,
	}

	err := poller.RegisterWatch(wt.GetSelfLink(), eventCh, errCh)
	assert.Nil(t, err)

	assert.Equal(t, int64(234), seq.GetSequence())
}

func TestPollerRegisterWatchError(t *testing.T) {
	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	wt := mv1.NewWatchTopic("mocktopic")
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, &mockTokenGetter{}, seq)
	poller := newPollManager(hcfg, cfg.GetPollInterval())

	eventCh, errCh := make(chan *proto.Event), make(chan error)
	poller.harvester = &mockHarvester{
		seqID: 0,
		err:   fmt.Errorf("harvester error"),
	}

	err := poller.RegisterWatch(wt.GetSelfLink(), eventCh, errCh)
	assert.Nil(t, err)

	err = <-errCh
	assert.NotNil(t, err)
}
