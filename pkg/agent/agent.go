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

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	apiV1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	hc "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"git.ecd.axway.org/apigov/service-mesh-agent/pkg/apicauth"
)

// AgentStatus - status for Agent resource
const (
	AgentRunning = "running"
	AgentStopped = "stopped"
	AgentFailed  = "failed"
)

// AgentResourceType - Holds the type for agent resource in Central
var AgentResourceType string

// dataplaneResourceTypeMap - Agent Resource map
var dataplaneResourceTypeMap = map[string]string{
	v1alpha1.EdgeDiscoveryAgentResource:    v1alpha1.EdgeDataplaneResource,
	v1alpha1.EdgeTraceabilityAgentResource: v1alpha1.EdgeDataplaneResource,
	v1alpha1.AWSDiscoveryAgentResource:     v1alpha1.AWSDataplaneResource,
	v1alpha1.AWSTraceabilityAgentResource:  v1alpha1.AWSDataplaneResource,
}

type agentData struct {
	agentResource     *apiV1.ResourceInstance
	dataplaneResource *apiV1.ResourceInstance
	apicClient        apic.Client
	cfg               *config.CentralConfiguration
	tokenRequester    *apicauth.PlatformTokenGetter
	loggerName        string
	logLevel          string
	logFormat         string
	logOutput         string
	logPath           string
}

var agent = agentData{}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
	agent.cfg = centralCfg.(*config.CentralConfiguration)

	// validate the central config
	err := config.ValidateConfig(centralCfg)
	if err != nil {
		return err
	}

	// Init apic client
	agent.apicClient = apic.New(centralCfg)
	initalizeTokenRequester(centralCfg)

	// Get Agent Resources
	err = RefreshResources()
	if err != nil {
		return err
	}

	// merge agent resource config with central config
	mergeResourceWithConfig()
	// Do we still want to validate central config after merge???

	setupSignalProcessor()
	updateAgentStatus(AgentRunning, "")

	// only do the periodic healthcheck stuff if NOT in unit tests, or the tests will fail
	if flag.Lookup("test.v") == nil {
		// only do continuous healthchecking in binary agents
		if !isRunningInDockerContainer() {
			go runPeriodicHealthChecks()
		}
	}

	return nil
}

func runPeriodicHealthChecks() {
	for {
		if hc.RunChecks() != hc.OK {
			log.Error(errors.ErrHealthCheck)
			os.Exit(1)
		}
		// Set sleep time based on configured interval
		time.Sleep(hc.GetStatusConfig().GetHealthCheckInterval())
	}
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

// initalizeTokenRequester - Create a new auth token requestor
func initalizeTokenRequester(centralCfg config.CentralConfig) {
	agent.tokenRequester = apicauth.NewPlatformTokenGetter(
		centralCfg.GetAuthConfig().GetPrivateKey(),
		centralCfg.GetAuthConfig().GetPublicKey(),
		centralCfg.GetAuthConfig().GetKeyPassword(),
		centralCfg.GetAuthConfig().GetTokenURL(),
		centralCfg.GetAuthConfig().GetAudience(),
		centralCfg.GetAuthConfig().GetClientID(),
		centralCfg.GetAuthConfig().GetTimeout(),
	)
}

// GetCentralAuthToken - Returns the Auth token from AxwayID to make API call to Central
func GetCentralAuthToken() (string, error) {
	if agent.tokenRequester == nil {
		return "", apic.ErrAuthenticationCall
	}
	return agent.tokenRequester.GetToken()
}

// SetupLogging - Setup agent logging
func SetupLogging(agentName, logLevel, logFormat, logOutput, logPath string) {
	if agentName != "" {
		agent.loggerName = agentName
	}
	if logLevel != "" {
		agent.logLevel = logLevel
	}
	if logFormat != "" {
		agent.logFormat = logFormat
	}
	if logOutput != "" {
		agent.logOutput = logOutput
	}
	if logPath != "" {
		agent.logPath = logPath
	}

	log.SetupLogging(agent.loggerName, agent.logLevel, agent.logFormat, agent.logOutput, agent.logPath)
}

// GetAPICClient - Returns the apic client
func GetAPICClient() apic.Client {
	return agent.apicClient
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

// RefreshResources - Gets the agent and dataplane resources from API server
func RefreshResources() error {
	// IMP - To be removed once the model is in production
	if agent.cfg.GetAgentName() == "" {
		return nil
	}

	var err error
	agent.agentResource, err = getAgentResource()
	if err != nil {
		return err
	}
	// Get Dataplane Resources
	agent.dataplaneResource, err = getDataplaneResource(agent.agentResource)
	if err != nil {
		return err
	}
	return nil
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
		agentRes := edgeTraceabilitAgent(res)
		return agentRes.Spec.Dataplane
	case v1alpha1.AWSDiscoveryAgentResource:
		agentRes := awsDiscoveryAgent(res)
		return agentRes.Spec.Dataplane
	case v1alpha1.AWSTraceabilityAgentResource:
		agentRes := awsTraceabilitAgent(res)
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
	SetupLogging("", logLevel, "", "", "")
}
