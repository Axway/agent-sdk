package cmd

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cmd/agentsync"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/cmd/properties/resolver"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

// Constants for cmd flags
const (
	pathConfigFlag         = "pathConfig"
	beatsPathConfigFlag    = "path.config"
	EnvFileFlag            = "envFile"
	EnvFileFlagDescription = "Path of the file with environment variables to override configuration"
	cpuprofile             = "cpuprofile"
	memprofile             = "memprofile"
	httpprofile            = "httpprofile"
)

type NewCommandOption func(*agentRootCommand)

// CommandHandler - Root command execution handler
type CommandHandler func() error

// InitConfigHandler - Handler to be invoked on config initialization
type InitConfigHandler func(centralConfig config.CentralConfig) (interface{}, error)
type FinalizeAgentInitHandler func() error

// AgentRootCmd - Root Command for the Agents
type AgentRootCmd interface {
	RootCmd() *cobra.Command
	Execute() error

	// Get the agentType
	GetAgentType() config.AgentType
	AddCommand(*cobra.Command)

	GetProperties() properties.Properties
}

// agentRootCommand - Represents the agent root command
type agentRootCommand struct {
	agentName         string
	rootCmd           *cobra.Command
	commandHandler    CommandHandler
	initConfigHandler InitConfigHandler
	finalizeAgentInit FinalizeAgentInitHandler
	agentType         config.AgentType
	props             properties.Properties
	statusCfg         config.StatusConfig
	agentFeaturesCfg  config.AgentFeaturesConfig
	centralCfg        config.CentralConfig
	agentCfg          interface{}
	secretResolver    resolver.SecretResolver
	initialized       bool
	memprofile        string
	cpuprofile        string
	httpprofile       bool
}

func init() {
	config.AgentTypeName = BuildAgentName
	config.AgentVersion = BuildVersion + "-" + BuildCommitSha
	config.AgentDataPlaneType = apic.Unidentified.String()
	if BuildDataPlaneType != "" {
		config.AgentDataPlaneType = BuildDataPlaneType
	}

	config.SDKVersion = SDKBuildVersion
	// initalize the global Source used by rand.Intn() and other functions of the rand package using rand.Seed().
	rand.Seed(time.Now().UnixNano())
}

func buildCmdVersion(desc string) string {
	return fmt.Sprintf("- %s", buildAgentInfo(desc))
}

func buildAgentInfo(desc string) string {
	return fmt.Sprintf("%s version %s-%s, Amplify Agents SDK version %s", desc, BuildVersion, BuildCommitSha, SDKBuildVersion)
}

func WithFinalizeAgentInitFunc(f FinalizeAgentInitHandler) NewCommandOption {
	return func(c *agentRootCommand) {
		c.finalizeAgentInit = f
	}
}

// NewRootCmd - Creates a new Agent Root Command
func NewRootCmd(exeName, desc string, initConfigHandler InitConfigHandler, commandHandler CommandHandler, agentType config.AgentType, opts ...NewCommandOption) AgentRootCmd {
	c := &agentRootCommand{
		agentName:         exeName,
		commandHandler:    commandHandler,
		initConfigHandler: initConfigHandler,
		agentType:         agentType,
		secretResolver:    resolver.NewSecretResolver(),
		initialized:       false,
	}

	for _, o := range opts {
		o(c)
	}

	// use the description from the build if available
	if BuildAgentDescription != "" {
		desc = BuildAgentDescription
	}

	c.rootCmd = &cobra.Command{
		Use:     c.agentName,
		Short:   desc,
		Version: buildCmdVersion(desc),
		RunE:    c.run,
		PreRunE: c.initialize,
	}

	c.props = properties.NewPropertiesWithSecretResolver(c.rootCmd, c.secretResolver)
	c.addBaseProps(agentType)
	config.AddLogConfigProperties(c.props, fmt.Sprintf("%s.log", exeName))
	config.AddMetricLogConfigProperties(c.props, agentType)
	config.AddUsageConfigProperties(c.props, agentType)
	agentsync.AddSyncConfigProperties(c.props)
	config.AddCentralConfigProperties(c.props, agentType)
	config.AddStatusConfigProperties(c.props)
	config.AddAgentFeaturesConfigProperties(c.props)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)

	// Call the config add props
	return c
}

