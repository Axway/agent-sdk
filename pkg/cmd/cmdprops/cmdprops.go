package cmdprops

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// CmdProps - Root Command for the Agents
type CmdProps interface {
	AddBaseConfigProperties()

	// Methods for adding yaml properties and command flag
	AddStringProperty(name, flagName string, defaultVal string, description string)
	AddDurationProperty(name, flagName string, defaultVal time.Duration, description string)
	AddIntProperty(name, flagName string, defaultVal int, description string)
	AddBoolProperty(name, flagName string, defaultVal bool, description string)
	AddStringSliceProperty(name, flagName string, defaultVal []string, description string)
	AddBoolFlag(flagName string, description string)

	// Methods to get the configured properties
	StringPropertyValue(name string) string
	DurationPropertyValue(name string) time.Duration
	IntPropertyValue(name string) int
	BoolPropertyValue(name string) bool
	StringSlicePropertyValue(name string) []string
	BoolFlagValue(name string) bool
}

// cmdProperties - Represents the agent root command
type cmdProperties struct {
	CmdProps
	rootCmd *cobra.Command
}

// NewCmdProperties - Creates a new Agent Root Command
func NewCmdProperties(rootCmd *cobra.Command) CmdProps {
	c := &cmdProperties{
		rootCmd: rootCmd,
	}

	return c
}

func (c *cmdProperties) AddBaseConfigProperties() {
	// Log yaml properties and command flags
	c.AddStringProperty("log.level", "logLevel", "info", "Log level (debug, info, warn, error)")
	c.AddStringProperty("log.format", "logFormat", "json", "Log format (json, line, package)")
	c.AddStringProperty("log.output", "logOutput", "stdout", "Log output type (stdout, file, both)")
	c.AddStringProperty("log.path", "logPath", "logs", "Log file path if output type is file or both")
	c.AddStringProperty("path.config", "pathConfig", ".", "Configuration file path for the agent")
}

func (c *cmdProperties) bindOrPanic(key string, flg *flag.Flag) {
	if err := viper.BindPFlag(key, flg); err != nil {
		panic(err)
	}
}

func (c *cmdProperties) AddStringProperty(name, flagName string, defaultVal string, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().String(flagName, defaultVal, description)
		c.bindOrPanic(name, c.rootCmd.Flags().Lookup(flagName))
	}
}

func (c *cmdProperties) AddStringSliceProperty(name, flagName string, defaultVal []string, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().StringSlice(flagName, defaultVal, description)
		c.bindOrPanic(name, c.rootCmd.Flags().Lookup(flagName))
	}
}

func (c *cmdProperties) AddDurationProperty(name, flagName string, defaultVal time.Duration, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().Duration(flagName, defaultVal, description)
		c.bindOrPanic(name, c.rootCmd.Flags().Lookup(flagName))
	}
}

func (c *cmdProperties) AddIntProperty(name, flagName string, defaultVal int, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().Int(flagName, defaultVal, description)
		c.bindOrPanic(name, c.rootCmd.Flags().Lookup(flagName))
	}
}

func (c *cmdProperties) AddBoolProperty(name, flagName string, defaultVal bool, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().Bool(flagName, defaultVal, description)
		c.bindOrPanic(name, c.rootCmd.Flags().Lookup(flagName))
	}
}

func (c *cmdProperties) AddBoolFlag(flagName string, description string) {
	if c.rootCmd != nil {
		c.rootCmd.Flags().Bool(flagName, false, description)
	}
}

func (c *cmdProperties) StringSlicePropertyValue(name string) []string {
	val := viper.Get(name)

	// special check to differentiate between yaml and commandline parsing. For commandline, must
	// turn it into an array ourselves
	switch val.(type) {
	case string:
		return c.convertStringToSlice(fmt.Sprintf("%v", viper.Get(name)))
	default:
		return viper.GetStringSlice(name)
	}
}

func (c *cmdProperties) convertStringToSlice(value string) []string {
	slc := strings.Split(value, ",")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	return slc
}

func (c *cmdProperties) StringPropertyValue(name string) string {
	return viper.GetString(name)
}

func (c *cmdProperties) DurationPropertyValue(name string) time.Duration {
	return viper.GetDuration(name)
}

func (c *cmdProperties) IntPropertyValue(name string) int {
	return viper.GetInt(name)
}

func (c *cmdProperties) BoolPropertyValue(name string) bool {
	return viper.GetBool(name)
}

func (c *cmdProperties) BoolFlagValue(name string) bool {
	flag := c.rootCmd.Flag(name)
	if flag == nil {
		return false
	}
	if flag.Value.String() == "true" {
		return true
	}
	return false
}
