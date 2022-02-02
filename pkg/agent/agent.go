package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/agent/stream"
	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
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
	tokenRequester   auth.PlatformTokenGetter

	teamMap                    cache.Cache
	cacheManager               agentcache.Manager
	apiValidator               APIValidator
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	proxyResourceHandler       *handler.StreamWatchProxyHandler
	isInitialized              bool
}

var agent agentData

func init() {
	agent.proxyResourceHandler = handler.NewStreamWatchProxyHandler()
}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
	return InitializeWithAgentFeatures(centralCfg, config.NewAgentFeaturesConfiguration())
}

// InitializeWithAgentFeatures - Initializes the agent with agent features
func InitializeWithAgentFeatures(centralCfg config.CentralConfig, agentFeaturesCfg config.AgentFeaturesConfig) error {
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

	// Only create the api map cache if it does not already exist
	if agent.cacheManager == nil {
		agent.cacheManager = agentcache.NewAgentCacheManager(centralCfg, agentFeaturesCfg.PersistCacheEnabled())
	}

	if centralCfg.GetUsageReportingConfig().IsOfflineMode() {
		// Offline mode does not need more initialization
		agent.cfg = centralCfg
		return nil
	}

	agent.cfg = centralCfg
	api.SetConfigAgent(centralCfg.GetEnvironmentName(), isRunningInDockerContainer(), centralCfg.GetAgentName())

	if agentFeaturesCfg.ConnectionToCentralEnabled() {
		err = initializeTokenRequester(centralCfg)
		if err != nil {
			return err
		}

		// Init apic client when the agent starts, and on config change.
		agent.apicClient = apic.New(centralCfg, agent.tokenRequester, agent.cacheManager)

		if util.IsNotTest() {
			err = initEnvResources(centralCfg, agent.apicClient)
			if err != nil {
				return err
			}
		}

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
			startTeamACLCache(agent.cfg, agent.apicClient, agent.cacheManager)

			err := registerSubscriptionWebhook(agent.cfg.GetAgentType(), agent.apicClient)
			if err != nil {
				return errors.Wrap(errors.ErrRegisterSubscriptionWebhook, err.Error())
			}

			// Set agent running
			if agent.agentResourceManager != nil {
				UpdateStatusWithPrevious(AgentRunning, "", "")
			}
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
	agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
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

// RegisterResourceEventHandler - Registers handler for resource events
func RegisterResourceEventHandler(name string, resourceEventHandler handler.Handler) {
	agent.proxyResourceHandler.RegisterTargetHandler(name, resourceEventHandler)
}

// UnregisterResourceEventHandler - removes the specified resource event handler
func UnregisterResourceEventHandler(name string) {
	agent.proxyResourceHandler.UnregisterTargetHandler(name)
}

func startAPIServiceCache() {
	instanceCacheLock := &sync.Mutex{}

	// register the update cache job
	newDiscoveryCacheJob := newDiscoveryCache(agent.agentResourceManager, false, instanceCacheLock)
	if !agent.cfg.IsUsingGRPC() {
		// healthcheck for central in gRPC mode is registered by streamer
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
		if !agent.cacheManager.HasLoadedPersistedCache() {
			err := newDiscoveryCacheJob.Execute()
			if err != nil {
				log.Error(err)
				return
			}
			// trigger early saving for the initialized cache, following save will be done by interval job
			agent.cacheManager.SaveCache()
		}

		err := startStreamMode(agent)
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

func registerSubscriptionWebhook(at config.AgentType, client apic.Client) error {
	if at == config.DiscoveryAgent {
		return client.RegisterSubscriptionWebhook()
	}
	return nil
}

func startTeamACLCache(cfg config.CentralConfig, client apic.Client, caches agentcache.Manager) {
	// register the team cache and acl update jobs
	var teamChannel chan string

	// Only discovery agents need to start the ACL handler
	if cfg.GetAgentType() == config.DiscoveryAgent {
		teamChannel = make(chan string)
		registerAccessControlListHandler(teamChannel)
	}

	registerTeamMapCacheJob(teamChannel, caches, client)
}

func isRunningInDockerContainer() bool {
	// Within the cgroup file, if you are not in a docker container all entries are like these devices:/
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
	if agent.cacheManager == nil {
		agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, agent.agentFeaturesCfg.PersistCacheEnabled())
	}
	return agent.cacheManager.GetAPIServiceCache()
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

func startStreamMode(agent agentData) error {
	handlers := []handler.Handler{
		handler.NewAPISvcHandler(agent.cacheManager),
		handler.NewInstanceHandler(agent.cacheManager),
		handler.NewCategoryHandler(agent.cacheManager),
		handler.NewAgentResourceHandler(agent.agentResourceManager),
		agent.proxyResourceHandler,
	}

	cs, err := stream.NewStreamer(
		api.NewClient(agent.cfg.GetTLSConfig(), agent.cfg.GetProxyURL()),
		agent.cfg,
		agent.tokenRequester,
		agent.cacheManager,
		func(s stream.Streamer) {
			hc.RegisterHealthcheck(util.AmplifyCentral, "central", s.Healthcheck)
		},
		handlers...,
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	stream.NewClientStreamJob(cs)

	return err
}
