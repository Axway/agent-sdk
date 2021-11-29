package cmd

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd/agentsync"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/cmd/properties/resolver"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/errors"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	log "github.com/Axway/agent-sdk/pkg/util/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

// Constants for cmd flags
const (
	pathConfigFlag        = "pathConfig"
	beatsPathConfigFlag   = "path.config"
	EnvFileFlag           = "envFile"
	EnvFileFlagDesciption = "Path of the file with environment variables to override configuration"
)

// CommandHandler - Root command execution handler
type CommandHandler func() error

// InitConfigHandler - Handler to be invoked on config initialization
type InitConfigHandler func(centralConfig config.CentralConfig) (interface{}, error)

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
	agentType         config.AgentType
	props             properties.Properties
	statusCfg         config.StatusConfig
	centralCfg        config.CentralConfig
	agentCfg          interface{}
	secretResolver    resolver.SecretResolver
	initialized       bool
}

func init() {
	config.AgentTypeName = BuildAgentName
	config.AgentVersion = BuildVersion + "-" + BuildCommitSha
	config.AgentDataPlaneType = BuildDataPlaneType
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

// NewRootCmd - Creates a new Agent Root Command
func NewRootCmd(exeName, desc string, initConfigHandler InitConfigHandler, commandHandler CommandHandler, agentType config.AgentType) AgentRootCmd {
	c := &agentRootCommand{
		agentName:         exeName,
		commandHandler:    commandHandler,
		initConfigHandler: initConfigHandler,
		agentType:         agentType,
		secretResolver:    resolver.NewSecretResolver(),
		initialized:       false,
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
	c.addBaseProps()
	config.AddLogConfigProperties(c.props, fmt.Sprintf("%s.log", exeName))
	agentsync.AddSyncConfigProperties(c.props)
	config.AddCentralConfigProperties(c.props, agentType)
	config.AddStatusConfigProperties(c.props)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)

	// Call the config add props
	return c
}

// NewCmd - Creates a new Agent Root Command using existing cmd
func NewCmd(rootCmd *cobra.Command, exeName, desc string, initConfigHandler InitConfigHandler, commandHandler CommandHandler, agentType config.AgentType) AgentRootCmd {
	c := &agentRootCommand{
		agentName:         exeName,
		commandHandler:    commandHandler,
		initConfigHandler: initConfigHandler,
		agentType:         agentType,
		secretResolver:    resolver.NewSecretResolver(),
		initialized:       false,
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
	if agentType == config.TraceabilityAgent {
		properties.SetAliasKeyPrefix(c.agentName)
	}

	c.addBaseProps()
	config.AddLogConfigProperties(c.props, fmt.Sprintf("%s.log", exeName))
	agentsync.AddSyncConfigProperties(c.props)
	config.AddCentralConfigProperties(c.props, agentType)
	config.AddStatusConfigProperties(c.props)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)

	// Call the config add props
	return c
}

// Add the command line properties for the logger and path config
func (c *agentRootCommand) addBaseProps() {
	c.props.AddStringPersistentFlag(pathConfigFlag, ".", "Path to the directory containing the YAML configuration file for the agent")
	c.props.AddStringPersistentFlag(EnvFileFlag, "", EnvFileFlagDesciption)
}

func (c *agentRootCommand) initialize(cmd *cobra.Command, args []string) error {
	_, envFile := c.props.StringFlagValue(EnvFileFlag)
	err := util.LoadEnvFromFile(envFile)
	if err != nil {
		return errors.Wrap(config.ErrEnvConfigOverride, err.Error())
	}

	_, agentConfigFilePath := c.props.StringFlagValue(pathConfigFlag)
	_, beatsConfigFilePath := c.props.StringFlagValue(beatsPathConfigFlag)

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
		if err != nil {
			fmt.Println("Error in getting status : " + err.Error())
			os.Exit(1)
		}
		fmt.Println(statusOut)
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

// initConfig - Initializes the central config and invokes initConfig handler
// to initialize the agent config. Performs validation on returned agent config
func (c *agentRootCommand) initConfig() error {
	// Clean the secret map on config change
	c.secretResolver.ResetResolver()

	_, err := config.ParseAndSetupLogConfig(c.GetProperties())
	if err != nil {
		return err
	}

	c.statusCfg, err = config.ParseStatusConfig(c.GetProperties())
	err = c.statusCfg.ValidateCfg()
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

	err = agent.Initialize(c.centralCfg)
	if err != nil {
		return err
	}

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
		// Start the initial and recurring version check jobs
		startVersionCheckJobs(c.centralCfg)
		// Init the healthcheck API
		hc.HandleRequests()
	}
	c.initialized = true
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
			if c.agentType == config.TraceabilityAgent {
				properties.SetAliasKeyPrefix(c.agentName)
				log.SetIsLogP()
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
