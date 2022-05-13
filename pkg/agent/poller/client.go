package poller

import (
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
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
	listener           events.Listener
	newListener        events.NewListenerFunc
	onStreamConnection OnStreamConnection
	poller             *manager
	seq                events.SequenceProvider
	topicSelfLink      string
}

// OnStreamConnection func for updating the PollClient after connecting to central
type OnStreamConnection func(*PollClient)

// NewPollClient creates a polling client
func NewPollClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	onStreamConnection OnStreamConnection,
	cacheBuildSignal chan interface{},
	handlers ...handler.Handler,
) (*PollClient, error) {
	wt, err := events.GetWatchTopic(cfg, apiClient)
	if err != nil {
		return nil, err
	}

	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, getToken, seq)
	poller := newPollManager(hcfg, cfg.GetPollInterval(), cacheBuildSignal)

	pc := &PollClient{
		apiClient:          apiClient,
		handlers:           handlers,
		listener:           nil,
		newListener:        events.NewEventListener,
		onStreamConnection: onStreamConnection,
		poller:             poller,
		seq:                hcfg.SequenceProvider,
		topicSelfLink:      wt.GetSelfLink(),
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

	listenCh := p.listener.Listen()

	err := p.poller.RegisterWatch(p.topicSelfLink, eventCh, eventErrorCh)
	if err != nil {
		return err
	}

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

// Stop stops the streamer
func (p *PollClient) Stop() {
	p.poller.Stop()
	p.listener.Stop()
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
