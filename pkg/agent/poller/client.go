package poller

import (
	"context"
	"fmt"
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
	onReconnect        func()
	poller             *pollExecutor
	newPollManager     newPollExecutorFunc
	harvesterConfig    harvesterConfig
	mutex              sync.RWMutex
	initialized        bool
	firstStart         bool
	connectedCh        chan struct{} // closed when the first connection is live in Start()
	connectedOnce      sync.Once    // ensures connectedCh is closed at most once across reconnects
	startErrCh         chan error    // buffered(1): receives error if Start() fails before connecting
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
		firstStart:     true,
		connectedCh:    make(chan struct{}),
		startErrCh:     make(chan error, 1),
	}

	for _, opt := range options {
		opt(pc)
	}

	return pc, nil
}

// Start the polling client
func (p *PollClient) Start() error {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	eventCh := make(chan *proto.Event)

	p.mutex.Lock()

	p.listener = p.newListener(ctx, cancel, eventCh, p.apiClient, p.harvesterConfig.sequence, p.handlers...)

	p.poller = p.newPollManager(p.interval, withOnStop(p.onClientStop), withHarvester(p.harvesterConfig), WithContext(ctx, cancel))
	p.mutex.Unlock()
	p.listener.Listen()
	p.poller.RegisterWatch(eventCh)

	// pollExecutor may cancel ctx internally (e.g. harvester not configured).
	select {
	case <-ctx.Done():
		err := context.Cause(ctx)
		if err == nil {
			err = fmt.Errorf("poll client context cancelled during setup")
		}
		select {
		case p.startErrCh <- err:
		default:
		}
		return err
	default:
	}

	if p.onStreamConnection != nil {
		p.onStreamConnection()
	}
	p.connectedOnce.Do(func() { close(p.connectedCh) })

	if p.onReconnect != nil && !p.firstStart {
		go p.onReconnect()
	}
	p.firstStart = false

	p.mutex.Lock()
	p.initialized = true
	p.mutex.Unlock()

	<-ctx.Done()
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return fmt.Errorf("poll client context has been closed")
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

// WaitForReady blocks until Start() has established a connection to Central,
// or until ctx is cancelled, or until Start() exits with an error before connecting.
func (p *PollClient) WaitForReady(ctx context.Context) error {
	select {
	case <-p.connectedCh:
		return nil
	case err := <-p.startErrCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the streamer
func (p *PollClient) Stop() {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.poller != nil {
		p.poller.Stop()
	}
	if p.listener != nil {
		p.listener.Stop()
	}
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
