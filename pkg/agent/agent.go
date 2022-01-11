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

	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/agent/stream"
	"github.com/Axway/agent-sdk/pkg/api"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
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

// DeleteServiceValidator - Callback for validating if the service should be deleted along with the consumer instance
type DeleteServiceValidator func(apiID, stageName string) bool

// ConfigChangeHandler - Callback for Config change event
type ConfigChangeHandler func()

type agentData struct {
	agentResourceManager resource.Manager

	apicClient     apic.Client
	cfg            config.CentralConfig
	tokenRequester auth.PlatformTokenGetter

	apiMap                     cache.Cache
	instanceMap                cache.Cache
	categoryMap                cache.Cache
	cacheMap                   cache.Cache
	apiValidator               APIValidator
	deleteServiceValidator     DeleteServiceValidator
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	proxyResourceHandler       *handler.StreamWatchProxyHandler
	isInitialized              bool
}

var agent = agentData{
	proxyResourceHandler: handler.NewStreamWatchProxyHandler(),
}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
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
	if agent.cacheMap == nil {
		agent.cacheMap = cache.New()

		err := agent.cacheMap.Load("offlineCache" + ".cache")
		if err != nil {
			agent.cacheMap.SetWithSecondaryKey("cachesCache", "apiServiceInstancesKey", &agent.apiMap)
			agent.cacheMap.SetWithSecondaryKey("cachesCache", "apiServicesKey", &agent.instanceMap)
			agent.cacheMap.SetWithSecondaryKey("cachesCache", "categoriesKey", &agent.categoryMap)

			if util.IsNotTest() {
				agent.cacheMap.Save("offlineCache" + ".cache")
			}
		}
		//
		//
		// else {
		// 	fmt.Println("TODO: load caches into offlineCache")

		// 	apiCache, err := agent.cacheMap.GetBySecondaryKey("apiServiceInstancesKey")
		// 	if err != nil {
		// 		agent.apiMap = apiCache
		// 	}
		// 	fmt.Println("err: ", err)
		// 	fmt.Println("apiCache: ", apiCache)

		// 	instanceCache, err := agent.cacheMap.GetBySecondaryKey("apiServicesKey")
		// 	fmt.Println("err: ", err)
		// 	fmt.Println("instanceCache: ", instanceCache)
		// 	categoriesCache, err := agent.cacheMap.GetBySecondaryKey("categoriesKey")
		// 	fmt.Println("err: ", err)
		// 	fmt.Println("categoriesCache: ", categoriesCache)
	}

	err := checkRunningAgent()
	if err != nil {
		return err
	}

	// validate the central config
	err = config.ValidateConfig(centralCfg)
	if err != nil {
		return err
	}

	if centralCfg.GetUsageReportingConfig().IsOfflineMode() {
		// Offline mode does not need more initialization
		agent.cfg = centralCfg
		return nil
	}

	err = initializeTokenRequester(centralCfg)
	if err != nil {
		return err
	}

	// Init apic client when the agent starts, and on config change.
	agent.apicClient = apic.New(centralCfg, agent.tokenRequester)
	agent.apicClient.AddCategoryCache(agent.categoryMap)
	agent.apicClient.AddCachesCache(agent.cacheMap)

	if util.IsNotTest() {
		err = initEnvResources(centralCfg, agent.apicClient)
		if err != nil {
			return err
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

		if util.IsNotTest() {
			StartAgentStatusUpdate()
			startAPIServiceCache()
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
	if agent.apiMap == nil {
		agent.apiMap = cache.New()
	}
	return agent.apiMap
}

// GetCachesCache - Returns the cache of caches
func GetCachesCache() cache.Cache {
	if agent.cacheMap == nil {
		agent.cacheMap = cache.New()
	}
	return agent.cacheMap
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
		handler.NewAPISvcHandler(agent.apiMap),
		handler.NewInstanceHandler(agent.instanceMap),
		handler.NewCategoryHandler(agent.categoryMap),
		handler.NewCacheHandler(agent.cacheMap),
		handler.NewAgentResourceHandler(agent.agentResourceManager),
		agent.proxyResourceHandler,
	}

	cs, err := stream.NewStreamer(
		api.NewClient(agent.cfg.GetTLSConfig(), agent.cfg.GetProxyURL()),
		agent.cfg,
		agent.tokenRequester,
		handlers...,
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	stopCh := make(chan interface{})
	streamJob := stream.NewClientStreamJob(cs, stopCh)
	_, err = jobs.RegisterChannelJobWithName(streamJob, stopCh, "Stream Client")

	return err
}
