package stream

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

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

	utilError "github.com/Axway/agent-sdk/pkg/util/errors"

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
	requestQueue       events.RequestQueue
	newRequestQueue    events.NewRequestQueueFunc
	newManager         wm.NewManagerFunc
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
	isInitialized      bool
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

	s.listener = s.newListener(
		eventCh,
		s.apiClient,
		s.sequence,
		s.handlers...,
	)
	defer s.listener.Stop()

	s.requestQueue = s.newRequestQueue(requestCh)
	defer s.requestQueue.Stop()
	wmOptions := append(s.watchOpts, wm.WithRequestChannel(requestCh))

	manager, err := s.newManager(s.watchCfg, wmOptions...)
	if err != nil {
		return err
	}

	s.manager = manager
	s.isInitialized = false

	s.mutex.Unlock()

	listenCh := s.listener.Listen()
	s.requestQueue.Start()

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
		return utilError.ErrGrpcConnection
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

func (s *StreamerClient) CanUpdateStatus() bool {
	return s.requestQueue != nil && s.requestQueue.IsActive()
}

func canTransitionAgentState(state, prevState string) bool {
	return state != prevState && state != "stopped"
}

func (s *StreamerClient) UpdateAgentStatus(state, prevState, message string) error {
	if s.CanUpdateStatus() {
		// Initial running status and stopped status set by watch-controller
		// allow running status after recovery from unhealthy state
		if canTransitionAgentState(state, prevState) {
			req := &proto.Request{
				RequestType: proto.RequestType_AGENT_STATUS.Enum(),
				AgentStatus: &proto.AgentStatus{
					State:   state,
					Message: message,
				},
			}
			s.requestQueue.Write(req)
		} else {
			s.logger.
				WithField("status", state).
				WithField("previousStatus", prevState).
				Debug("skipping agent status update request")
		}
	} else {
		return errors.New("stream request queue is not active")
	}
	return nil
}
