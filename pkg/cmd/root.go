package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/agent"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util"
	"github.com/spf13/cobra"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/agentsync"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
	hc "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/healthcheck"
	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/spf13/viper"
)

// Constants for cmd flags
const (
	EnvFileFlag           = "envFile"
	EnvFileFlagDesciption = "Path of the file with environment variables to override configuration"
)

// CommandHandler - Root command execution handler
type CommandHandler func() error

// InitConfigHandler - Handler to be invoked on config initialization
type InitConfigHandler func(centralConfig corecfg.CentralConfig) (interface{}, error)

// AgentRootCmd - Root Command for the Agents
type AgentRootCmd interface {
	RootCmd() *cobra.Command
	Execute() error

	// Get the agentType
	GetAgentType() corecfg.AgentType
	AddCommand(*cobra.Command)

	GetProperties() properties.Properties
}

// agentRootCommand - Represents the agent root command
type agentRootCommand struct {
	agentName         string
	rootCmd           *cobra.Command
	commandHandler    CommandHandler
	initConfigHandler InitConfigHandler
	agentType         corecfg.AgentType
	props             properties.Properties
}

func init() {
	corecfg.AgentTypeName = BuildAgentName
	corecfg.AgentVersion = BuildVersion + "-" + BuildCommitSha
}

// NewRootCmd - Creates a new Agent Root Command
func NewRootCmd(exeName, desc string, initConfigHandler InitConfigHandler, commandHandler CommandHandler, agentType corecfg.AgentType) AgentRootCmd {
	c := &agentRootCommand{
		agentName:         exeName,
		commandHandler:    commandHandler,
		initConfigHandler: initConfigHandler,
		agentType:         agentType,
	}

	c.rootCmd = &cobra.Command{
		Use:     c.agentName,
		Short:   desc,
		Version: fmt.Sprintf("%s-%s", BuildVersion, BuildCommitSha),
		RunE:    c.run,
		PreRunE: c.initialize,
	}
	c.props = properties.NewProperties(c.rootCmd)
	c.addBaseProps()
	agentsync.AddSyncConfigProperties(c.props)
	corecfg.AddCentralConfigProperties(c.props, agentType)
	corecfg.AddStatusConfigProperties(c.props)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)

	// Call the config add props
	return c
}

// Add the command line properties for the logger and path config
func (c *agentRootCommand) addBaseProps() {
	c.props.AddStringProperty("log.level", "info", "Log level (debug, info, warn, error)")
	c.props.AddStringProperty("log.format", "json", "Log format (json, line, package)")
	c.props.AddStringProperty("log.output", "stdout", "Log output type (stdout, file, both)")
	c.props.AddStringProperty("log.path", "logs", "Log file path if output type is file or both")
	c.props.AddStringProperty("log.maskedValues", "", "List of key words in the config to be masked (e.g. pwd, password, secret, key")
	c.props.AddStringPersistentFlag("pathConfig", ".", "Configuration file path for the agent")
	c.props.AddStringProperty(EnvFileFlag, "", EnvFileFlagDesciption)
}

func (c *agentRootCommand) initialize(cmd *cobra.Command, args []string) error {
	envFile := c.props.StringPropertyValue(EnvFileFlag)
	err := util.LoadEnvFromFile(envFile)
	if err != nil {
		return errors.Wrap(config.ErrEnvConfigOverride, err.Error())
	}

	_, configFilePath := c.props.StringFlagValue("pathConfig")
	viper.SetConfigName(c.agentName)
	// viper.SetConfigType("yaml")  //Comment out since yaml, yml is a support extension already.  We need an updated story to take into account the other supported extensions
	viper.AddConfigPath(configFilePath)
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
		statusOut, _ := hc.GetHealthcheckOutput(urlObj.String())
		fmt.Println(statusOut)
		os.Exit(0)
	}
}

// initConfig - Initializes the central config and invokes initConfig handler
// to initialize the agent config. Performs validation on returned agent config
func (c *agentRootCommand) initConfig() error {
	c.setupLogger()

	// Init the healthcheck API
	statusCfg, err := corecfg.ParseStatusConfig(c.GetProperties())
	err = statusCfg.ValidateConfig()
	if err != nil {
		return err
	}

	hc.SetStatusConfig(statusCfg)
	hc.HandleRequests()

	// Init Central Config
	centralCfg, err := corecfg.ParseCentralConfig(c.GetProperties(), c.GetAgentType())
	if err != nil {
		return err
	}

	err = agent.Initialize(centralCfg)
	if err != nil {
		return err
	}

	// Initialize Agent Config
	agentCfg, err := c.initConfigHandler(centralCfg)
	if err != nil {
		return err
	}

	err = agent.ApplyResouceToConfig(agentCfg)
	if err != nil {
		return err
	}
	c.GetProperties().DebugLogProperties()

	// Validate Agent Config
	err = config.ValidateConfig(agentCfg)
	if err != nil {
		return err
	}

	// Check the sync flag
	exitcode := agentsync.CheckSyncFlag()
	if exitcode > -1 {
		os.Exit(exitcode)
	}

	return err
}

// parse the logger config values and setup the logger
func (c *agentRootCommand) setupLogger() {
	logLevel := c.props.StringPropertyValue("log.level")
	logFormat := c.props.StringPropertyValue("log.format")
	logOutput := c.props.StringPropertyValue("log.output")
	logPath := c.props.StringPropertyValue("log.path")
	maskedValues := c.props.StringPropertyValue("log.maskedValues")
	// Only attempt to mask values if the key maskValues AND key words for maskValues exist
	if maskedValues != "" {
		c.props.MaskValues(maskedValues)
	}

	log.SetupLogging(c.agentName, logLevel, logFormat, logOutput, logPath)
}

// run - Executes the agent command
func (c *agentRootCommand) run(cmd *cobra.Command, args []string) (err error) {
	err = c.initConfig()
	statusText := ""
	if err == nil {
		log.Infof("Starting %s (%s)", c.rootCmd.Short, c.rootCmd.Version)
		if c.commandHandler != nil {
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
	agent.UpdateStatus(status, statusText)
	return
}

func (c *agentRootCommand) RootCmd() *cobra.Command {
	return c.rootCmd
}

func (c *agentRootCommand) Execute() error {
	return c.rootCmd.Execute()
}

func (c *agentRootCommand) GetAgentType() corecfg.AgentType {
	return c.agentType
}

func (c *agentRootCommand) GetProperties() properties.Properties {
	return c.props
}

func (c *agentRootCommand) AddCommand(cmd *cobra.Command) {
	c.rootCmd.AddCommand(cmd)
}
