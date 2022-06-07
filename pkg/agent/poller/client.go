package poller

import (
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util"
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
}

type harvesterConfig struct {
	sequence      events.SequenceProvider
	topicSelfLink string
	hClient       harvester.Harvest
}

type onClientStopCb func()

type ClientOpt func(pc *PollClient)

// WithHarvester configures the polling client to use harvester
func WithHarvester(hClient harvester.Harvest, sequence events.SequenceProvider, topicSelfLink string) ClientOpt {
	return func(pc *PollClient) {
		pc.harvesterConfig.hClient = hClient
		pc.harvesterConfig.topicSelfLink = topicSelfLink
		pc.harvesterConfig.sequence = sequence
	}
}

// WithOnClientStop func to execute when the client shuts down
func WithOnClientStop(cb func()) ClientOpt {
	return func(pc *PollClient) {
		pc.onClientStop = cb
	}
}

// WithOnConnect func to execute when a connection to central is made
func WithOnConnect() ClientOpt {
	return func(pc *PollClient) {
		pc.onStreamConnection = func() {
			hc.RegisterHealthcheck(util.AmplifyCentral, "central", pc.Healthcheck)
		}
	}
}

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

	p.listener = p.newListener(
		eventCh,
		p.apiClient,
		p.harvesterConfig.sequence,
		p.handlers...,
	)

	poller := p.newPollManager(
		p.interval,
		withOnStop(p.onClientStop),
		withHarvester(p.harvesterConfig),
	)
	p.poller = poller

	listenCh := p.listener.Listen()

	p.poller.RegisterWatch(eventCh, eventErrorCh)

	if p.onStreamConnection != nil {
		p.onStreamConnection()
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
