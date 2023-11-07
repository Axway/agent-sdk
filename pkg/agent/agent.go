package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
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
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/migrate"
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
	// CorsField -
	CorsField = "cors"
	// RedirectURLsField -
	RedirectURLsField = "redirectURLs"
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
	apiValidatorLock           sync.Mutex
	apiValidatorJobID          string
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	proxyResourceHandler       *handler.StreamWatchProxyHandler
	isInitialized              bool

	provisioner          provisioning.Provisioning
	marketplaceMigration migrate.Migrator
	streamer             *stream.StreamerClient
	authProviderRegistry oauth.ProviderRegistry

	publishingLock *sync.Mutex
	ardLock        sync.Mutex

	// profiling
	profileDone chan struct{}
}

var agent agentData
var agentMutex sync.RWMutex

var logger log.FieldLogger

func init() {
	logger = log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("agent")
	agent.proxyResourceHandler = handler.NewStreamWatchProxyHandler()
	agentMutex = sync.RWMutex{}
	agent.publishingLock = &sync.Mutex{}
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
	centralCfg.SetIsMarketplaceSubsEnabled(agentFeaturesCfg.MarketplaceProvisioningEnabled())

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

	setCentralConfig(centralCfg)

	if centralCfg.GetUsageReportingConfig().IsOfflineMode() {
		// Offline mode does not need more initialization
		return nil
	}

	singleEntryFilter := []string{
		// Traceability host URL will be added by the traceability factory
		centralCfg.GetURL(),
		centralCfg.GetPlatformURL(),
		centralCfg.GetAuthConfig().GetTokenURL(),
		centralCfg.GetUsageReportingConfig().GetURL(),
	}
	api.SetConfigAgent(
		centralCfg.GetEnvironmentName(),
		centralCfg.IsUsingGRPC(),
		isRunningInDockerContainer(),
		centralCfg.GetAgentName(),
		centralCfg.GetSingleURL(),
		singleEntryFilter,
	)

	if agentFeaturesCfg.ConnectionToCentralEnabled() {
		err = handleCentralConfig(centralCfg)
		if err != nil {
			return err
		}
	}

	if !agent.isInitialized {
		err = handleInitialization()
		if err != nil {
			return err
		}
	}

	agent.isInitialized = true
	return nil
}

func handleCentralConfig(centralCfg config.CentralConfig) error {
	err := initializeTokenRequester(centralCfg)
	if err != nil {
		return fmt.Errorf("could not authenticate to Amplify, please check your keys and key password")
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
			agent.agentResourceManager, err = resource.NewAgentResourceManager(
				agent.cfg, agent.apicClient, agent.agentResourceChangeHandler,
			)
			if err != nil {
				return err
			}
		} else {
			agent.agentResourceManager.OnConfigChange(agent.cfg, agent.apicClient)
		}
	}

	return nil
}

func handleInitialization() error {
	setupSignalProcessor()
	// only do the periodic health check stuff if NOT in unit tests and running binary agents
	if util.IsNotTest() {
		hc.StartPeriodicHealthCheck()
	}

	if util.IsNotTest() && agent.agentFeaturesCfg.ConnectionToCentralEnabled() {
		// if credentials can expire and need to be deprovisioned then start the credential checker

		registerCredentialChecker()

		err := registerExternalIDPs()
		if err != nil {
			return err
		}

		startTeamACLCache()

		err = registerSubscriptionWebhook(agent.cfg.GetAgentType(), agent.apicClient)
		if err != nil {
			return errors.Wrap(errors.ErrRegisterSubscriptionWebhook, err.Error())
		}
	}

	return nil
}

// InitializeProfiling - setup the CPU and Memory profiling if options given
func InitializeProfiling(cpuProfile, memProfile string) {
	if memProfile != "" || cpuProfile != "" {
		setupProfileSignalProcessor(cpuProfile, memProfile)
	}
}

