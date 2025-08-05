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
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/customunit"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
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

// APIValidator - Callback for validating the API
type APIValidator func(apiID, stageName string) bool

// ConfigChangeHandler - Callback for Config change event
type ConfigChangeHandler func()

// ShutdownHandler - function that the agent may implement to be called when a shutdown request is received
type ShutdownHandler func()
type agentData struct {
	agentResourceManager resource.Manager
	apicClient           apic.Client
	cfg                  config.CentralConfig
	agentFeaturesCfg     config.AgentFeaturesConfig
	tokenRequester       auth.PlatformTokenGetter

	teamMap                    cache.Cache
	cacheManager               agentcache.Manager
	apiValidator               APIValidator
	apiValidatorLock           sync.Mutex
	apiValidatorJobID          string
	configChangeHandler        ConfigChangeHandler
	agentResourceChangeHandler ConfigChangeHandler
	customUnitHandler          *customunit.CustomUnitHandler
	agentShutdownHandler       ShutdownHandler
	proxyResourceHandler       *handler.StreamWatchProxyHandler
	isInitialized              bool
	isFinalized                bool

	streamer             *stream.StreamerClient
	authProviderRegistry oauth.ProviderRegistry

	finalizeAgentInit func() error
	publishingLock    *sync.Mutex
	ardLock           sync.Mutex

	status                       string
	statusConfig                 *config.StatusConfiguration
	healthcheckManager           *hc.Manager
	hcmMutex                     sync.RWMutex
	applicationProfileDefinition string

	entitlements map[string]interface{}

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
	agent.statusConfig = config.NewStatusConfig()
}

// Initialize - Initializes the agent
func Initialize(centralCfg config.CentralConfig) error {
	return InitializeWithAgentFeatures(centralCfg, config.NewAgentFeaturesConfiguration(), nil)
}

type PostCentralConfigProc func(config.CentralConfig, config.AgentFeaturesConfig) error

