package properties

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Properties - Root Command Properties interface for all configs to use for adding and parsing values
type Properties interface {
	// Methods for adding yaml properties and command flag
	AddStringProperty(name, flagName string, defaultVal string, description string)
	AddDurationProperty(name, flagName string, defaultVal time.Duration, description string)
	AddIntProperty(name, flagName string, defaultVal int, description string)
	AddBoolProperty(name, flagName string, defaultVal bool, description string)
	AddBoolFlag(flagName, description string)
	AddStringSliceProperty(name, flagName string, defaultVal []string, description string)
	AddBaseConfigProperties()

	// Methods to get the configured properties
	StringPropertyValue(name string) string
	DurationPropertyValue(name string) time.Duration
	IntPropertyValue(name string) int
	BoolPropertyValue(name string) bool
	BoolFlagValue(name string) bool
	StringSlicePropertyValue(name string) []string
}

type properties struct {
	Properties
	rootCmd *cobra.Command
}

// NewProperties - Creates a new Properties struct
func NewProperties(rootCmd *cobra.Command) Properties {
	cmdprops := &properties{
		rootCmd: rootCmd,
	}

	return cmdprops
}

// AddBaseConfigProperties -
func (p *properties) AddBaseConfigProperties() {
	// Log yaml properties and command flags
	p.AddStringProperty("log.level", "logLevel", "info", "Log level (debug, info, warn, error)")
	p.AddStringProperty("log.format", "logFormat", "json", "Log format (json, line, package)")
	p.AddStringProperty("log.output", "logOutput", "stdout", "Log output type (stdout, file, both)")
	p.AddStringProperty("log.path", "logPath", "logs", "Log file path if output type is file or both")
	p.AddStringProperty("path.config", "pathConfig", ".", "Configuration file path for the agent")
}

func (p *properties) bindOrPanic(key string, flg *flag.Flag) {
	if err := viper.BindPFlag(key, flg); err != nil {
		panic(err)
	}
}

func (p *properties) AddStringProperty(name, flagName string, defaultVal string, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().String(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddStringSliceProperty(name, flagName string, defaultVal []string, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().StringSlice(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddDurationProperty(name, flagName string, defaultVal time.Duration, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().Duration(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddIntProperty(name, flagName string, defaultVal int, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().Int(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddBoolProperty(name, flagName string, defaultVal bool, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().Bool(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddBoolFlag(flagName string, description string) {
	if p.rootCmd != nil {
		p.rootCmd.Flags().Bool(flagName, false, description)
	}
}

func (p *properties) StringSlicePropertyValue(name string) []string {
	val := viper.Get(name)

	// special check to differentiate between yaml and commandline parsing. For commandline, must
	// turn it into an array ourselves
	switch val.(type) {
	case string:
		return p.convertStringToSlice(fmt.Sprintf("%v", viper.Get(name)))
	default:
		return viper.GetStringSlice(name)
	}
}

func (p *properties) convertStringToSlice(value string) []string {
	slc := strings.Split(value, ",")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	return slc
}

func (p *properties) StringPropertyValue(name string) string {
	return viper.GetString(name)
}

func (p *properties) DurationPropertyValue(name string) time.Duration {
	return viper.GetDuration(name)
}

func (p *properties) IntPropertyValue(name string) int {
	return viper.GetInt(name)
}

func (p *properties) BoolPropertyValue(name string) bool {
	return viper.GetBool(name)
}

func (p *properties) BoolFlagValue(name string) bool {
	flag := p.rootCmd.Flag(name)
	if flag == nil {
		return false
	}
	if flag.Value.String() == "true" {
		return true
	}
	return false
}

func (p *properties) nameToFlagName(name string) (flagName string) {
	parts := strings.Split(name, ".")
	flagName = parts[0]
	for _, part := range parts[1:] {
		flagName += strings.Title(part)
	}
	return
}