// NewCmd - Creates a new Agent Root Command using existing cmd
func NewCmd(rootCmd *cobra.Command, exeName, desc string, initConfigHandler InitConfigHandler, commandHandler CommandHandler, agentType config.AgentType, opts ...NewCommandOption) AgentRootCmd {
	c := &agentRootCommand{
		agentName:         exeName,
		commandHandler:    commandHandler,
		initConfigHandler: initConfigHandler,
		agentType:         agentType,
		secretResolver:    resolver.NewSecretResolver(),
		initialized:       false,
	}

	for _, o := range opts {
		o(c)
	}

	// use the description from the build if available
	if BuildAgentDescription != "" {
		desc = BuildAgentDescription
	}

	c.rootCmd = rootCmd
	c.rootCmd.Use = c.agentName
	c.rootCmd.Short = desc
	c.rootCmd.Version = buildCmdVersion(desc)
	c.rootCmd.RunE = c.run
	c.rootCmd.PreRunE = c.initialize

	c.props = properties.NewPropertiesWithSecretResolver(c.rootCmd, c.secretResolver)
	if agentType == config.TraceabilityAgent || agentType == config.ComplianceAgent {
		properties.SetAliasKeyPrefix(c.agentName)
	}

	c.addBaseProps(agentType)
	config.AddLogConfigProperties(c.props, fmt.Sprintf("%s.log", exeName))
	agentsync.AddSyncConfigProperties(c.props)
	config.AddCentralConfigProperties(c.props, agentType)
	config.AddStatusConfigProperties(c.props)
	config.AddAgentFeaturesConfigProperties(c.props)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)

	removeBeatSubCommands(c.rootCmd)
	// Call the config add props
	return c
}

func removeBeatSubCommands(rootCmd *cobra.Command) {
	removeBeatSubCommand(rootCmd, "export")
	removeBeatSubCommand(rootCmd, "keystore")
	removeBeatSubCommand(rootCmd, "run")
	removeBeatSubCommand(rootCmd, "setup")
	removeBeatSubCommand(rootCmd, "test")
	removeBeatSubCommand(rootCmd, "version")
}

func removeBeatSubCommand(rootCmd *cobra.Command, subCmdName string) {
	subCmd, _, err := rootCmd.Find([]string{subCmdName})
	if err == nil {
		rootCmd.RemoveCommand(subCmd)
	}
}

// Add the command line properties for the logger and path config
func (c *agentRootCommand) addBaseProps(agentType config.AgentType) {
	c.props.AddStringPersistentFlag(pathConfigFlag, ".", "Path to the directory containing the YAML configuration file for the agent")
	c.props.AddStringPersistentFlag(EnvFileFlag, "", EnvFileFlagDescription)
	if agentType == config.DiscoveryAgent {
		c.props.AddStringProperty(cpuprofile, "", "write cpu profile to `file`")
		c.props.AddStringProperty(memprofile, "", "write memory profile to `file`")
		c.props.AddBoolProperty(httpprofile, false, "set to setup the http profiling endpoints")
	}
}

