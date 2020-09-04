package cmd

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"

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
	hc.SetStatusConfig(statusCfg)
	hc.HandleRequests()

	// Init Central Config
	centralCfg, err := corecfg.ParseCentralConfig(c.GetProperties(), c.GetAgentType())
	if err != nil {
		return err
	}

	// Initialize Agent Config
	agentCfg, err := c.initConfigHandler(centralCfg)
	if err != nil {
		return err
	}

	// Validate Agent Config
	err = c.validateAgentConfig(agentCfg)
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
	log.SetupLogging(c.agentName, logLevel, logFormat, logOutput, logPath)
}

// validateAgentConfig - Validates the agent config
// Uses reflection to get the Validate method on the config struct or
// struct variable.
// Makes call to Validate method except if the struct variable is of CentralConfig type
// as the validation for CentralConfig is already done during parseCentralConfig
func (c *agentRootCommand) validateAgentConfig(agentCfg interface{}) error {
	// Check if top level struct has Validate. If it does then call Validate
	// only at top level
	if objInterface, ok := agentCfg.(interface{ Validate() error }); ok {
		return objInterface.Validate()
	}

	// If the parameter is of struct pointer, use indirection to get the
	// real value object
	v := reflect.ValueOf(agentCfg)
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}

	// Look for Validate method on struct properties and invoke it
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() {
			fieldInterface := v.Field(i).Interface()
			// Skip the property it is CentralConfig type as its already Validated
			// during parseCentralConfig
			if _, ok := fieldInterface.(corecfg.CentralConfig); !ok {
				if objInterface, ok := fieldInterface.(interface{ Validate() error }); ok {
					err := objInterface.Validate()
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// run - Executes the agent command
func (c *agentRootCommand) run(cmd *cobra.Command, args []string) (err error) {
	err = c.initConfig()

	if err == nil {
		log.Infof("Starting %s (%s)", c.rootCmd.Short, c.rootCmd.Version)
		err = c.commandHandler()
		if err != nil {
			log.Error(err.Error())
		}
	}

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