func registerExternalIDPs() error {
	if agent.cfg.GetAgentType() != config.TraceabilityAgent {
		idPCfg := agent.agentFeaturesCfg.GetExternalIDPConfig()

		proxy := agent.cfg.GetProxyURL()
		timeout := agent.cfg.GetClientTimeout()
		for _, idp := range idPCfg.GetIDPList() {
			tlsCfg := idp.GetTLSConfig()
			if idp.GetTLSConfig() == nil {
				tlsCfg = agent.cfg.GetTLSConfig()
			}
			err := registerCredentialProvider(idp, tlsCfg, proxy, timeout)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func registerCredentialProvider(idp config.IDPConfig, tlsCfg config.TLSConfig, proxyURL string, clientTimeout time.Duration) error {
	err := GetAuthProviderRegistry().RegisterProvider(idp, tlsCfg, proxyURL, clientTimeout)
	if err != nil {
		logger.
			WithField("name", idp.GetIDPName()).
			WithField("type", idp.GetIDPType()).
			WithField("metadata-url", idp.GetMetadataURL()).
			Errorf("unable to register external IdP provider, any credential request to the IdP will not be processed. %s", err.Error())
	}
	crdName := idp.GetIDPName() + " " + provisioning.OAuthIDPCRD

	crd, err := NewOAuthCredentialRequestBuilder(
		WithCRDName(crdName),
		WithCRDTitle(idp.GetIDPTitle()),
		WithCRDOAuthSecret(),
		WithCRDRequestSchemaProperty(getCorsSchemaPropertyBuilder()),
		WithCRDRequestSchemaProperty(getAuthRedirectSchemaPropertyBuilder()),
		WithCRDIsSuspendable(),
	).Register()
	if err != nil {
		logger.
			WithField("name", crd.Name).
			WithField("title", idp.GetIDPTitle()).
			Errorf("unable to create and register credential request definition. %s", err.Error())
	} else {
		logger.
			WithField("name", crd.Name).
			WithField("title", idp.GetIDPTitle()).
			Info("successfully created and registered credential request definition.")
	}
	return err
}

func getCorsSchemaPropertyBuilder() provisioning.PropertyBuilder {
	// register the supported credential request defs
	return provisioning.NewSchemaPropertyBuilder().
		SetName(CorsField).
		SetLabel("Javascript Origins").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("Origins").
				IsString())
}

func getAuthRedirectSchemaPropertyBuilder() provisioning.PropertyBuilder {
	return provisioning.NewSchemaPropertyBuilder().
		SetName(RedirectURLsField).
		SetLabel("Redirect URLs").
		IsArray().
		AddItem(
			provisioning.NewSchemaPropertyBuilder().
				SetName("URL").
				IsString())
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

	// Set up credential config from environment resource policies
	cfg.GetCredentialConfig().SetShouldDeprovisionExpired(env.Policies.Credentials.Expiry.Action == "deprovision")
	cfg.GetCredentialConfig().SetExpirationDays(int(env.Policies.Credentials.Expiry.Period))

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
	if agent.cfg != nil {
		agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
	}
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

// GetAuthProviderRegistry - Returns the auth provider registry
func GetAuthProviderRegistry() oauth.ProviderRegistry {
	if agent.authProviderRegistry == nil {
		agent.authProviderRegistry = oauth.NewProviderRegistry()
	}
	return agent.authProviderRegistry
}

func registerSubscriptionWebhook(at config.AgentType, client apic.Client) error {
	if at == config.DiscoveryAgent {
		return client.RegisterSubscriptionWebhook()
	}
	return nil
}

func startTeamACLCache() {
	// Only discovery agents need to start the ACL handler
	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		registerAccessControlListHandler()
	}

	registerTeamMapCacheJob()
}

func isRunningInDockerContainer() bool {
	// Within the cgroup file, if you are not in a docker container all entries are like these devices:/
	// If in a docker container, entries are like this: devices:/docker/xxxxxxxxx.
	// So, all we need to do is see if ":/docker" exists somewhere in the file.
	bytes, err := os.ReadFile("/proc/1/cgroup")
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
	agentMutex.Lock()
	defer agentMutex.Unlock()
	return agent.cfg
}

// setCentralConfig - Sets the central config
func setCentralConfig(cfg config.CentralConfig) {
	agentMutex.Lock()
	defer agentMutex.Unlock()
	agent.cfg = cfg
}

// GetAPICache - Returns the cache
func GetAPICache() cache.Cache {
	if agent.cacheManager == nil {
		agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, agent.agentFeaturesCfg.PersistCacheEnabled())
	}
	return agent.cacheManager.GetAPIServiceCache()
}

// GetCacheManager - Returns the cache
func GetCacheManager() agentcache.Manager {
	if agent.cacheManager == nil {
		agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, agent.agentFeaturesCfg.PersistCacheEnabled())
	}
	return agent.cacheManager
}

func GetResourceManager() resource.Manager {
	if agent.agentResourceManager == nil {
		return nil
	}
	return agent.agentResourceManager
}

// GetAgentResource - Returns Agent resource
func GetAgentResource() *apiV1.ResourceInstance {
	if agent.agentResourceManager == nil {
		return nil
	}
	return agent.agentResourceManager.GetAgentResource()
}