func (c *agentRootCommand) initialize(cmd *cobra.Command, args []string) error {
	_, envFile := c.props.StringFlagValue(EnvFileFlag)
	err := util.LoadEnvFromFile(envFile)
	if err != nil {
		return errors.Wrap(config.ErrEnvConfigOverride, err.Error())
	}

	_, agentConfigFilePath := c.props.StringFlagValue(pathConfigFlag)
	_, beatsConfigFilePath := c.props.StringFlagValue(beatsPathConfigFlag)
	if c.agentType == config.DiscoveryAgent {
		_, c.cpuprofile = c.props.StringFlagValue(cpuprofile)
		_, c.memprofile = c.props.StringFlagValue(memprofile)
		c.httpprofile = c.props.BoolFlagValue(httpprofile)
	}

	// If the Agent pathConfig value is set and the beats path.config is not then use the pathConfig value for both
	if beatsConfigFilePath == "" && agentConfigFilePath != "" {
		c.props.SetStringFlagValue(beatsPathConfigFlag, agentConfigFilePath)
		_, beatsConfigFilePath = c.props.StringFlagValue(beatsPathConfigFlag)
	}

	viper.SetConfigName(c.agentName)
	// viper.SetConfigType("yaml")  //Comment out since yaml, yml is a support extension already.  We need an updated story to take into account the other supported extensions

	// Add both the agent pathConfig and beats path.config paths to the config path array
	viper.AddConfigPath(agentConfigFilePath)
	viper.AddConfigPath(beatsConfigFilePath)
	viper.AddConfigPath(".")
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err = viper.ReadInConfig()
	if err != nil {
		if envFile == "" {
			return err
		} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Debugf("Config file changed : %s", e.Name)
		c.onConfigChange()
	})

	c.checkStatusFlag()
	agentsync.SetSyncMode(c.GetProperties())
	return nil
}

