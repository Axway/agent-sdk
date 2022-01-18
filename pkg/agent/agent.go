package agent

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

// agentTypesMap - Agent Types map
var agentTypesMap = map[config.AgentType]string{
	config.DiscoveryAgent:    "discoveryagents",
	config.TraceabilityAgent: "traceabilityagents",
	config.GovernanceAgent:   "governanceagents",
}

type agentData struct {
	agentResource     *apiV1.ResourceInstance
	prevAgentResource *apiV1.ResourceInstance

	apicClient       apic.Client
	cfg              config.CentralConfig
	agentFeaturesCfg config.AgentFeaturesConfig
	agentCfg         interface{}
	tokenRequester   auth.PlatformTokenGetter
	loggerName       string
	logLevel         string
	logFormat        string
	logOutput        string
	logPath          string

	apiMap                     cache.Cache
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
		// Init apic client
		if agent.apicClient == nil {
			agent.apicClient = apic.New(centralCfg, agent.tokenRequester)
			agent.apicClient.AddCache(agent.categoryMap, agent.teamMap)
		} else {
			agent.apicClient.SetTokenGetter(agent.tokenRequester)
			agent.apicClient.OnConfigChange(centralCfg)
		}
	}
	agent.cfg = centralCfg
	coreapi.SetConfigAgent(centralCfg.GetEnvironmentName(), isRunningInDockerContainer(), centralCfg.GetAgentName())

	if agent.isInitialized {
		mergeResourceWithConfig()
	}

	if !agent.isInitialized {
		if getAgentResourceType() != "" {
			fetchConfig()
			updateAgentStatus(AgentRunning, "", "")
		} else if agent.agentFeaturesCfg.ConnectionToCentralEnabled() && agent.cfg.GetAgentName() != "" {
			return errors.Wrap(apic.ErrCentralConfig, "Agent name cannot be set. Config is used only for agents with API server resource definition")
		}

		setupSignalProcessor()
		// only do the periodic healthcheck stuff if NOT in unit tests and running binary agents
		if util.IsNotTest() && !isRunningInDockerContainer() {
			hc.StartPeriodicHealthCheck()
		}

		if util.IsNotTest() && agent.agentFeaturesCfg.ConnectionToCentralEnabled() {
			StartAgentStatusUpdate()
			startAPIServiceCache()
			startTeamACLCache()
			err := registerSubscriptionWebhook(agent.cfg.GetAgentType(), agent.apicClient)
			if err != nil {
				return errors.Wrap(errors.ErrRegisterSubscriptionWebhook, err.Error())
			}
		}
	}
	agent.isInitialized = true
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
	// register the update cache job
	newDiscoveryCacheJob := newDiscoveryCache(false)
	id, err := jobs.RegisterIntervalJobWithName(newDiscoveryCacheJob, agent.cfg.GetPollInterval(), "New APIs Cache")
	if err != nil {
		log.Errorf("could not start the New APIs cache update job: %v", err.Error())
		return
	}
	log.Tracef("registered API cache update job: %s", id)

	// Start the full update after the first interval
	go func() {
		time.Sleep(time.Hour)
		allDiscoveryCacheJob := newDiscoveryCache(true)
		id, err := jobs.RegisterIntervalJobWithName(allDiscoveryCacheJob, time.Hour, "All APIs Cache")
		if err != nil {
			log.Errorf("could not start the All APIs cache update job: %v", err.Error())
			return
		}
		log.Tracef("registered API cache update all job: %s", id)
	}()
}

func registerSubscriptionWebhook(at config.AgentType, client apic.Client) error {
	if at == config.DiscoveryAgent {
		return client.RegisterSubscriptionWebhook()
	}
	return nil
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

// initializeTokenRequester - Create a new auth token requestor
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
	return agent.agentResource
}

// UpdateStatus - Updates the agent state
func UpdateStatus(status, description string) {
	// send the current status as the previous
	updateAgentStatus(status, status, description)
}

// UpdateStatusWithPrevious - Updates the agent state providing a previous state
func UpdateStatusWithPrevious(status, prevStatus, description string) {
	updateAgentStatus(status, prevStatus, description)
}

func fetchConfig() error {
	// Get Agent Resources
	isChanged, err := refreshResources()
	if err != nil {
		return err
	}

	if isChanged {
		// merge agent resource config with central config
		mergeResourceWithConfig()
		if agent.agentResourceChangeHandler != nil {
			agent.agentResourceChangeHandler()
		}
	}
	return nil
}

