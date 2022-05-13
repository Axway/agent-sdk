package stream

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/util/errors"

	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
)

// OnStreamConnection - callback StreamerClient will invoke after stream connection is established
type OnStreamConnection func(*StreamerClient)

// StreamerClient client for starting a watch controller stream and handling the events
type StreamerClient struct {
	apiClient          events.APIClient
	handlers           []handler.Handler
	listener           events.Listener
	manager            wm.Manager
	newListener        events.NewListenerFunc
	newManager         wm.NewManagerFunc
	onStreamConnection OnStreamConnection
	seq                events.SequenceProvider
	topicSelfLink      string
	watchCfg           *wm.Config
	watchOpts          []wm.Option
}

// NewStreamerClient creates a StreamerClient
func NewStreamerClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	onStreamConnection OnStreamConnection,
	cacheBuildSignal chan interface{},
	handlers ...handler.Handler,
) (*StreamerClient, error) {
	tenant := cfg.GetTenantID()

	wt, err := events.GetWatchTopic(cfg, apiClient)
	if err != nil {
		return nil, err
	}

	host, port := getWatchServiceHostPort(cfg)

	watchCfg := &wm.Config{
		Host:        host,
		Port:        uint32(port),
		TenantID:    tenant,
		TokenGetter: getToken.GetToken,
	}
	seq := events.NewSequenceProvider(cacheManager, wt.Name)

	watchOpts := []wm.Option{
		wm.WithLogger(logrus.NewEntry(log.Get())),
		wm.WithSyncEvents(seq),
		wm.WithTLSConfig(cfg.GetTLSConfig().BuildTLSConfig()),
		wm.WithProxy(cfg.GetProxyURL()),
		wm.WithHarvesterSignalChan(cacheBuildSignal),
	}

	if cfg.GetSingleURL() != "" {
		singleEntryURL, err := url.Parse(cfg.GetSingleURL())
		if err == nil {
			singleEntryAddr := util.ParseAddr(singleEntryURL)
			wm.WithSingleEntryAddr(singleEntryAddr)
		}
	}

	return &StreamerClient{
		handlers:           handlers,
		apiClient:          apiClient,
		topicSelfLink:      wt.Metadata.SelfLink,
		watchCfg:           watchCfg,
		watchOpts:          watchOpts,
		newManager:         wm.New,
		newListener:        events.NewEventListener,
		seq:                seq,
		onStreamConnection: onStreamConnection,
	}, nil
}

func getWatchServiceHostPort(cfg config.CentralConfig) (string, int) {
	u, _ := url.Parse(cfg.GetURL())
	host := cfg.GetGRPCHost()
	port := cfg.GetGRPCPort()
	if host == "" {
		host = u.Host
	}

	if port == 0 {
		if u.Port() == "" {
			port, _ = net.LookupPort("tcp", u.Scheme)
		} else {
			port, _ = strconv.Atoi(u.Port())
		}
	}

	return host, port
}

// Start creates and starts everything needed for a stream connection to central.
func (c *StreamerClient) Start() error {
	eventCh, eventErrorCh := make(chan *proto.Event), make(chan error)

	c.listener = c.newListener(
		eventCh,
		c.apiClient,
		c.seq,
		c.handlers...,
	)

	manager, err := c.newManager(c.watchCfg, c.watchOpts...)
	if err != nil {
		return err
	}

	c.manager = manager

	listenCh := c.listener.Listen()

	// lock the cache until all harvester events have been saved
	_, err = c.manager.RegisterWatch(c.topicSelfLink, eventCh, eventErrorCh)
	if err != nil {
		return err
	}

	if c.onStreamConnection != nil {
		c.onStreamConnection(c)
	}

	select {
	case err := <-listenCh:
		return err
	case err := <-eventErrorCh:
		return err
	}
}

// Status returns the health status
func (c *StreamerClient) Status() error {
	if c.manager == nil || c.listener == nil {
		return fmt.Errorf("stream client is not ready")
	}
	if ok := c.manager.Status(); !ok {
		return errors.ErrGrpcConnection
	}

	return nil
}

// Stop stops the StreamerClient
func (c *StreamerClient) Stop() {
	c.manager.CloseConn()
	c.listener.Stop()
}

// Healthcheck - health check for stream client
func (c *StreamerClient) Healthcheck(_ string) *hc.Status {
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
