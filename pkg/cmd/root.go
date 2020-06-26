package cmd

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/cmdprops"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	hc "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/healthcheck"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
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
	GetCmdProps() cmdprops.CmdProps
}

// agentRootCommand - Represents the agent root command
type agentRootCommand struct {
	agentName         string
	rootCmd           *cobra.Command
	commandHandler    CommandHandler
	initConfigHandler InitConfigHandler
	agentType         corecfg.AgentType
	cmdProps          cmdprops.CmdProps
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
		PreRun:  c.initialize,
	}

	c.cmdProps = cmdprops.NewCmdProperties(c.rootCmd)

	hc.SetNameAndVersion(exeName, c.rootCmd.Version)
	c.GetCmdProps().AddBaseConfigProperties()
	corecfg.AddCentralConfigProperties(c.GetCmdProps(), c.GetAgentType())
	corecfg.AddStatusConfigProperties(c.GetCmdProps())
	corecfg.AddSubscriptionsConfigProperties(c.GetCmdProps())

	return c
}

func (c *agentRootCommand) initialize(cmd *cobra.Command, args []string) {
	configFilePath := c.GetCmdProps().StringPropertyValue("path.config")
	viper.SetConfigName(c.agentName)
	// viper.SetConfigType("yaml")  //Comment out since yaml, yml is a support extension already.  We need an updated story to take into account the other supported extensions
	viper.AddConfigPath(configFilePath)
	viper.AddConfigPath(".")
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err.Error())
	}
	c.checkStatusFlag()
}

func (c *agentRootCommand) checkStatusFlag() {
	statusPort := c.GetCmdProps().IntPropertyValue("status.port")
	if c.GetCmdProps().BoolFlagValue("status") {
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

func (c *agentRootCommand) bindOrPanic(key string, flg *flag.Flag) {
	if err := viper.BindPFlag(key, flg); err != nil {
		panic(err)
	}
}

// initConfig - Initializes the central config and invokes initConfig handler
// to initialize the agent config. Performs validation on returned agent config
func (c *agentRootCommand) initConfig() error {
	logLevel := c.GetCmdProps().StringPropertyValue("log.level")
	logFormat := c.GetCmdProps().StringPropertyValue("log.format")
	logOutput := c.GetCmdProps().StringPropertyValue("log.output")
	logPath := c.GetCmdProps().StringPropertyValue("log.path")
	log.SetupLogging(c.agentName, logLevel, logFormat, logOutput, logPath)

	// Init Status Config
	statusCfg, err := corecfg.ParseStatusConfig(c.GetCmdProps())
	if err != nil {
		return err
	}

	// Start Health Checker
	hc.SetStatusConfig(statusCfg)
	hc.HandleRequests()

	// Init Central Config
	centralCfg, err := corecfg.ParseCentralConfig(c.GetCmdProps(), c.GetAgentType())
	if err != nil {
		return err
	}

	// Init Subscription Config
	_, err = corecfg.ParseSubscriptionConfig(c.GetCmdProps())
	if err != nil {
		return err
	}

	// Initialize Agent Config
	agentCfg, err := c.initConfigHandler(centralCfg)
	if err != nil {
		return err
	}

	// Validate Agent Config
	return c.validateAgentConfig(agentCfg)
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

func (c *agentRootCommand) GetCmdProps() cmdprops.CmdProps {
	return c.cmdProps
}
