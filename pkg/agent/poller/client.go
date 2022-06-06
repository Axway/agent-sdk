package poller

import (
	"fmt"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// PollClient is a client for polling harvester
type PollClient struct {
	apiClient          events.APIClient
	handlers           []handler.Handler
	hcfg               *harvester.Config
	interval           time.Duration
	listener           events.Listener
	newListener        events.NewListenerFunc
	onClientStop       onClientStopCb
	onStreamConnection OnStreamConnection
	poller             *manager
	seq                events.SequenceProvider
	topicSelfLink      string
	newPollManager     newPollManagerFunc
}

type onClientStopCb func()

// OnStreamConnection func for updating the PollClient after connecting to central
type OnStreamConnection func(*PollClient)

// NewPollClient creates a polling client
func NewPollClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	onStreamConnection OnStreamConnection,
	onClientStop onClientStopCb,
	wt *management.WatchTopic,
	handlers ...handler.Handler,
) (*PollClient, error) {

	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, getToken, seq)

	pc := &PollClient{
		apiClient:          apiClient,
		handlers:           handlers,
		hcfg:               hcfg,
		interval:           cfg.GetPollInterval(),
		listener:           nil,
		newListener:        events.NewEventListener,
		onClientStop:       onClientStop,
		onStreamConnection: onStreamConnection,
		poller:             nil,
		seq:                hcfg.SequenceProvider,
		topicSelfLink:      wt.GetSelfLink(),
		newPollManager:     newPollManager,
	}

	return pc, nil
}

// Start the polling client
func (p *PollClient) Start() error {
	eventCh, eventErrorCh := make(chan *proto.Event), make(chan error)

	p.listener = p.newListener(
		eventCh,
		p.apiClient,
		p.seq,
		p.handlers...,
	)

	poller := p.newPollManager(p.hcfg, p.interval, p.onClientStop)
	p.poller = poller

	listenCh := p.listener.Listen()

	p.poller.RegisterWatch(p.topicSelfLink, eventCh, eventErrorCh)

	if p.onStreamConnection != nil {
		p.onStreamConnection(p)
	}

	select {
	case err := <-listenCh:
		return err
	case err := <-eventErrorCh:
		return err
	}
}

// Status returns an error if the poller is not running
func (p *PollClient) Status() error {
	if p.poller == nil || p.listener == nil {
		return fmt.Errorf("harvester polling client is not ready")
	}
	if ok := p.poller.Status(); !ok {
		return errors.ErrHarvesterConnection
	}

	return nil
}

// Stop stops the streamer
func (p *PollClient) Stop() {
	p.poller.Stop()
	p.listener.Stop()
}

// Healthcheck returns a healthcheck
func (p *PollClient) Healthcheck(_ string) *hc.Status {
	err := p.Status()
	if err != nil {
		return &hc.Status{
			Result:  hc.FAIL,
			Details: err.Error(),
		}
	}
	return &hc.Status{
		Result: hc.OK,
	}
}