// InitializeWithAgentFeatures - Initializes the agent with agent features
func InitializeWithAgentFeatures(centralCfg config.CentralConfig, agentFeaturesCfg config.AgentFeaturesConfig, postCfgProcessor PostCentralConfigProc) error {
	if agent.teamMap == nil {
		agent.teamMap = cache.New()
	}

	setupHealthcheckManager()
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

	// check to confirm usagereporting.offline and agentfeatures.persistcache are not both set to true
	if agentFeaturesCfg.PersistCacheEnabled() && centralCfg.GetUsageReportingConfig().IsOfflineMode() {
		agentFeaturesCfg.SetPersistentCache(false)
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
		centralCfg.GetURL(),
		centralCfg.GetPlatformURL(),
		centralCfg.GetAuthConfig().GetTokenURL(),
		centralCfg.GetUsageReportingConfig().GetURL(),
	}
	if centralCfg.GetTraceabilityProtocol() == "https" {
		// add the traceability host to the single entry filter for https only
		singleEntryFilter = append(singleEntryFilter, fmt.Sprintf("https://%s", centralCfg.GetTraceabilityHost()))
	}
	api.SetConfigAgent(
		GetUserAgent(),
		centralCfg.GetSingleURL(),
		singleEntryFilter)

	if agentFeaturesCfg.ConnectionToCentralEnabled() {
		err = handleCentralConfig(centralCfg)
		if err != nil {
			return err
		}

		if postCfgProcessor != nil {
			err = postCfgProcessor(centralCfg, agentFeaturesCfg)
			if err != nil {
				return err
			}
		}
	}

	// call the metric services.
	metricServicesConfigs := agentFeaturesCfg.GetMetricServicesConfigs()
	if agent.cfg.GetAgentType() != config.ComplianceAgent {
		agent.customUnitHandler = customunit.NewCustomUnitHandler(metricServicesConfigs, agent.cacheManager, centralCfg.GetAgentType())
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

func setupHealthcheckManager() {
	agent.hcmMutex.Lock()
	defer agent.hcmMutex.Unlock()

	hcOpts := []hc.Option{
		hc.SetAsGlobalHealthCheckManager(),
		hc.WithPort(agent.statusConfig.GetPort()),
		hc.WithInterval(agent.statusConfig.GetHealthCheckInterval()),
		hc.WithPeriod(agent.statusConfig.GetHealthCheckPeriod()),
		hc.WithName(agent.statusConfig.Name),
		hc.WithVersion(agent.statusConfig.Version),
	}

	if !util.IsNotTest() {
		hcOpts = append(hcOpts, hc.IsUnitTest())
	}
	if agent.statusConfig.HTTPProfile {
		hcOpts = append(hcOpts, hc.WithPprof())
	}
	agent.healthcheckManager = hc.NewManager(hcOpts...)
}

func SetStatusConfig(statusConfig *config.StatusConfiguration) {
	agent.statusConfig = statusConfig
}

func GetHealthcheckManager() *hc.Manager {
	agent.hcmMutex.Lock()
	defer agent.hcmMutex.Unlock()
	return agent.healthcheckManager
}

func RegisterHealthcheck(name, endpoint string, check hc.CheckStatus) (string, error) {
	return agent.healthcheckManager.RegisterHealthcheck(name, endpoint, check)
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

	// update the managed envs from the agent resource config
	if centralCfg.GetAgentType() == config.ComplianceAgent {
		resource.MergeComplianceAgentWithConfig(agent.agentResourceManager.GetAgentResource(), centralCfg)
	}

	// do not get entitlements in test
	if !util.IsNotTest() {
		return nil
	}
	return getEntitlements()
}

func getEntitlements() error {
	// pull the entitlements for the org
	entitlements, err := agent.apicClient.GetEntitlements()
	if err != nil {
		return err
	}
	agent.entitlements = entitlements

	logger.WithField("entitlements", agent.entitlements).Trace("retrieved entitlements")
	return nil
}

func handleInitialization() error {
	setupSignalProcessor()

	if util.IsNotTest() && agent.agentFeaturesCfg.ConnectionToCentralEnabled() {
		// if credentials can expire and need to be deprovisioned then start the credential checker

		registerCredentialChecker()

		startTeamACLCache()
	}

	return nil
}

// InitializeProfiling - setup the CPU and Memory profiling if options given
func InitializeProfiling(cpuProfile, memProfile string) {
	if memProfile != "" || cpuProfile != "" {
		setupProfileSignalProcessor(cpuProfile, memProfile)
	}
}

func SetFinalizeAgentFunc(f func() error) {
	agent.finalizeAgentInit = f
}

func finalizeInitialization() error {
	if agent.isFinalized {
		return nil
	}

	err := registerExternalIDPs()
	if err != nil {
		// if an error happened registering IdPs we should kill the agent to avoid
		//   updating instances with wrong credential request def types
		logger.WithError(err).Fatal("failed to register CRDs for external IdP config")
	}

	if agent.finalizeAgentInit != nil {
		err := agent.finalizeAgentInit()
		if err != nil {
			return err
		}
	}

	agent.healthcheckManager.StartServer()
	agent.isFinalized = true
	return nil
}

func registerExternalIDPs() error {
	if !util.IsNotTest() || !agent.agentFeaturesCfg.ConnectionToCentralEnabled() || agent.cfg.GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		idPCfg := agent.agentFeaturesCfg.GetExternalIDPConfig()
		if idPCfg == nil {
			return nil
		}

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

func CacheInitSync() error {
	if !util.IsNotTest() || !agent.agentFeaturesCfg.ConnectionToCentralEnabled() || agent.cfg.GetUsageReportingConfig().IsOfflineMode() {
		return nil
	}

	eventSync, err := newEventSync()
	if err != nil {
		return errors.Wrap(errors.ErrInitServicesNotReady, err.Error())
	}

	if err := eventSync.SyncCache(); err != nil {
		return errors.Wrap(errors.ErrInitServicesNotReady, err.Error())
	}
	// set the rebuild function in the agent resource manager
	agent.agentResourceManager.SetRebuildCacheFunc(eventSync)
	return nil
}

func registerCredentialProvider(idp config.IDPConfig, tlsCfg config.TLSConfig, proxyURL string, clientTimeout time.Duration) error {
	logger := logger.WithField("title", idp.GetIDPTitle()).WithField("name", idp.GetIDPName()).WithField("type", idp.GetIDPType()).WithField("metadata-url", idp.GetMetadataURL())
	err := GetAuthProviderRegistry().RegisterProvider(idp, tlsCfg, proxyURL, clientTimeout)
	if err != nil {
		logger.WithError(err).Errorf("unable to register external IdP provider, any credential request to the IdP will not be processed.")
		return err
	}
	crdName := idp.GetIDPName() + "-" + provisioning.OAuthIDPCRD
	provider, err := GetAuthProviderRegistry().GetProviderByName(idp.GetIDPName())
	if err != nil {
		return err
	}
	crd, err := NewOAuthCredentialRequestBuilder(
		WithCRDType(provisioning.CrdTypeOauth),
		WithCRDName(crdName),
		WithCRDForIDP(provider, provider.GetSupportedScopes()),
		WithCRDOAuthSecret(),
		WithCRDRequestSchemaProperty(getCorsSchemaPropertyBuilder()),
		WithCRDRequestSchemaProperty(getAuthRedirectSchemaPropertyBuilder()),
		WithCRDIsSuspendable(),
	).Register()
	if err != nil {
		logger.
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
	if !agent.isInitialized && !isRunningInDockerContainer() {
		return agent.healthcheckManager.CheckIsRunning()
	}
	return nil
}

type TestOpt func(*testOpts)

type testOpts struct {
	marketplace bool
	agentType   config.AgentType
}

// InitializeForTest - Initialize for test
func InitializeForTest(apicClient apic.Client, opts ...TestOpt) {
	agent.apicClient = apicClient
	tOpts := &testOpts{}
	for _, o := range opts {
		o(tOpts)
	}
	if agent.cfg == nil {
		agent.cfg = config.NewTestCentralConfig(tOpts.agentType)
	}
	agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
	agent.agentFeaturesCfg = &config.AgentFeaturesConfiguration{
		ConnectToCentral:     true,
		ProcessSystemSignals: true,
		VersionChecker:       true,
		PersistCache:         true,
		AgentStatusUpdates:   true,
	}
}

func TestWithMarketplace() func(*testOpts) {
	return func(o *testOpts) {
		o.marketplace = true
	}
}

func TestWithCentralConfig(cfg config.CentralConfig) func(*testOpts) {
	return func(o *testOpts) {
		agent.cfg = cfg
	}
}

func TestWithAgentType(agentType config.AgentType) func(*testOpts) {
	return func(o *testOpts) {
		o.agentType = agentType
	}
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

func GetCustomUnitHandler() *customunit.CustomUnitHandler {
	return agent.customUnitHandler
}

// RegisterShutdownHandler - Registers shutdown handler
func RegisterShutdownHandler(handler ShutdownHandler) {
	agent.agentShutdownHandler = handler
}

func startTeamACLCache() {
	// Only discovery agents need to start the ACL handler
	if agent.cfg.GetAgentType() == config.DiscoveryAgent {
		registerAccessControlListHandler()
	}
	handler.RefreshTeamCache(agent.apicClient, agent.cacheManager)
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

func GetUserAgent() string {
	envName := ""
	agentName := ""
	isGRPC := false
	if agent.cfg != nil {
		envName = agent.cfg.GetEnvironmentName()
		agentName = agent.cfg.GetAgentName()
		isGRPC = agent.cfg.IsUsingGRPC()
	}
	return util.NewUserAgent(
		config.AgentTypeName,
		config.AgentVersion,
		config.SDKVersion,
		envName,
		agentName,
		isGRPC).FormatUserAgent()
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

// GetStatus - get the last reported status
func GetStatus() string {
	return agent.status
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
	agent.status = status
	logger := ctx.Value(ctxLogger).(log.FieldLogger)
	if agent.cfg != nil && agent.cfg.IsUsingGRPC() {
		updateStatusOverStream(status, prevStatus, description)
		return
	}

	if agent.agentResourceManager != nil {
		err := agent.agentResourceManager.UpdateAgentStatus(status, prevStatus, description)
		if err != nil {
			logger.WithError(err).Warn("could not update the agent status reference")
		}
	} else {
		logger.WithField("status", agent.status).Trace("skipping status update, agent resource manager is not initialized")
	}
}

func updateStatusOverStream(status, prevStatus, description string) {
	if agent.streamer != nil {
		err := agent.streamer.UpdateAgentStatus(status, prevStatus, description)
		if err != nil {
			logger.WithError(err).Warn("could not update the agent status reference")
		}
	} else {
		logger.WithField("status", agent.status).Trace("skipping status update, stream client is not initialized")
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

		// call the agent shutdown handler
		if agent.agentShutdownHandler != nil {
			agent.agentShutdownHandler()
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
	// stopped status updated with gRPC watch
	if !agent.cfg.IsUsingGRPC() {
		UpdateStatusWithPrevious(AgentStopped, AgentRunning, "")
	}
}

func newHandlers() []handler.Handler {
	envName := GetCentralConfig().GetEnvironmentName()
	handlers := []handler.Handler{
		handler.NewAPISvcHandler(agent.cacheManager, envName),
		handler.NewInstanceHandler(agent.cacheManager, envName),
		handler.NewAgentResourceHandler(agent.agentResourceManager, sampling.GetGlobalSampling(), agent.cacheManager, agent.apicClient),
		agent.proxyResourceHandler,
	}

	switch agent.cfg.GetAgentType() {
	case config.DiscoveryAgent:
		handlers = append(
			handlers,
			handler.NewWatchResourceHandler(agent.cacheManager, handler.WithWatchTopicFeatures(agent.cfg)),
			handler.NewCRDHandler(agent.cacheManager),
			handler.NewARDHandler(agent.cacheManager),
			handler.NewAPDHandler(agent.cacheManager),
			handler.NewEnvironmentHandler(agent.cacheManager, agent.cfg.GetCredentialConfig(), envName),
		)
	case config.TraceabilityAgent:
		// Register managed application and access handler for traceability agent
		// For discovery agent, the handlers gets registered while setting up provisioner
		handlers = append(
			handlers,
			handler.NewWatchResourceHandler(agent.cacheManager, handler.WithWatchTopicFeatures(agent.cfg)),
			handler.NewTraceAccessRequestHandler(agent.cacheManager, agent.apicClient),
			handler.NewTraceManagedApplicationHandler(agent.cacheManager),
		)
	case config.ComplianceAgent:
		handlers = append(
			handlers,
			handler.NewWatchResourceHandler(agent.cacheManager,
				handler.WithWatchTopicFeatures(agent.cfg),
				handler.WithWatchTopicGroupKind(
					[]apiV1.GroupKind{
						management.EnvironmentGVK().GroupKind,
						management.APIServiceInstanceGVK().GroupKind,
						management.ComplianceRuntimeResultGVK().GroupKind,
					},
				),
			),
			handler.NewCRRHandler(agent.cacheManager),
		)
	}

	return handlers
}
