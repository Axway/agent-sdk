package agent

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/agent/stream"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// AgentStatus - status for Agent resource
const (
	AgentRunning   = "running"
	AgentStopped   = "stopped"
	AgentFailed    = "failed"
	AgentUnhealthy = "unhealthy"
)

// AgentResourceType - Holds the type for agent resource in Central
var AgentResourceType string

// APIValidator - Callback for validating the API
type APIValidator func(apiID, stageName string) bool

// ConfigChangeHandler - Callback for Config change event
type ConfigChangeHandler func()

type agentData struct {
	agentResourceManager resource.Manager

	apicClient       apic.Client
	cfg              config.CentralConfig
	agentFeaturesCfg config.AgentFeaturesConfig
	agentCfg         interface{}
	tokenRequester   auth.PlatformTokenGetter

	apiMap                     cache.Cache
	instanceMap                cache.Cache
	categoryMap                cache.Cache
	teamMap                    cache.Cache
	apiValidator               APIValidator
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	isInitialized              bool
}

var agent = agentData{}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
	return InitializeWithAgentFeatures(centralCfg, config.NewAgentFeaturesConfiguration())
}

// InitializeWithAgentFeatures - Initializes the agent with agent features
func InitializeWithAgentFeatures(centralCfg config.CentralConfig, agentFeaturesCfg config.AgentFeaturesConfig) error {
	// Only create the api map cache if it does not already exist
	if agent.apiMap == nil {
		agent.apiMap = cache.New()
	}
	if agent.instanceMap == nil {
		agent.instanceMap = cache.New()
	}
	if agent.categoryMap == nil {
		agent.categoryMap = cache.New()
	}
	if agent.teamMap == nil {
		agent.teamMap = cache.New()
	}

	err := checkRunningAgent()
	if err != nil {
		return err
	}

	err = config.ValidateConfig(agentFeaturesCfg)
	if err != nil {
		return err
	}
	agent.agentFeaturesCfg = agentFeaturesCfg

	// validate the central config
	if agentFeaturesCfg.ConnectionToCentralEnabled() {
		err = config.ValidateConfig(centralCfg)
		if err != nil {
			return err
		}
	}

	if centralCfg.GetUsageReportingConfig().IsOfflineMode() {
		// Offline mode does not need more initialization
		agent.cfg = centralCfg
		return nil
	}

	if agentFeaturesCfg.ConnectionToCentralEnabled() {
		err = initializeTokenRequester(centralCfg)
		if err != nil {
			return err
		}
		// Init apic client when the agent starts, and on config change.
		agent.apicClient = apic.New(centralCfg, agent.tokenRequester)
		agent.apicClient.AddCache(agent.categoryMap, agent.teamMap)

		if util.IsNotTest() {
			err = initEnvResources(centralCfg, agent.apicClient)
			if err != nil {
				return err
			}
		}
	}

	agent.cfg = centralCfg
	coreapi.SetConfigAgent(centralCfg.GetEnvironmentName(), isRunningInDockerContainer(), centralCfg.GetAgentName())

	if centralCfg.GetAgentName() != "" {
		if agent.agentResourceManager == nil {
			agent.agentResourceManager, err = resource.NewAgentResourceManager(agent.cfg, agent.apicClient, agent.agentResourceChangeHandler)
			if err != nil {
				return err
			}
		} else {
			agent.agentResourceManager.OnConfigChange(agent.cfg, agent.apicClient)
		}
	}

	if !agent.isInitialized {
		setupSignalProcessor()
		// only do the periodic healthcheck stuff if NOT in unit tests and running binary agents
		if util.IsNotTest() && !isRunningInDockerContainer() {
			hc.StartPeriodicHealthCheck()
		}

		if util.IsNotTest() && agent.agentFeaturesCfg.ConnectionToCentralEnabled() {
			StartAgentStatusUpdate()
			startAPIServiceCache()
			startTeamACLCache()
		}
		// Set agent running
		if agent.agentResourceManager != nil {
			UpdateStatusWithPrevious(AgentRunning, "", "")
		}
	}

	agent.isInitialized = true
	return nil
}

func initEnvResources(cfg config.CentralConfig, client apic.Client) error {
	env, err := client.GetEnvironment()
	if err != nil {
		return err
	}

	cfg.SetAxwayManaged(env.Spec.AxwayManaged)
	if cfg.GetEnvironmentID() == "" {
		// need to save this ID for the traceability agent for later
		cfg.SetEnvironmentID(env.Metadata.ID)
	}

	if cfg.GetTeamID() == "" {
		team, err := client.GetCentralTeamByName(cfg.GetTeamName())
		if err != nil {
			return err
		}

		cfg.SetTeamID(team.ID)
	}

	return nil
}