// GetAgentResourceManager - Returns Agent resource
func GetAgentResourceManager() resource.Manager {
	return agent.agentResourceManager
}

// AddUpdateAgentDetails - Adds a new or Updates an existing key on the agent details sub resource
func AddUpdateAgentDetails(key, value string) {
	if agent.agentResourceManager != nil {
		agent.agentResourceManager.AddUpdateAgentDetails(key, value)
	}
}

// GetDetailFromAgentResource - gets the value of an agent detail from the resource
func GetDetailFromAgentResource(key string) string {
	if agent.agentResourceManager == nil {
		return ""
	}
	val, _ := util.GetAgentDetailsValue(agent.agentResourceManager.GetAgentResource(), key)
	return val
}

// UpdateStatus - Updates the agent state
func UpdateStatus(status, description string) {
	UpdateStatusWithPrevious(status, status, description)
}

// UpdateStatusWithPrevious - Updates the agent state providing a previous state
func UpdateStatusWithPrevious(status, prevStatus, description string) {
	ctx := context.WithValue(context.Background(), ctxLogger, logger)
	UpdateStatusWithContext(ctx, status, prevStatus, description)
}

// UpdateStatusWithContext - Updates the agent state providing a context
func UpdateStatusWithContext(ctx context.Context, status, prevStatus, description string) {
	logger := ctx.Value(ctxLogger).(log.FieldLogger)
	if agent.agentResourceManager != nil {
		err := agent.agentResourceManager.UpdateAgentStatus(status, prevStatus, description)
		if err != nil {
			logger.WithError(err).Warnf("could not update the agent status reference")
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
		logger.Info("Stopping agent")
		if agent.profileDone != nil {
			<-agent.profileDone
		}
		cleanUp()
		agent.cacheManager.SaveCache()
		os.Exit(0)
	}()
}

func setupProfileSignalProcessor(cpuProfile, memProfile string) {
	if agent.agentFeaturesCfg.ProcessSystemSignalsEnabled() {
		// create channel for base signal processor
		agent.profileDone = make(chan struct{})
	}

	// start the CPU profiling
	var cpuFile *os.File
	if cpuProfile != "" {
		var err error
		cpuFile, err = os.Create(cpuProfile)
		if err != nil {
			fmt.Printf("Error creating cpu profiling file: %v", err)
		}
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			fmt.Printf("Error running the cpu profiling: %v", err)
		}
	}

	// Listen for a system signal to stop the agent
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigs

		// stop cpu profiling if it was started
		if cpuProfile != "" {
			pprof.StopCPUProfile()
			cpuFile.Close()
		}

		// run the memory profiling
		if memProfile != "" {
			memFile, err := os.Create(memProfile)
			if err != nil {
				fmt.Printf("Error creating memory profiling file: %v", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(memFile); err != nil {
				fmt.Printf("Error running the memory profiling: %v", err)
			}
			memFile.Close() // error handling omitted for example
		}

		if agent.agentFeaturesCfg.ProcessSystemSignalsEnabled() {
			// signal the base signal processor now that the profiling output is complete
			agent.profileDone <- struct{}{}
		}
	}()
}

// cleanUp - AgentCleanup
func cleanUp() {
	UpdateStatusWithPrevious(AgentStopped, AgentRunning, "")
}

func newHandlers() []handler.Handler {
	handlers := []handler.Handler{
		handler.NewAPISvcHandler(agent.cacheManager),
		handler.NewInstanceHandler(agent.cacheManager),
		handler.NewAgentResourceHandler(agent.agentResourceManager),
		handler.NewWatchResourceHandler(agent.cacheManager, agent.cfg),
		agent.proxyResourceHandler,
	}

	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		handlers = append(
			handlers,
			handler.NewCategoryHandler(agent.cacheManager),
			handler.NewCRDHandler(agent.cacheManager),
			handler.NewARDHandler(agent.cacheManager),
			handler.NewEnvironmentHandler(agent.cacheManager, agent.cfg.GetCredentialConfig()),
		)
	}

	// Register managed application and access handler for traceability agent
	// For discovery agent, the handlers gets registered while setting up provisioner
	if agent.cfg.GetAgentType() == config.TraceabilityAgent {
		handlers = append(
			handlers,
			handler.NewTraceAccessRequestHandler(agent.cacheManager, agent.apicClient),
			handler.NewTraceManagedApplicationHandler(agent.cacheManager),
		)
	}

	return handlers
}
