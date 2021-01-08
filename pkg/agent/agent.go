package agent

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	apiV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// AgentStatus - status for Agent resource
const (
	AgentRunning = "running"
	AgentStopped = "stopped"
	AgentFailed  = "failed"
)

// AgentResourceType - Holds the type for agent resource in Central
var AgentResourceType string

// APIValidator - Callback for validating the API
type APIValidator func(apiID, stageName string) bool

// ConfigChangeHandler - Callback for Config change event
type ConfigChangeHandler func()

// dataplaneResourceTypeMap - Agent Resource map
var dataplaneResourceTypeMap = map[string]string{
	v1alpha1.EdgeDiscoveryAgentResource:    v1alpha1.EdgeDataplaneResource,
	v1alpha1.EdgeTraceabilityAgentResource: v1alpha1.EdgeDataplaneResource,
	v1alpha1.AWSDiscoveryAgentResource:     v1alpha1.AWSDataplaneResource,
	v1alpha1.AWSTraceabilityAgentResource:  v1alpha1.AWSDataplaneResource,
}

type agentData struct {
	agentResource         *apiV1.ResourceInstance
	dataplaneResource     *apiV1.ResourceInstance
	prevAgentResource     *apiV1.ResourceInstance
	prevDataplaneResource *apiV1.ResourceInstance

	apicClient     apic.Client
	cfg            *config.CentralConfiguration
	agentCfg       interface{}
	tokenRequester auth.PlatformTokenGetter
	loggerName     string
	logLevel       string
	logFormat      string
	logOutput      string
	logPath        string

	apiMap                     cache.Cache
	apiValidator               APIValidator
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	isInitialized              bool
}

var agent = agentData{}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
	agent.apiMap = cache.New()

	agent.cfg = centralCfg.(*config.CentralConfiguration)

	// validate the central config
	err := config.ValidateConfig(centralCfg)
	if err != nil {
		return err
	}

	err = initializeTokenRequester(centralCfg)
	if err != nil {
		return err
	}
	// Init apic client
	if agent.apicClient == nil {
		agent.apicClient = apic.New(centralCfg, agent.tokenRequester)
	} else {
		agent.apicClient.SetTokenGetter(agent.tokenRequester)
		agent.apicClient.OnConfigChange(centralCfg)
	}

	if !agent.isInitialized {
		if getAgentResourceType() != "" {
			fetchConfig()
			updateAgentStatus(AgentRunning, "")
		} else if agent.cfg.AgentName != "" {
			return errors.Wrap(apic.ErrCentralConfig, "Agent name cannot be set. Config is used only for agents with API server resource definition")
		}

		setupSignalProcessor()
		// only do the periodic healthcheck stuff if NOT in unit tests, or the tests will fail
		if flag.Lookup("test.v") == nil {
			// only do continuous healthchecking in binary agents
			if !isRunningInDockerContainer() {
				go runPeriodicHealthChecks()
			}
		}

		startAPIServiceCache()
	}
	agent.isInitialized = true
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

func runPeriodicHealthChecks() {
	for {
		// Initial check done by the agents startup, so wait for the next interval
		// Use the default wait time of 30s if status config is not set yet
		waitInterval := 30 * time.Second
		if hc.GetStatusConfig() != nil {
			waitInterval = hc.GetStatusConfig().GetHealthCheckInterval()
		}
		// Set sleep time based on configured interval
		time.Sleep(waitInterval)
		if hc.RunChecks() != hc.OK {
			log.Error(errors.ErrHealthCheck)
			os.Exit(1)
		}
	}
}

func startAPIServiceCache() {
	// Load the cache before the agents start discovering the APIs from remote gateway
	updateAPICache()

	// Start period update of the cache by querying API server resources published by the agent
	go func() {
		for {
			time.Sleep(agent.cfg.PollInterval)
			updateAPICache()
			validateConsumerInstances()
			fetchConfig()
		}
	}()
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
	if strings.Contains(text, ":/docker") {
		return true
	}

	return false
}