func checkRunningAgent() error {
	// Check only on startup of binary agents
	if !agent.isInitialized && util.IsNotTest() && !isRunningInDockerContainer() {
		return hc.CheckIsRunning()
	}
	return nil
}

// InitializeForTest - Initialize for test
func InitializeForTest(apicClient apic.Client) {
	agent.apiMap = cache.New()
	agent.apicClient = apicClient
}

// GetConfigChangeHandler - returns registered config change handler
func GetConfigChangeHandler() ConfigChangeHandler {
	return agent.configChangeHandler
}

// OnConfigChange - Registers handler for config change event
func OnConfigChange(configChangeHandler ConfigChangeHandler) {
	agent.configChangeHandler = configChangeHandler
}

// OnAgentResourceChange - Registers handler for resource change event
func OnAgentResourceChange(agentResourceChangeHandler ConfigChangeHandler) {
	agent.agentResourceChangeHandler = agentResourceChangeHandler
}

func startAPIServiceCache() {
	instanceCacheLock := &sync.Mutex{}

	// register the update cache job
	newDiscoveryCacheJob := newDiscoveryCache(agent.agentResourceManager, false, instanceCacheLock)
	if !agent.cfg.IsUsingGRPC() {
		hc.RegisterHealthcheck(util.AmplifyCentral, "central", agent.apicClient.Healthcheck)

		id, err := jobs.RegisterIntervalJobWithName(newDiscoveryCacheJob, agent.cfg.GetPollInterval(), "New APIs Cache")
		if err != nil {
			log.Errorf("could not start the New APIs cache update job: %v", err.Error())
			return
		}
		// Start the full update after the first interval
		go startDiscoveryCache(instanceCacheLock)
		log.Tracef("registered API cache update job: %s", id)
	} else {
		// Load cache from API initially. Following updates to cache will be done using watch events
		err := newDiscoveryCacheJob.Execute()
		if err != nil {
			log.Error(err)
			return
		}

		err = startStreamMode(agent)
		if err != nil {
			log.Error(err)
			return
		}
	}

	if agent.apiValidator != nil {
		instanceValidator := newInstanceValidator(instanceCacheLock, !agent.cfg.IsUsingGRPC())
		_, err := jobs.RegisterIntervalJobWithName(instanceValidator, agent.cfg.GetPollInterval(), "API service instance validator")
		if err != nil {
			log.Error(err)
			return
		}
	}
}

func startTeamACLCache() {
	// register the team cache and acl update jobs
	var teamChannel chan string

	// Only discovery agents need to start the ACL handler
	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		teamChannel = make(chan string)
		registerAccessControlListHandler(teamChannel)
	}

	registerTeamMapCacheJob(teamChannel)
}

func isRunningInDockerContainer() bool {
	// Within the cgroup file, if you are not in a docker container all entries are like this devices:/
	// If in a docker container, entries are like this: devices:/docker/xxxxxxxxx.
	// So, all we need to do is see if ":/docker" exists somewhere in the file.
	bytes, err := ioutil.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	// Convert []byte to string and print to screen
	text := string(bytes)

	return strings.Contains(text, ":/docker")
}

// initializeTokenRequester - Create a new auth token requester
func initializeTokenRequester(centralCfg config.CentralConfig) error {
	var err error
	agent.tokenRequester = auth.NewPlatformTokenGetterWithCentralConfig(centralCfg)
	if util.IsNotTest() {
		_, err = agent.tokenRequester.GetToken()
	}
	return err
}

// GetCentralAuthToken - Returns the Auth token from AxwayID to make API call to Central
func GetCentralAuthToken() (string, error) {
	if agent.tokenRequester == nil {
		return "", apic.ErrAuthenticationCall
	}
	return agent.tokenRequester.GetToken()
}

// GetCentralClient - Returns the APIC Client
func GetCentralClient() apic.Client {
	return agent.apicClient
}

// GetCentralConfig - Returns the APIC Client
func GetCentralConfig() config.CentralConfig {
	return agent.cfg
}

// GetAPICache - Returns the cache
func GetAPICache() cache.Cache {
	if agent.apiMap == nil {
		agent.apiMap = cache.New()
	}
	return agent.apiMap
}

// GetAgentResource - Returns Agent resource
func GetAgentResource() *apiV1.ResourceInstance {
	if agent.agentResourceManager == nil {
		return nil
	}
	return agent.agentResourceManager.GetAgentResource()
}

