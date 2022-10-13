package stream

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"

	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/sirupsen/logrus"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/util/errors"

	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
)

// StreamerClient client for starting a watch controller stream and handling the events
type StreamerClient struct {
	apiClient               events.APIClient
	handlers                []handler.Handler
	listener                *events.EventListener
	manager                 wm.Manager
	newListener             events.NewListenerFunc
	newManager              wm.NewManagerFunc
	onStreamConnection      func()
	sequence                events.SequenceProvider
	topicSelfLink           string
	watchCfg                *wm.Config
	watchOpts               []wm.Option
	cacheManager            agentcache.Manager
	resourcesOnStartupTimer *time.Timer
	logger                  log.FieldLogger
	environmentURL          string
	wt                      *management.WatchTopic
	harvester               harvester.Harvest
	onEventSyncError        func()
	mutex                   sync.RWMutex
	isInitialized           bool
}

// NewStreamerClient creates a StreamerClient
func NewStreamerClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	handlers []handler.Handler,
	options ...StreamerOpt,
) (*StreamerClient, error) {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent.stream").
		WithComponent("Client")

	tenant := cfg.GetTenantID()
	host, port := getWatchServiceHostPort(cfg)

	watchCfg := &wm.Config{
		Host:        host,
		Port:        uint32(port),
		TenantID:    tenant,
		TokenGetter: getToken.GetToken,
	}

	s := &StreamerClient{
		handlers:       handlers,
		apiClient:      apiClient,
		watchCfg:       watchCfg,
		newManager:     wm.New,
		newListener:    events.NewEventListener,
		logger:         logger,
		environmentURL: cfg.GetEnvironmentURL(),
	}

	for _, opt := range options {
		opt(s)
	}

	s.watchOpts = []wm.Option{
		wm.WithLogger(logrus.NewEntry(log.Get())),
		wm.WithHarvester(s.harvester, s.sequence),
		wm.WithProxy(cfg.GetProxyURL()),
		wm.WithEventSyncError(s.onEventSyncError),
	}

	if cfg.IsGRPCInsecure() {
		s.watchOpts = append(s.watchOpts, wm.WithTLSConfig(nil))
	} else {
		s.watchOpts = append(s.watchOpts, wm.WithTLSConfig(cfg.GetTLSConfig().BuildTLSConfig()))
	}

	if cfg.GetSingleURL() != "" {
		singleEntryURL, err := url.Parse(cfg.GetSingleURL())
		if err == nil {
			singleEntryAddr := util.ParseAddr(singleEntryURL)
			s.watchOpts = append(s.watchOpts, wm.WithSingleEntryAddr(singleEntryAddr))
		}
	}

	return s, nil
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
func (s *StreamerClient) Start() error {
	eventCh, eventErrorCh := make(chan *proto.Event), make(chan error)

	s.mutex.Lock()

	s.listener = s.newListener(
		eventCh,
		s.apiClient,
		s.sequence,
		s.handlers...,
	)
	defer s.listener.Stop()

	manager, err := s.newManager(s.watchCfg, s.watchOpts...)
	if err != nil {
		return err
	}

	s.manager = manager
	s.isInitialized = false

	s.mutex.Unlock()

	listenCh := s.listener.Listen()

	_, err = s.manager.RegisterWatch(s.topicSelfLink, eventCh, eventErrorCh)
	if s.onStreamConnection != nil {
		s.onStreamConnection()
	}

	s.mutex.Lock()
	s.isInitialized = true
	s.mutex.Unlock()

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

// Status returns the health status
func (s *StreamerClient) Status() error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if !s.isInitialized {
		return nil
	}

	if s.manager == nil || s.listener == nil {
		return fmt.Errorf("stream client is not ready")
	}
	if ok := s.manager.Status(); !ok {
		return errors.ErrGrpcConnection
	}

	return nil
}

// Stop stops the StreamerClient
func (s *StreamerClient) Stop() {
	s.manager.CloseConn()
	s.listener.Stop()
}

// Healthcheck - health check for stream client
func (s *StreamerClient) Healthcheck(_ string) *hc.Status {
	if err := s.Status(); err != nil {
		return &hc.Status{
			Result:  hc.FAIL,
			Details: err.Error(),
		}
	}
	return &hc.Status{
		Result: hc.OK,
	}
}