// initializeTokenRequester - Create a new auth token requestor
func initializeTokenRequester(centralCfg config.CentralConfig) error {
	var err error
	agent.tokenRequester = auth.NewPlatformTokenGetterWithCentralConfig(centralCfg)
	if flag.Lookup("test.v") == nil {
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

// GetDataplaneResource - Returns dataplane resource
func GetDataplaneResource() *apiV1.ResourceInstance {
	return agent.dataplaneResource
}

// UpdateStatus - Updates the agent state
func UpdateStatus(status, description string) {
	updateAgentStatus(status, description)
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
	// Get Dataplane Resources
	agent.dataplaneResource, err = getDataplaneResource(agent.agentResource)
	if err != nil {
		return false, err
	}

	isChanged := true
	if agent.prevAgentResource != nil && agent.prevDataplaneResource != nil {
		agentResHash, _ := util.ComputeHash(agent.agentResource)
		dataplaneResHash, _ := util.ComputeHash(agent.dataplaneResource)

		prevAgentResHash, _ := util.ComputeHash(agent.prevAgentResource)
		prevDataplaneResHash, _ := util.ComputeHash(agent.prevDataplaneResource)
		if prevAgentResHash == agentResHash && prevDataplaneResHash == dataplaneResHash {
			isChanged = false
		}
	}

	agent.prevAgentResource = agent.agentResource
	agent.prevDataplaneResource = agent.dataplaneResource
	return isChanged, nil
}

func setupSignalProcessor() {
	// IMP - To be removed once the model is in production
	if agent.cfg.GetAgentName() == "" {
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigs
		cleanUp()
		os.Exit(0)
	}()
}

// cleanUp - AgentCleanup
func cleanUp() {
	updateAgentStatus(AgentStopped, "")
}

// GetAgentResourceType - Returns the Agent Resource path element
func getAgentResourceType() string {
	return AgentResourceType
}

// GetDataplaneType - Returns the Dataplane Resource path element
func getDataplaneType() string {
	res, _ := dataplaneResourceTypeMap[AgentResourceType]
	return res
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

// GetDataplaneResource - returns the dataplane resource
func getDataplaneResource(agentResource *apiV1.ResourceInstance) (*apiV1.ResourceInstance, error) {
	dataplaneName := getDataplaneNameFromAgent(agentResource)
	dataplaneResourceType := getDataplaneType()
	dataplaneResourceURL := agent.cfg.GetEnvironmentURL() + "/" + dataplaneResourceType + "/" + dataplaneName

	response, err := agent.apicClient.ExecuteAPI(coreapi.GET, dataplaneResourceURL, nil, nil)
	if err != nil {
		return nil, err
	}

	dataplane := apiV1.ResourceInstance{}

	json.Unmarshal(response, &dataplane)

	// Set the data plane name
	agent.cfg.SetDataPlaneName(dataplane.Title)

	return &dataplane, nil
}

// updateAgentStatus - Updates the agent status in agent resource
func updateAgentStatus(status, message string) error {
	// IMP - To be removed once the model is in production
	if agent.cfg == nil || agent.cfg.GetAgentName() == "" {
		return nil
	}

	if agent.agentResource != nil {
		agentResourceType := getAgentResourceType()
		resource := createAgentStatusSubResource(agentResourceType, status, message)
		buffer, err := json.Marshal(resource)
		if err != nil {
			return nil
		}

		subResURL := agent.cfg.GetEnvironmentURL() + "/" + agentResourceType + "/" + agent.cfg.GetAgentName() + "/status"
		_, err = agent.apicClient.ExecuteAPI(coreapi.PUT, subResURL, nil, buffer)
		if err != nil {
			return err
		}
	}
	return nil
}

func getDataplaneNameFromAgent(res *apiV1.ResourceInstance) string {
	switch getAgentResourceType() {
	case v1alpha1.EdgeDiscoveryAgentResource:
		agentRes := edgeDiscoveryAgent(res)
		return agentRes.Spec.Dataplane
	case v1alpha1.EdgeTraceabilityAgentResource:
		agentRes := edgeTraceabilityAgent(res)
		return agentRes.Spec.Dataplane
	case v1alpha1.AWSDiscoveryAgentResource:
		agentRes := awsDiscoveryAgent(res)
		return agentRes.Spec.Dataplane
	case v1alpha1.AWSTraceabilityAgentResource:
		agentRes := awsTraceabilityAgent(res)
		return agentRes.Spec.Dataplane
	default:
		panic("Unsupported agent type")
	}
}

func createAgentStatusSubResource(agentResourceType, status, message string) interface{} {
	switch agentResourceType {
	case v1alpha1.EdgeDiscoveryAgentResource:
		return createEdgeDiscoveryAgentStatusResource(status, message)
	case v1alpha1.EdgeTraceabilityAgentResource:
		return createEdgeTraceabilityAgentStatusResource(status, message)
	case v1alpha1.AWSDiscoveryAgentResource:
		return createAWSDiscoveryAgentStatusResource(status, message)
	case v1alpha1.AWSTraceabilityAgentResource:
		return createAWSTraceabilityAgentStatusResource(status, message)
	default:
		panic("Unsupported agent type")
	}
}

func mergeResourceWithConfig() {
	// IMP - To be removed once the model is in production
	if agent.cfg.GetAgentName() == "" {
		return
	}

	switch getAgentResourceType() {
	case v1alpha1.EdgeDiscoveryAgentResource:
		mergeEdgeDiscoveryAgentWithConfig(agent.cfg)
	case v1alpha1.EdgeTraceabilityAgentResource:
		mergeEdgeTraceabilityAgentWithConfig(agent.cfg)
	case v1alpha1.AWSDiscoveryAgentResource:
		mergeAWSDiscoveryAgentWithConfig(agent.cfg)
	case v1alpha1.AWSTraceabilityAgentResource:
		mergeAWSTraceabilityAgentWithConfig(agent.cfg)
	default:
		panic("Unsupported agent type")
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
}