// UpdateStatus - Updates the agent state
func UpdateStatus(status, description string) {
	UpdateStatusWithPrevious(status, status, description)
}

// UpdateStatusWithPrevious - Updates the agent state providing a previous state
func UpdateStatusWithPrevious(status, prevStatus, description string) {
	if agent.agentResourceManager != nil {
		err := agent.agentResourceManager.UpdateAgentStatus(status, prevStatus, description)
		if err != nil {
			log.Warnf("could not update the agent status reference, %s", err.Error())
		}
	}
}

func setupSignalProcessor() {
	if !agent.agentFeaturesCfg.ProcessSystemSignalsEnabled() {
		return
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigs
		cleanUp()
		log.Info("Stopping agent")
		os.Exit(0)
	}()
}

// cleanUp - AgentCleanup
func cleanUp() {
	UpdateStatusWithPrevious(AgentStopped, AgentRunning, "")
}

func startDiscoveryCache(instanceCacheLock *sync.Mutex) {
	time.Sleep(time.Hour)
	allDiscoveryCacheJob := newDiscoveryCache(agent.agentResourceManager, true, instanceCacheLock)
	id, err := jobs.RegisterIntervalJobWithName(allDiscoveryCacheJob, time.Hour, "All APIs Cache")
	if err != nil {
		log.Errorf("could not start the All APIs cache update job: %v", err.Error())
		return
	}
	log.Tracef("registered API cache update all job: %s", id)
}

// Todo - To be updated after cache persistence story
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

func newWatchManager(host, tenantID string, isInsecure bool, getToken auth.TokenGetter, wtName string) (wm.Manager, error) {
	u, _ := url.Parse(host)
	port := 443

	if u.Port() == "" {
		port, _ = net.LookupPort("tcp", u.Scheme)
	} else {
		port, _ = strconv.Atoi(u.Port())
	}

	cfg := &wm.Config{
		Host:        u.Host,
		Port:        uint32(port),
		TenantID:    tenantID,
		TokenGetter: getToken.GetToken,
	}

	entry := logrus.NewEntry(log.Get())

	var watchOptions []wm.Option
	watchOptions = append(watchOptions, wm.WithLogger(entry))

	if isInsecure {
		watchOptions = append(watchOptions, wm.WithTLSConfig(nil))
	}
	watchOptions = append(watchOptions, wm.WithSyncEvents(getAgentSequenceManager(wtName)))

	return wm.New(cfg, watchOptions...)
}

func startStreamMode(agent agentData) error {
	host := agent.cfg.GetURL()
	tenantID := agent.cfg.GetTenantID()
	insecure := agent.cfg.GetTLSConfig().IsInsecureSkipVerify()

	rc := stream.NewResourceClient(
		host+"/apis",
		tenantID,
		coreapi.NewClient(agent.cfg.GetTLSConfig(), agent.cfg.GetProxyURL()),
		agent.tokenRequester,
	)
	wt, err := getWatchTopic(rc)
	if err != nil {
		return err
	}

	manager, err := newWatchManager(host, tenantID, insecure, agent.tokenRequester, wt.Name)
	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	stopCh, events := make(chan interface{}), make(chan *proto.Event)

	eventListener := stream.NewEventListener(
		events,
		rc,
		handler.NewAPISvcHandler(agent.apiMap),
		handler.NewInstanceHandler(agent.instanceMap),
		handler.NewCategoryHandler(agent.categoryMap),
		handler.NewAgentResourceHandler(agent.agentResourceManager),
	)

	streamClient := stream.NewClient(
		wt.Metadata.SelfLink,
		manager,
		eventListener,
		events,
	)

	streamJob := stream.NewClientStreamJob(streamClient, stopCh)
	_, err = jobs.RegisterChannelJobWithName(streamJob, stopCh, "Stream Client")

	return err
}

func getWatchTopic(rc stream.ResourceClient) (*v1alpha1.WatchTopic, error) {
	env := agent.cfg.GetEnvironmentName()
	agentName := agent.cfg.GetAgentName()

	wtName := getWatchTopicName(env, agentName, agent.cfg.GetAgentType())
	wt, err := stream.GetCachedWatchTopic(cache.New(), wtName)
	if err != nil || wt == nil {
		wt, err = stream.GetOrCreateWatchTopic(wtName, env, rc, agent.cfg.GetAgentType())
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
	return "-" + resource.AgentTypesMap[agentType]
}
