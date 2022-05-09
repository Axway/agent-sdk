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

type pollClient struct {
	apiClient     events.APIClient
	handlers      []handler.Handler
	listener      events.Listener
	poller        *manager
	newListener   events.NewListenerFunc
	seq           events.SequenceProvider
	topicSelfLink string
}

func NewPollClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	handlers ...handler.Handler,
) (*pollClient, error) {
	wt, err := events.GetWatchTopic(cfg, apiClient)
	if err != nil {
		return nil, err
	}

	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	hcfg := harvester.NewConfig(cfg, getToken, seq)
	poller := newPollManager(hcfg, cfg.GetPollInterval())

	pc := &pollClient{
		poller:        poller,
		handlers:      handlers,
		listener:      nil,
		newListener:   events.NewEventListener,
		topicSelfLink: wt.GetSelfLink(),
	}

	return pc, nil
}

func (c *pollClient) Start() error {
	eventCh, eventErrorCh := make(chan *proto.Event), make(chan error)
	c.listener = c.newListener(
		eventCh,
		c.apiClient,
		c.seq,
		c.handlers...,
	)

	listenCh := c.listener.Listen()

	_, err := c.poller.RegisterWatch(c.topicSelfLink, eventCh, eventErrorCh)
	if err != nil {
		return err
	}

	select {
	case err := <-listenCh:
		return err
	case err := <-eventErrorCh:
		return err
	}
}

// Stop stops the streamer
func (c *pollClient) Stop() {
	c.listener.Stop()
	c.poller.Stop()
}

func (c *pollClient) Status() error {
	if c.poller == nil || c.listener == nil {
		return fmt.Errorf("harvester polling client is not ready")
	}
	if ok := c.poller.Status(); !ok {
		return errors.ErrHarvesterConnection
	}

	return nil
}

func (c *pollClient) HealthCheck(_ string) *hc.Status {
	err := c.Status()
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