// refreshResources - Gets the agent and dataplane resources from API server
func refreshResources() (bool, error) {
	// IMP - To be removed once the model is in production
	if agent.cfg.GetAgentName() == "" {
		return false, nil
	}
	var err error
	agent.agentResource, err = getAgentResource()
	if err != nil {
		return false, err
	}

	isChanged := agent.isInitialized
	if agent.prevAgentResource != nil {
		agentResHash, _ := util.ComputeHash(agent.agentResource)
		prevAgentResHash, _ := util.ComputeHash(agent.prevAgentResource)

		if prevAgentResHash == agentResHash {
			isChanged = false
		}
	}
	agent.prevAgentResource = agent.agentResource

	return isChanged, nil
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
	updateAgentStatus(AgentStopped, AgentRunning, "")
}

// GetAgentResourceType - Returns the Agent Resource path element
func getAgentResourceType() string {
	// Set resource for Agent Type
	return agentTypesMap[agent.cfg.GetAgentType()]
}

// GetAgentResource - returns the agent resource
func getAgentResource() (*apiV1.ResourceInstance, error) {
	agentResourceType := getAgentResourceType()
	agentResourceURL := agent.cfg.GetEnvironmentURL() + "/" + agentResourceType + "/" + agent.cfg.GetAgentName()

	response, err := agent.apicClient.ExecuteAPI(coreapi.GET, agentResourceURL, nil, nil)
	if err != nil {
		return nil, err
	}

	agent := apiV1.ResourceInstance{}
	json.Unmarshal(response, &agent)
	return &agent, nil
}

// updateAgentStatus - Updates the agent status in agent resource
func updateAgentStatus(status, prevStatus, message string) error {
	if agent.cfg == nil || !agent.agentFeaturesCfg.ConnectionToCentralEnabled() {
		return nil
	}
	// IMP - To be removed once the model is in production
	if agent.cfg == nil || agent.cfg.GetAgentName() == "" {
		return nil
	}

	if agent.agentResource != nil {
		agentResourceType := getAgentResourceType()
		resource := createAgentStatusSubResource(agentResourceType, status, prevStatus, message)

		err := updateAgentStatusAPI(resource, agentResourceType)
		if err != nil {
			log.Warn("Could not update the agent status reference")
			return err
		}
	}
	return nil
}

func updateAgentStatusAPI(resource interface{}, agentResourceType string) error {
	buffer, err := json.Marshal(resource)
	if err != nil {
		return nil
	}

	subResURL := agent.cfg.GetEnvironmentURL() + "/" + agentResourceType + "/" + agent.cfg.GetAgentName() + "/status"
	_, err = agent.apicClient.ExecuteAPI(coreapi.PUT, subResURL, nil, buffer)
	if err != nil {
		return err
	}
	return nil
}

func createAgentStatusSubResource(agentResourceType, status, prevStatus, message string) *v1.ResourceInstance {
	switch agentResourceType {
	case v1alpha1.DiscoveryAgentResourceName:
		agentRes := createDiscoveryAgentStatusResource(status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance
	case v1alpha1.TraceabilityAgentResourceName:
		agentRes := createTraceabilityAgentStatusResource(status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance
	case v1alpha1.GovernanceAgentResourceName:
		agentRes := createGovernanceAgentStatusResource(status, prevStatus, message)
		resourceInstance, _ := agentRes.AsInstance()
		return resourceInstance
	default:
		panic(ErrUnsupportedAgentType)
	}
}

func mergeResourceWithConfig() {
	// IMP - To be removed once the model is in production
	if agent.cfg.GetAgentName() == "" {
		return
	}

	switch getAgentResourceType() {
	case v1alpha1.DiscoveryAgentResourceName:
		mergeDiscoveryAgentWithConfig(agent.cfg.(*config.CentralConfiguration))
	case v1alpha1.TraceabilityAgentResourceName:
		mergeTraceabilityAgentWithConfig(agent.cfg.(*config.CentralConfiguration))
	case v1alpha1.GovernanceAgentResourceName:
		mergeGovernanceAgentWithConfig(agent.cfg.(*config.CentralConfiguration))
	default:
		panic(ErrUnsupportedAgentType)
	}
}

func applyResConfigToCentralConfig(cfg *config.CentralConfiguration, resCfgAdditionalTags, resCfgTeamName, resCfgLogLevel string) {
	if cfg.TagsToPublish == "" && resCfgAdditionalTags != "" {
		cfg.TagsToPublish = resCfgAdditionalTags
	}

	logLevel := agent.logLevel
	if strings.ToUpper(agent.logLevel) == "INFO" && strings.ToUpper(resCfgLogLevel) != "INFO" {
		logLevel = resCfgLogLevel
	}
	agent.logLevel = logLevel
	if logLevel != "" {
		log.GlobalLoggerConfig.Level(logLevel).Apply()
	}

	// If config team is blank, check resource team name.  If resource team name is not blank, use resource team name
	if cfg.TeamName == "" && resCfgTeamName != "" {
		cfg.TeamName = resCfgTeamName
	}
}
