package stream

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/sirupsen/logrus"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/jobs"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/util/errors"

	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
)

// constants for retry interval for stream job
const (
	defaultRetryInterval = 100 * time.Millisecond
	maxRetryInterval     = 5 * time.Minute
	clientStreamJobName  = "Stream Client"
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
	Healthcheck(_ string) *hc.Status
}

// NewClientStreamJob creates a job for the streamer
func NewClientStreamJob(streamer Streamer) jobs.Job {
	streamJob := &ClientStreamJob{
		streamer:      streamer,
		stop:          make(chan interface{}),
		retryInterval: defaultRetryInterval,
	}
	streamJob.jobID, _ = jobs.RegisterDetachedChannelJobWithName(streamJob, streamJob.stop, clientStreamJobName)

	return streamJob
}

// ClientStreamJob job wrapper for a streamer that starts a stream and an event manager.
type ClientStreamJob struct {
	streamer      Streamer
	stop          chan interface{}
	jobID         string
	retryInterval time.Duration
}

// Execute starts the stream
func (j *ClientStreamJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
		j.renewRegistration()
	}()

	return j.streamer.Start()
}

// Status gets the status
func (j *ClientStreamJob) Status() error {
	status := j.streamer.Status()
	if status == nil {
		j.retryInterval = defaultRetryInterval
	}
	return status
}

// Ready checks if the job to start the stream is ready
func (j *ClientStreamJob) Ready() bool {
	return true
}

func (j *ClientStreamJob) renewRegistration() {
	if j.jobID != "" {
		jobs.UnregisterJob(j.jobID)
		j.jobID = ""

		j.retryInterval = j.retryInterval * 2
		if j.retryInterval > maxRetryInterval {
			j.retryInterval = defaultRetryInterval
		}

		time.AfterFunc(j.retryInterval, func() {
			j.jobID, _ = jobs.RegisterDetachedChannelJobWithName(j, j.stop, clientStreamJobName)
		})
		return
	}
}

// OnStreamConnection - callback streamer will invoke after stream connection is established
type OnStreamConnection func(Streamer)

type streamer struct {
	handlers           []handler.Handler
	listener           Listener
	manager            wm.Manager
	rc                 ResourceClient
	topicSelfLink      string
	watchCfg           *wm.Config
	watchOpts          []wm.Option
	newManager         wm.NewManagerFunc
	newListener        newListenerFunc
	sequenceManager    *agentSequenceManager
	onStreamConnection OnStreamConnection
}

// NewStreamer creates a Streamer
func NewStreamer(
	apiClient api.Client,
	cfg config.CentralConfig,
	getToken auth.TokenGetter,
	cacheManager agentcache.Manager,
	onStreamConnection OnStreamConnection,
	handlers ...handler.Handler,
) (Streamer, error) {
	apiServerHost := cfg.GetURL() + "/apis"
	tenant := cfg.GetTenantID()

	rc := NewResourceClient(
		apiServerHost,
		tenant,
		apiClient,
		getToken,
	)

	wt, err := getWatchTopic(cfg, rc)
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
	sequenceManager := newAgentSequenceManager(cacheManager, wt.Name)
	watchOpts := []wm.Option{
		wm.WithLogger(logrus.NewEntry(log.Get())),
		wm.WithSyncEvents(sequenceManager),
		wm.WithTLSConfig(cfg.GetTLSConfig().BuildTLSConfig()),
		wm.WithProxy(cfg.GetProxyURL()),
	}

	return &streamer{
		handlers:           handlers,
		rc:                 rc,
		topicSelfLink:      wt.Metadata.SelfLink,
		watchCfg:           watchCfg,
		watchOpts:          watchOpts,
		newManager:         wm.New,
		newListener:        NewEventListener,
		sequenceManager:    sequenceManager,
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
func (c *streamer) Start() error {
	events, eventErrorCh := make(chan *proto.Event), make(chan error)

	c.listener = c.newListener(
		events,
		c.rc,
		c.sequenceManager,
		c.handlers...,
	)

	manager, err := c.newManager(c.watchCfg, c.watchOpts...)
	if err != nil {
		return err
	}

	c.manager = manager

	listenCh := c.listener.Listen()
	_, err = c.manager.RegisterWatch(c.topicSelfLink, events, eventErrorCh)
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
func (c *streamer) Status() error {
	if c.manager == nil || c.listener == nil {
		return fmt.Errorf("stream client is not ready")
	}
	if ok := c.manager.Status(); !ok {
		return errors.ErrGrpcConnection
	}

	return nil
}

// Stop stops the streamer
func (c *streamer) Stop() {
	c.manager.CloseConn()
	c.listener.Stop()
}

// Healthcheck - healthchecker for stream client
func (c *streamer) Healthcheck(_ string) *hc.Status {
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
