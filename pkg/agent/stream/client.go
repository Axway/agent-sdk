package stream

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"

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

	sdkErrors "github.com/Axway/agent-sdk/pkg/util/errors"

	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
)

// StreamerClient client for starting a watch controller stream and handling the events
type StreamerClient struct {
	apiClient          events.APIClient
	handlers           []handler.Handler
	listener           *events.EventListener
	manager            wm.Manager
	newListener        events.NewListenerFunc
	newManager         wm.NewManagerFunc
	requestQueue       events.RequestQueue
	newRequestQueue    events.NewRequestQueueFunc
	onStreamConnection func()
	sequence           events.SequenceProvider
	topicSelfLink      string
	watchCfg           *wm.Config
	watchOpts          []wm.Option
	cacheManager       agentcache.Manager
	logger             log.FieldLogger
	environmentURL     string
	wt                 *management.WatchTopic
	harvester          harvester.Harvest
	onEventSyncError   func()
	mutex              sync.RWMutex
	isInitialized      atomic.Bool
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
		handlers:        handlers,
		apiClient:       apiClient,
		watchCfg:        watchCfg,
		newManager:      wm.New,
		newListener:     events.NewEventListener,
		newRequestQueue: events.NewRequestQueue,
		logger:          logger,
		environmentURL:  cfg.GetEnvironmentURL(),
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
	eventCh, requestCh, eventErrorCh := make(chan *proto.Event), make(chan *proto.Request, 1), make(chan error)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.listener = s.newListener(
		eventCh,
		s.apiClient,
		s.sequence,
		s.handlers...,
	)
	defer s.listener.Stop()

	s.requestQueue = s.newRequestQueue(requestCh)
	wmOptions := append(s.watchOpts, wm.WithRequestChannel(requestCh))
	defer s.requestQueue.Stop()

	manager, err := s.newManager(s.watchCfg, wmOptions...)
	if err != nil {
		return err
	}

	s.manager = manager
	s.isInitialized.Store(false)

	listenCh := s.listener.Listen()
	s.requestQueue.Start()

	_, err = s.manager.RegisterWatch(s.topicSelfLink, eventCh, eventErrorCh)
	if s.onStreamConnection != nil {
		s.onStreamConnection()
	}
	s.isInitialized.Store(true)

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
	if !s.isInitialized.Load() {
		return nil
	}

	if s.manager == nil || s.listener == nil || s.requestQueue == nil {
		return fmt.Errorf("stream client is not ready")
	}
	if ok := s.manager.Status(); !ok {
		return sdkErrors.ErrGrpcConnection
	}

	return nil
}

// Stop stops the StreamerClient
func (s *StreamerClient) Stop() {
	s.manager.CloseConn()
	s.listener.Stop()
	s.requestQueue.Stop()
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

func (s *StreamerClient) UpdateAgentStatus(state, prevState, message string) error {
	// Initial running status and stopped status set by watch-controller
	// allow running status after recovery from unhealthy state
	if canTransitionAgentState(state, prevState) {
		return s.writeStatusRequest(state, message)
	}
	s.logger.
		WithField("status", state).
		WithField("previousStatus", prevState).
		Trace("skipping agent status update request")
	return nil
}

func (s *StreamerClient) writeStatusRequest(state, message string) error {
	if s.canUpdateStatus() {
		req := &proto.Request{
			SelfLink:    s.topicSelfLink,
			RequestType: proto.RequestType_AGENT_STATUS.Enum(),
			AgentStatus: &proto.AgentStatus{
				State:   state,
				Message: message,
			},
		}
		return s.requestQueue.Write(req)
	}
	return errors.New("stream request queue is not active")
}

func (s *StreamerClient) canUpdateStatus() bool {
	return s.requestQueue != nil && s.requestQueue.IsActive()
}

func canTransitionAgentState(state, prevState string) bool {
	return state != prevState && state != "stopped"
}
