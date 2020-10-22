package properties

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Properties - Root Command Properties interface for all configs to use for adding and parsing values
type Properties interface {
	// Methods for adding yaml properties and command flag
	AddStringProperty(name string, defaultVal string, description string)
	AddStringPersistentFlag(name string, defaultVal string, description string)
	AddStringFlag(name string, description string)
	AddDurationProperty(name string, defaultVal time.Duration, description string)
	AddIntProperty(name string, defaultVal int, description string)
	AddBoolProperty(name string, defaultVal bool, description string)
	AddBoolFlag(name, description string)
	AddStringSliceProperty(name string, defaultVal []string, description string)

	// Methods to get the configured properties
	StringPropertyValue(name string) string
	StringFlagValue(name string) (bool, string)
	DurationPropertyValue(name string) time.Duration
	IntPropertyValue(name string) int
	BoolPropertyValue(name string) bool
	BoolFlagValue(name string) bool
	StringSlicePropertyValue(name string) []string

	// Log Properties
	MaskValues(name string)
	DebugLogProperties()
}

type properties struct {
	Properties
	rootCmd             *cobra.Command
	flattenedProperties map[string]string
}

// NewProperties - Creates a new Properties struct
func NewProperties(rootCmd *cobra.Command) Properties {
	cmdprops := &properties{
		rootCmd:             rootCmd,
		flattenedProperties: make(map[string]string),
	}

	return cmdprops
}

func (p *properties) bindOrPanic(key string, flg *flag.Flag) {
	if err := viper.BindPFlag(key, flg); err != nil {
		panic(err)
	}
}

func (p *properties) AddStringProperty(name string, defaultVal string, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().String(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddStringPersistentFlag(flagName string, defaultVal string, description string) {
	if p.rootCmd != nil {
		p.rootCmd.PersistentFlags().String(flagName, "", description)
	}
}

func (p *properties) AddStringFlag(flagName string, description string) {
	if p.rootCmd != nil {
		p.rootCmd.Flags().String(flagName, "", description)
	}
}

func (p *properties) AddStringSliceProperty(name string, defaultVal []string, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().StringSlice(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddDurationProperty(name string, defaultVal time.Duration, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().Duration(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddIntProperty(name string, defaultVal int, description string) {
	if p.rootCmd != nil {
		flagName := p.nameToFlagName(name)
		p.rootCmd.Flags().Int(flagName, defaultVal, description)
		p.bindOrPanic(name, p.rootCmd.Flags().Lookup(flagName))
	}
}

func (p *properties) AddBoolProperty(name string, defaultVal bool, description string) {
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
		p.addPropertyToFlatMap(name, val.(string))
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
	s := viper.GetString(name)
	p.addPropertyToFlatMap(name, s)
	return s
}

func (p *properties) StringFlagValue(name string) (bool, string) {
	flag := p.rootCmd.Flag(name)
	if flag == nil || flag.Value.String() == "" {
		return false, ""
	}
	fv := flag.Value.String()
	p.addPropertyToFlatMap(name, fv)
	return true, fv
}

func (p *properties) DurationPropertyValue(name string) time.Duration {
	d := viper.GetDuration(name)
	p.addPropertyToFlatMap(name, d.String())
	return d
}

func (p *properties) IntPropertyValue(name string) int {
	i := viper.GetInt(name)
	p.addPropertyToFlatMap(name, strconv.Itoa(i))
	return i
}

func (p *properties) BoolPropertyValue(name string) bool {
	b := viper.GetBool(name)
	p.addPropertyToFlatMap(name, strconv.FormatBool(b))
	return b
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

// String array containing any sensitive data that needs to be masked with "*" (asterisks)
// Add any sensitive data here using flattened key format
var maskValues = make([]string, 0)

func (p *properties) addPropertyToFlatMap(key, value string) {
	for _, maskValue := range maskValues {
		match, _ := regexp.MatchString("\\b"+strings.TrimSpace(maskValue)+"\\b", key)
		if match {
			value = util.MaskValue(value)
		}
	}
	p.flattenedProperties[key] = value
}

func (p *properties) MaskValues(maskedKeys string) {
	maskValues = strings.Split(maskedKeys, ",")
}

func (p *properties) DebugLogProperties() {
	data, _ := json.MarshalIndent(p.flattenedProperties, "", " ")
	log.Debugf("%s\n", data)
}