func (c *agentRootCommand) checkStatusFlag() {
	statusPort := c.props.IntPropertyValue("status.port")
	if c.props.BoolFlagValue("status") {
		urlObj := url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", statusPort),
			Path:   "status",
		}
		statusOut, err := hc.GetHealthcheckOutput(urlObj.String())
		if statusOut != "" {
			fmt.Println(statusOut)
		}

		if err != nil {
			fmt.Println("Error in getting status : " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func (c *agentRootCommand) onConfigChange() {
	c.initConfig()
	agentConfigChangeHandler := agent.GetConfigChangeHandler()
	if agentConfigChangeHandler != nil {
		agentConfigChangeHandler()
	}
}

func (c *agentRootCommand) postCentralConfigProcessing(_ config.CentralConfig, agentFeaturesCfg config.AgentFeaturesConfig) error {
	err := config.ParseExternalIDPConfig(agentFeaturesCfg, c.GetProperties())
	if err != nil {
		return err
	}
	return nil
}

// initConfig - Initializes the central config and invokes initConfig handler
// to initialize the agent config. Performs validation on returned agent config
func (c *agentRootCommand) initConfig() error {
	// Clean the secret map on config change
	c.secretResolver.ResetResolver()

	_, err := config.ParseAndSetupLogConfig(c.GetProperties(), c.agentType)
	if err != nil {
		return err
	}

	c.statusCfg, _ = config.ParseStatusConfig(c.GetProperties())
	err = c.statusCfg.ValidateCfg()
	if err != nil {
		return err
	}

	// Init Agent Features Config
	c.agentFeaturesCfg, err = config.ParseAgentFeaturesConfig(c.GetProperties())
	if err != nil {
		return err
	}

	// Init Central Config
	c.centralCfg, err = config.ParseCentralConfig(c.GetProperties(), c.GetAgentType())
	if err != nil {
		return err
	}

	// must set the hc config now, because the healthchecker loop starts in agent.Initialize
	hc.SetStatusConfig(c.statusCfg)

	err = agent.InitializeWithAgentFeatures(c.centralCfg, c.agentFeaturesCfg, c.postCentralConfigProcessing)
	if err != nil {
		return err
	}
	agent.InitializeProfiling(c.cpuprofile, c.memprofile)

	jobs.UpdateDurations(c.statusCfg.GetHealthCheckInterval(), c.centralCfg.GetJobExecutionTimeout())

	// Initialize Agent Config
	c.agentCfg, err = c.initConfigHandler(c.centralCfg)
	if err != nil {
		return err
	}

	if c.agentCfg != nil {
		err := agent.ApplyResourceToConfig(c.agentCfg)
		if err != nil {
			return err
		}

		// Validate Agent Config
		err = config.ValidateConfig(c.agentCfg)
		if err != nil {
			return err
		}
	}

	if !c.initialized {
		err = c.finishInit()
		if err != nil {
			return err
		}
	}
	c.initialized = true
	return nil
}

func (c *agentRootCommand) finishInit() error {
	agent.SetFinalizeAgentFunc(c.finalizeAgentInit)

	err := agent.CacheInitSync()
	if err != nil {
		return err
	}

	// Start the initial and recurring version check jobs
	startVersionCheckJobs(c.centralCfg, c.agentFeaturesCfg)

	if util.IsNotTest() {
		healthCheckServer := hc.NewServer(c.httpprofile)
		healthCheckServer.HandleRequests()
	}

	return nil
}

// run - Executes the agent command
func (c *agentRootCommand) run(cmd *cobra.Command, args []string) (err error) {
	err = c.initConfig()
	statusText := ""
	if err == nil {
		// Register resource change handler to re-initialize config on resource change
		// This should trigger config init and applyresourcechange handlers
		agent.OnAgentResourceChange(c.onConfigChange)

		// Check the sync flag
		exitcode := agentsync.CheckSyncFlag()
		if exitcode > -1 {
			os.Exit(exitcode)
		}

		log.Infof("Starting %s", buildAgentInfo(c.rootCmd.Short))
		if c.commandHandler != nil {
			// Setup logp to use beats logger.
			// Setting up late here as log entries for agent/command initialization are not logged
			// as the beats logger is initialized only when the beat instance is created.
			if c.agentType == config.TraceabilityAgent || c.agentType == config.ComplianceAgent {
				properties.SetAliasKeyPrefix(c.agentName)
				log.SetIsLogP()
			}

			c.healthCheckTicker()

			if util.IsNotTest() && c.agentFeaturesCfg.AgentStatusUpdatesEnabled() && !c.centralCfg.GetUsageReportingConfig().IsOfflineMode() {
				agent.StartAgentStatusUpdate()
			}

			err = c.commandHandler()
			if err != nil {
				log.Error(err.Error())
				statusText = err.Error()
			}
		}
	} else {
		statusText = err.Error()
	}
	status := agent.AgentStopped
	if statusText != "" {
		status = agent.AgentFailed
	}
	agent.UpdateStatusWithPrevious(status, agent.AgentRunning, statusText)
	return
}

// Run health check ticker for every 5 seconds
// If after 5 minutes, the health checker still returns HC status !OK, exit the agent.  Otherwise, return true and continue processing
func (c *agentRootCommand) healthCheckTicker() {
	if !util.IsNotTest() {
		log.Trace("Skipping health check ticker in test mode")
		return
	}
	log.Trace("run health checker ticker to check health status on RunChecks")
	ticker := time.NewTicker(5 * time.Second)
	tickerTimeout := time.NewTicker(5 * time.Minute)

	defer ticker.Stop()
	defer tickerTimeout.Stop()

	for {
		select {
		case <-tickerTimeout.C:
			log.Error("healthcheck run checks failing. Stopping agent - Check docs.axway.com for more info on the reported error code")
			agent.UpdateStatus(agent.AgentFailed, "healthchecks on startup failed")
			os.Exit(0)
		case <-ticker.C:
			status := hc.RunChecks()
			if status == hc.OK {
				log.Trace("healthcheck on startup is OK. Continue processing")
				return
			} else {
				log.Warn("healthchecks on startup are still processing")
			}
		}
	}
}

func (c *agentRootCommand) RootCmd() *cobra.Command {
	return c.rootCmd
}

func (c *agentRootCommand) Execute() error {
	return c.rootCmd.Execute()
}

func (c *agentRootCommand) GetAgentType() config.AgentType {
	return c.agentType
}

func (c *agentRootCommand) GetProperties() properties.Properties {
	return c.props
}

func (c *agentRootCommand) AddCommand(cmd *cobra.Command) {
	c.rootCmd.AddCommand(cmd)
}
