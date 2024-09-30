package poller

import (
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
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
	interval           time.Duration
	listener           events.Listener
	newListener        events.NewListenerFunc
	onClientStop       onClientStopCb
	onStreamConnection func()
	poller             *pollExecutor
	newPollManager     newPollExecutorFunc
	harvesterConfig    harvesterConfig
	mutex              sync.RWMutex
	initialized        bool
}

type harvesterConfig struct {
	sequence      events.SequenceProvider
	topicSelfLink string
	hClient       harvester.Harvest
}

type onClientStopCb func()

// NewPollClient creates a polling client
func NewPollClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	handlers []handler.Handler,
	options ...ClientOpt,
) (*PollClient, error) {
	pc := &PollClient{
		apiClient:      apiClient,
		handlers:       handlers,
		interval:       cfg.GetPollInterval(),
		listener:       nil,
		newListener:    events.NewEventListener,
		newPollManager: newPollExecutor,
		poller:         nil,
	}

	for _, opt := range options {
		opt(pc)
	}

	return pc, nil
}

// Start the polling client
func (p *PollClient) Start() error {
	eventCh, eventErrorCh := make(chan *proto.Event), make(chan error)

	p.mutex.Lock()

	p.listener = p.newListener(
		eventCh,
		p.apiClient,
		p.harvesterConfig.sequence,
		p.handlers...,
	)

	p.poller = p.newPollManager(
		p.interval,
		withOnStop(p.onClientStop),
		withHarvester(p.harvesterConfig),
	)
	p.mutex.Unlock()
	listenCh := p.listener.Listen()
	p.poller.RegisterWatch(eventCh, eventErrorCh)

	if p.onStreamConnection != nil {
		p.onStreamConnection()
	}
	p.initialized = true

	select {
	case err := <-listenCh:
		return err
	case err := <-eventErrorCh:
		return err
	}
}

// Status returns an error if the poller is not running
func (p *PollClient) Status() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.initialized {
		if ok := p.poller.Status(); !ok {
			return errors.ErrHarvesterConnection
		}
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
