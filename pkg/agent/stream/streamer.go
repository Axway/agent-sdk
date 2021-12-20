package stream

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/util/errors"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
)

// agentTypesMap - Agent Types map
var agentTypesMap = map[config.AgentType]string{
	config.DiscoveryAgent:    "discoveryagents",
	config.TraceabilityAgent: "traceabilityagents",
	config.GovernanceAgent:   "governanceagents",
}

// Streamer interface for starting a service
type Streamer interface {
	Start() error
	Status() error
	Stop()
}

// NewClientStreamJob creates a job for the stream client
func NewClientStreamJob(streamer Streamer, stop chan interface{}) jobs.Job {
	return &ClientStreamJob{
		streamer: streamer,
		stop:     stop,
	}
}

// ClientStreamJob job wrapper for a client that starts a stream and an event manager.
type ClientStreamJob struct {
	streamer Streamer
	stop     chan interface{}
}

// Execute starts the stream
func (j *ClientStreamJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
	}()

	return j.streamer.Start()
}

// Status gets the status
func (j *ClientStreamJob) Status() error {
	return j.streamer.Status()
}

// Ready checks if the job to start the stream is ready
func (j *ClientStreamJob) Ready() bool {
	return true
}

type centralStreamer struct {
	handlers      []Handler
	listener      Listener
	manager       wm.Manager
	rc            ResourceClient
	topicSelfLink string
	watchCfg      *wm.Config
	watchOpts     []wm.Option
}

func (c *centralStreamer) Start() error {
	events := make(chan *proto.Event)
	eventErrorCh := make(chan error)

	c.listener = NewEventListener(
		events,
		c.rc,
		c.handlers...,
	)

	manager, err := wm.New(c.watchCfg, c.watchOpts...)
	if err != nil {
		return err
	}
	c.manager = manager

	errCh := make(chan error)

	listenCh := c.listener.Listen()
	go func() {
		_, err := c.manager.RegisterWatch(c.topicSelfLink, events, eventErrorCh)
		if err != nil {
			errCh <- err
		}
	}()

	var clientError error

	select {
	case err := <-listenCh:
		clientError = err
	case err := <-eventErrorCh:
		clientError = err
	case err := <-errCh:
		clientError = err
	}

	return clientError
}

func (c *centralStreamer) Status() error {
	if c.manager == nil || c.listener == nil {
		return fmt.Errorf("waiting to start")
	}
	if ok := c.manager.Status(); !ok {
		return errors.ErrGrpcConnection
	}

	return nil
}

func (c *centralStreamer) Stop() {
	c.manager.CloseConn()
	c.listener.Stop()
}

func NewCentralStreamer(
	cfg config.CentralConfig,
	getToken auth.PlatformTokenGetter,
	handlers ...Handler,
) (Streamer, error) {
	apiServerHost := cfg.GetURL() + "/apis"
	tenant := cfg.GetTenantID()
	isInsecure := cfg.GetTLSConfig().IsInsecureSkipVerify()

	rc := NewResourceClient(
		apiServerHost,
		tenant,
		api.NewClient(cfg.GetTLSConfig(), cfg.GetProxyURL()),
		getToken,
	)

	wt, err := getWatchTopic(cfg, rc)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(cfg.GetURL())
	port := 443

	if u.Port() == "" {
		port, _ = net.LookupPort("tcp", u.Scheme)
	} else {
		port, _ = strconv.Atoi(u.Port())
	}

	watchCfg := &wm.Config{
		Host:        u.Host,
		Port:        uint32(port),
		TenantID:    tenant,
		TokenGetter: getToken.GetToken,
	}

	watchOpts := []wm.Option{
		wm.WithLogger(logrus.NewEntry(log.Get())),
		wm.WithSyncEvents(getAgentSequenceManager(wt.Name)),
	}

	if isInsecure {
		watchOpts = append(watchOpts, wm.WithTLSConfig(nil))
	}

	return &centralStreamer{
		handlers:      handlers,
		rc:            rc,
		topicSelfLink: wt.Metadata.SelfLink,
		watchCfg:      watchCfg,
		watchOpts:     watchOpts,
	}, nil
}

func getWatchTopic(cfg config.CentralConfig, rc ResourceClient) (*v1alpha1.WatchTopic, error) {
	env := cfg.GetEnvironmentName()
	agentName := cfg.GetAgentName()

	wtName := getWatchTopicName(env, agentName, cfg.GetAgentType())
	wt, err := getCachedWatchTopic(cache.New(), wtName)
	if err != nil || wt == nil {
		wt, err = getOrCreateWatchTopic(wtName, env, rc, cfg.GetAgentType())
		if err != nil {
			return nil, err
		}
		// cache the watch topic
	}
	return wt, err
}

func getWatchTopicName(envName, agentName string, agentType config.AgentType) string {
	wtName := agentName
	if wtName == "" {
		wtName = envName
	}
	return wtName + getWatchTopicNameSuffix(agentType)
}

func getWatchTopicNameSuffix(agentType config.AgentType) string {
	return "-" + agentTypesMap[agentType]
}

type agentSequenceManager struct {
	sequenceCache cache.Cache
}

func (s *agentSequenceManager) GetSequence() int64 {
	if s.sequenceCache != nil {
		cachedSeqID, err := s.sequenceCache.Get("watchSequenceID")
		if err == nil {
			if seqID, ok := cachedSeqID.(float64); ok {
				return int64(seqID)
			}
		}
	}
	return 0
}

func getAgentSequenceManager(watchTopicName string) *agentSequenceManager {
	seqCache := cache.New()
	if watchTopicName != "" {
		err := seqCache.Load(watchTopicName + ".sequence")
		if err != nil {
			seqCache.Set("watchSequenceID", int64(0))
			seqCache.Save(watchTopicName + ".sequence")
		}
	}
	return &agentSequenceManager{sequenceCache: seqCache}
}
