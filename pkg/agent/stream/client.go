package stream

import (
	"fmt"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/sirupsen/logrus"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	apiClient               events.APIClient
	handlers                []handler.Handler
	listener                *events.EventListener
	manager                 wm.Manager
	newListener             events.NewListenerFunc
	newManager              wm.NewManagerFunc
	onStreamConnection      OnStreamConnection
	seq                     events.SequenceProvider
	topicSelfLink           string
	watchCfg                *wm.Config
	watchOpts               []wm.Option
	cacheManager            agentcache.Manager
	loadOnStartup           []v1alpha1.WatchTopicSpecFilters
	resourcesOnStartupTimer *time.Timer
	fetchOnStartupPageSize  int
	fetchOnStartupRetention time.Duration
	logger                  log.FieldLogger
	environmentURL          string
}

// NewStreamerClient creates a StreamerClient
func NewStreamerClient(
	apiClient events.APIClient,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	onStreamConnection OnStreamConnection,
	onEventSyncError func(),
	handlers ...handler.Handler,
) (*StreamerClient, error) {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent.stream").
		WithComponent("Client")
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
		wm.WithProxy(cfg.GetProxyURL()),
		wm.WithEventSyncError(onEventSyncError),
	}

	if cfg.IsGRPCInsecure() {
		watchOpts = append(watchOpts, wm.WithTLSConfig(nil))
	} else {
		watchOpts = append(watchOpts, wm.WithTLSConfig(cfg.GetTLSConfig().BuildTLSConfig()))
	}

	if cfg.IsGRPCInsecure() {
		watchOpts = append(watchOpts, wm.WithTLSConfig(nil))
	} else {
		watchOpts = append(watchOpts, wm.WithTLSConfig(cfg.GetTLSConfig().BuildTLSConfig()))
	}

	if cfg.GetSingleURL() != "" {
		singleEntryURL, err := url.Parse(cfg.GetSingleURL())
		if err == nil {
			singleEntryAddr := util.ParseAddr(singleEntryURL)
			wm.WithSingleEntryAddr(singleEntryAddr)
		}
	}

	fetchOnStartup := make([]v1alpha1.WatchTopicSpecFilters, 0)
	if cfg.IsFetchOnStartupEnabled() {
		for _, filter := range wt.Spec.Filters {
			for _, ftype := range filter.Type {
				if filter.Scope.Kind == v1alpha1.EnvironmentGVK().Kind &&
					(ftype == events.WatchTopicFilterTypeCreated || ftype == events.WatchTopicFilterTypeUpdated) {
					fetchOnStartup = append(fetchOnStartup, filter)
					break
				}
			}
		}
	}

	s := &StreamerClient{
		handlers:                handlers,
		apiClient:               apiClient,
		topicSelfLink:           wt.Metadata.SelfLink,
		watchCfg:                watchCfg,
		watchOpts:               watchOpts,
		loadOnStartup:           fetchOnStartup,
		newManager:              wm.New,
		newListener:             events.NewEventListener,
		seq:                     seq,
		onStreamConnection:      onStreamConnection,
		cacheManager:            cacheManager,
		logger:                  logger,
		environmentURL:          cfg.GetEnvironmentURL(),
		fetchOnStartupPageSize:  cfg.GetFetchOnStartupPageSize(),
		fetchOnStartupRetention: cfg.GetFetchOnStartupRetention(),
	}

	if cfg.IsFetchOnStartupEnabled() {
		log.Debug("[INIT] Fetch and cache watch topic resources")
		loadErr := cacheStartupResources(s)
		if loadErr != nil {
			return nil, loadErr
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

	s.listener = s.newListener(
		eventCh,
		s.apiClient,
		s.seq,
		s.handlers...,
	)

	manager, err := s.newManager(s.watchCfg, s.watchOpts...)
	if err != nil {
		return err
	}

	s.manager = manager

	listenCh := s.listener.Listen()

	_, err = s.manager.RegisterWatch(s.topicSelfLink, eventCh, eventErrorCh)
	if err != nil {
		return err
	}

	if s.onStreamConnection != nil {
		s.onStreamConnection(s)
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

// HandleFetchOnStartupResources retrieve fetch-on-start up resource from the cache
// and call events.EventListener HandleResource(...)
func (s *StreamerClient) HandleFetchOnStartupResources() {

	resources := s.cacheManager.GetAllFetchOnStartupResources()
	s.logger.Infof("Calling handlers for %d fetch-on-startup resource(s)", len(resources))

	for _, instance := range resources {
		metadata := &proto.EventMeta{}
		ctx := handler.NewEventContext(proto.Event_CREATED, metadata, instance.Name, instance.Kind)
		s.listener.HandleResource(ctx, metadata, instance)
	}

	s.logger.Infof("Evicting fetch-on-startup cache")
	if err := s.cacheManager.DeleteAllFetchOnStartupResources(); err != nil {
		s.logger.Errorf("Error evicting fetch-on-startup cache: %v", err)
	}
	if s.resourcesOnStartupTimer != nil {
		s.resourcesOnStartupTimer.Stop()
	}

}

func cacheStartupResources(s *StreamerClient) error {

	s.logger.Infof("Caching watch-topic %d resource(s)", len(s.loadOnStartup))
	err := s.cacheManager.DeleteAllFetchOnStartupResources()
	if err != nil {
		return err
	}

	for _, filterSpec := range s.loadOnStartup {
		instances := s.fetchLatest(filterSpec.Kind, filterSpec.Name)
		s.cacheManager.AddFetchOnStartupResources(instances)
	}

	s.resourcesOnStartupTimer = time.AfterFunc(s.fetchOnStartupRetention, func() {
		if len(s.cacheManager.GetAllFetchOnStartupResources()) > 0 {
			s.logger.Warnf("Evicting fetch-on-startup cache as not consumed after %s", s.fetchOnStartupRetention.String())
			if e := s.cacheManager.DeleteAllFetchOnStartupResources(); e != nil {
				s.logger.Errorf("Error evicting fetch-on-startup cache after timeout: %v", e)
			}
		}
	})

	return nil

}

func (s *StreamerClient) fetchLatest(kind string, name string) []*apiv1.ResourceInstance {

	var urlKind, ok = apiv1.GetPluralFromKind(kind)
	if !ok {
		s.logger.Errorf("Resource Kind: %s is not handled.", kind)
		return make([]*apiv1.ResourceInstance, 0)
	}

	var resourcesURL string
	if name != "*" {
		resourcesURL = fmt.Sprintf("%s/%s/%s", s.environmentURL, urlKind, name)
		resource, err := s.apiClient.GetResource(resourcesURL)
		if err != nil {
			return make([]*apiv1.ResourceInstance, 0)
		}
		return []*apiv1.ResourceInstance{resource}
	}

	resourcesURL = fmt.Sprintf("%s/%s", s.environmentURL, urlKind)

	pageSize := s.fetchOnStartupPageSize

	resourceInstances, err := s.apiClient.GetAPIV1ResourceInstancesWithPageSize(make(map[string]string, 0), resourcesURL, pageSize)
	if err != nil {
		s.logger.Warnf("Can't load resources at: %s (page %d). Cause: %v", resourcesURL, err)
	}

	s.logger.Infof("Loaded %d resources from %s", len(resourceInstances), resourcesURL)

	return resourceInstances
}
