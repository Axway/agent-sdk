package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

func getPFlag(cmd AgentRootCmd, flagName string) *flag.Flag {
	return cmd.RootCmd().Flags().Lookup(flagName)
}

func assertCmdFlag(t *testing.T, cmd AgentRootCmd, flagName, fType, description string) {
	pflag := getPFlag(cmd, flagName)
	assert.NotNil(t, &pflag)
	assert.Equal(t, fType, pflag.Value.Type())
	assert.Equal(t, description, pflag.Usage)
}

func assertStringCmdFlag(t *testing.T, cmd AgentRootCmd, propertyName, flagName, defaultVal, description string) {
	assertCmdFlag(t, cmd, flagName, "string", description)
	assert.Equal(t, defaultVal, viper.GetString(propertyName))
}

func assertStringSliceCmdFlag(t *testing.T, cmd AgentRootCmd, propertyName, flagName string, defaultVal []string, description string) {
	assertCmdFlag(t, cmd, flagName, "stringSlice", description)
	assert.Equal(t, defaultVal, viper.GetStringSlice(propertyName))
}

func assertBooleanCmdFlag(t *testing.T, cmd AgentRootCmd, propertyName, flagName string, defaultVal bool, description string) {
	assertCmdFlag(t, cmd, flagName, "bool", description)
	assert.Equal(t, defaultVal, viper.GetBool(propertyName))
}

func assertDurationCmdFlag(t *testing.T, cmd AgentRootCmd, propertyName, flagName string, defaultVal time.Duration, description string) {
	assertCmdFlag(t, cmd, flagName, "duration", description)
	assert.Equal(t, defaultVal, viper.GetDuration(propertyName))
}

type agentConfig struct {
	bProp                 bool
	dProp                 time.Duration
	iProp                 int
	sProp                 string
	sPropExt              string
	ssProp                []string
	osProp                []map[string]interface{}
	agentValidationCalled bool
}

func (a *agentConfig) ValidateCfg() error {
	a.agentValidationCalled = true
	if a.sProp == "" {
		return errors.New("agentConfig: String prop not set")
	}
	return nil
}

type configWithValidation struct {
	configValidationCalled bool
	CentralCfg             corecfg.CentralConfig
	AgentCfg               *agentConfig
}

func (c *configWithValidation) ValidateCfg() error {
	c.configValidationCalled = true
	if c.AgentCfg.sProp == "" {
		return errors.New("configWithValidation: String prop not set")
	}
	return nil
}

type configWithNoValidation struct {
	configValidationCalled bool
	CentralCfg             corecfg.CentralConfig
	AgentCfg               corecfg.IConfigValidator
}

func TestRootCmdFlags(t *testing.T) {
	// Discovery Agent
	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	assertStringCmdFlag(t, rootCmd, "central.url", "centralUrl", "", "URL of Amplify Central")              // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.platformURL", "centralPlatformURL", "", "URL of the platform") // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.singleURL", "centralSingleURL", "", "Alternate Connection for Agent if using static IP")
	assertStringCmdFlag(t, rootCmd, "central.organizationID", "centralOrganizationID", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.team", "centralTeam", "", "Team name for creating catalog")
	assertStringCmdFlag(t, rootCmd, "central.environment", "centralEnvironment", "", "The Environment that the APIs will be associated with in Amplify Central")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "centralAuthPrivateKey", "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "centralAuthPublicKey", "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.word", "centralAuthKeyPassword", "", "Path to the password file required by the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "centralAuthUrl", "", "Amplify Central authentication URL") // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.auth.realm", "centralAuthRealm", "Broker", "Amplify Central authentication Realm")
	assertStringCmdFlag(t, rootCmd, "central.auth.clientId", "centralAuthClientId", "", "Client ID for the service account")
	assertDurationCmdFlag(t, rootCmd, "central.auth.timeout", "centralAuthTimeout", 10*time.Second, "Timeout waiting for AxwayID response")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.nextProtos", "centralSslNextProtos", []string{}, "List of supported application level protocols, comma separated")
	assertBooleanCmdFlag(t, rootCmd, "central.ssl.insecureSkipVerify", "centralSslInsecureSkipVerify", false, "Controls whether a client verifies the server's certificate chain and host name")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.cipherSuites", "centralSslCipherSuites", corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	assertStringCmdFlag(t, rootCmd, "central.ssl.minVersion", "centralSslMinVersion", corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	assertStringCmdFlag(t, rootCmd, "central.ssl.maxVersion", "centralSslMaxVersion", "0", "Maximum acceptable SSL/TLS protocol version")
	assertBooleanCmdFlag(t, rootCmd, "central.migration.cleanInstances", "centralMigrationCleanInstances", false, "Set this to clean all but latest instance, per stage, within an API Service")

	// Traceability Agent
	rootCmd = NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	assertStringCmdFlag(t, rootCmd, "central.deployment", "centralDeployment", "", "Amplify Central")       // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.url", "centralUrl", "", "URL of Amplify Central")              // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.platformURL", "centralPlatformURL", "", "URL of the platform") // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.singleURL", "centralSingleURL", "", "Alternate Connection for Agent if using static IP")
	assertStringCmdFlag(t, rootCmd, "central.organizationID", "centralOrganizationID", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "centralAuthPrivateKey", "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "centralAuthPublicKey", "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.keyPassword", "centralAuthKeyPassword", "", "Path to the password file required by the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "centralAuthUrl", "", "Amplify Central authentication URL") // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, "central.auth.realm", "centralAuthRealm", "Broker", "Amplify Central authentication Realm")
	assertStringCmdFlag(t, rootCmd, "central.auth.clientId", "centralAuthClientId", "", "Client ID for the service account")
	assertDurationCmdFlag(t, rootCmd, "central.auth.timeout", "centralAuthTimeout", 10*time.Second, "Timeout waiting for AxwayID response")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.nextProtos", "centralSslNextProtos", []string{}, "List of supported application level protocols, comma separated")
	assertBooleanCmdFlag(t, rootCmd, "central.ssl.insecureSkipVerify", "centralSslInsecureSkipVerify", false, "Controls whether a client verifies the server's certificate chain and host name")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.cipherSuites", "centralSslCipherSuites", corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	assertStringCmdFlag(t, rootCmd, "central.ssl.minVersion", "centralSslMinVersion", corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	assertStringCmdFlag(t, rootCmd, "central.ssl.maxVersion", "centralSslMaxVersion", "0", "Maximum acceptable SSL/TLS protocol version")

	// Log yaml properties and command flags
	assertStringCmdFlag(t, rootCmd, "log.level", "logLevel", "info", "Log level (trace, debug, info, warn, error)")
	assertStringCmdFlag(t, rootCmd, "log.format", "logFormat", "json", "Log format (json, line)")
	assertStringCmdFlag(t, rootCmd, "log.output", "logOutput", "stdout", "Log output type (stdout, file, both)")
	assertStringCmdFlag(t, rootCmd, "log.file.path", "logFilePath", "logs", "Log file path if output type is file or both")
}

func TestNewCmd(t *testing.T) {
	rootCmd := &cobra.Command{}
	newCmd := NewCmd(
		rootCmd,
		"test",
		"discovery agent",
		func(centralConfig corecfg.CentralConfig) (interface{}, error) {
			return nil, nil
		},
		func() error {
			return nil
		},
		corecfg.DiscoveryAgent,
	)

	assert.NotNil(t, newCmd)

}

func TestRootCmdConfigFileLoad(t *testing.T) {

	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)

	err := rootCmd.Execute()

	// should be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.True(t, ok, "Incorrect error returned: %s", err.Error())
	}

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", nil, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)

	assert.Contains(t, "Error central.organizationID not set in config", errBuf.String())
}

func TestRootCmdConfigDefault(t *testing.T) {
	discoveryInitConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		assert.Equal(t, "https://apicentral.axway.com", centralConfig.GetURL())
		assert.Equal(t, "222222", centralConfig.GetTeamName())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker", centralConfig.GetAuthConfig().GetAudience())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker/protocol/openid-connect/token", centralConfig.GetAuthConfig().GetTokenURL())
		assert.Equal(t, "cccc", centralConfig.GetAuthConfig().GetClientID())
		assert.Equal(t, "Broker", centralConfig.GetAuthConfig().GetRealm())
		assert.Equal(t, "/etc/private_key.pem", centralConfig.GetAuthConfig().GetPrivateKey())
		assert.Equal(t, "/etc/public_key", centralConfig.GetAuthConfig().GetPublicKey())
		assert.Equal(t, "", centralConfig.GetAuthConfig().GetKeyPassword())
		assert.Equal(t, 10*time.Second, centralConfig.GetAuthConfig().GetTimeout())
		return centralConfig, errors.New("Test return error from init config handler")
	}

	traceabilityInitConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		assert.Equal(t, "prod", centralConfig.GetAPICDeployment())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker", centralConfig.GetAuthConfig().GetAudience())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker/protocol/openid-connect/token", centralConfig.GetAuthConfig().GetTokenURL())
		assert.Equal(t, "cccc", centralConfig.GetAuthConfig().GetClientID())
		assert.Equal(t, "Broker", centralConfig.GetAuthConfig().GetRealm())
		assert.Equal(t, "/etc/private_key.pem", centralConfig.GetAuthConfig().GetPrivateKey())
		assert.Equal(t, "/etc/public_key", centralConfig.GetAuthConfig().GetPublicKey())
		assert.Equal(t, "", centralConfig.GetAuthConfig().GetKeyPassword())
		assert.Equal(t, 10*time.Second, centralConfig.GetAuthConfig().GetTimeout())
		return centralConfig, errors.New("Test return error from init config handler")
	}

	// Discovery
	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", discoveryInitConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")
	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "Test return error from init config handler, Discovery Agent", errBuf.String())

	// Traceability
	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", traceabilityInitConfigHandler, nil, corecfg.TraceabilityAgent)
	viper.AddConfigPath("./testdata")
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf = new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "Test return error from init config handler, Traceability Agent", errBuf.String())
}

func TestRootCmdAgentConfigValidation(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	var rootCmd AgentRootCmd
	var cfg *configWithValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.GetProperties().BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.GetProperties().IntPropertyValue("agent.int"),
				sProp:                 rootCmd.GetProperties().StringPropertyValue("agent.string"),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("CENTRAL_PLATFORMURL", s.URL)

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property", properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty("agent.int", 0, "Agent Int Property")
	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "configWithValidation: String prop not set", errBuf.String())
	assert.Equal(t, true, cfg.configValidationCalled)
	assert.Equal(t, false, cfg.AgentCfg.agentValidationCalled)
}

func TestRootCmdAgentConfigChildValidation(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	var rootCmd AgentRootCmd
	var cfg *configWithNoValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithNoValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.GetProperties().BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.GetProperties().IntPropertyValue("agent.int"),
				sProp:                 rootCmd.GetProperties().StringPropertyValue("agent.string"),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property", properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty("agent.int", 0, "Agent Int Property")
	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "agentConfig: String prop not set", errBuf.String())
	assert.Equal(t, false, cfg.configValidationCalled)
	assert.Equal(t, true, cfg.AgentCfg.(*agentConfig).agentValidationCalled)
}

func TestRootCmdHandlersWithError(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		return centralConfig, nil
	}
	cmdHandler := func() error {
		return nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("Test", "TestRootCmd", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	err := rootCmd.Execute()

	// should be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.True(t, ok, "Incorrect error returned: %s", err.Error())
	}

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}
}

func TestRootCmdHandlers(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	var rootCmd AgentRootCmd
	var cfg *configWithNoValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithNoValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.GetProperties().BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.GetProperties().IntPropertyValue("agent.int"),
				sProp:                 rootCmd.GetProperties().StringPropertyValue("agent.string"),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue("agent.stringSlice"),
				osProp:                rootCmd.GetProperties().ObjectSlicePropertyValue("agent.objectSlice"),
			},
		}
		return cfg, nil
	}
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("AGENT_OBJECTSLICE_INDEX_1", "1")
	os.Setenv("AGENT_OBJECTSLICE_NAME_1", "osp1_name")
	os.Setenv("AGENT_OBJECTSLICE_NAMEVALUE_1", "osp1_value")
	os.Setenv("AGENT_OBJECTSLICE_NAMETITLE_1", "osp1_title")
	os.Setenv("AGENT_OBJECTSLICE_INDEX_2", "2")
	os.Setenv("AGENT_OBJECTSLICE_NAMEVALUE_2", "osp2_value")
	os.Setenv("AGENT_OBJECTSLICE_NAMETITLE_2", "osp2_title")
	os.Setenv("AGENT_OBJECTSLICE_NAME_2", "osp2_name")
	os.Setenv("AGENT_OBJECTSLICE_INDEX_3", "3")
	os.Setenv("AGENT_OBJECTSLICE_NAMEVALUE_3", "osp3_value")
	os.Setenv("AGENT_OBJECTSLICE_NAME_3", "osp3_name")
	os.Setenv("AGENT_OBJECTSLICE_NAMETITLE_3", "osp3_title")

	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property", properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty("agent.int", 0, "Agent Int Property")
	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"index", "name", "namevalue", "nametitle"})
	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.Nil(t, err, "An unexpected error returned")
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Empty(t, "", errBuf.String())
	assert.Equal(t, false, cfg.configValidationCalled)
	agentCfg := cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, true, agentCfg.bProp)
	assert.Equal(t, 30*time.Second, agentCfg.dProp)
	assert.Equal(t, 555, agentCfg.iProp)
	assert.Equal(t, true, cmdHandlerInvoked)
	assert.Equal(t, []string{"ss1", "ss2"}, agentCfg.ssProp)
	if !assert.Len(t, agentCfg.osProp, 3, "the number of object slices expected was incorrect") {
		return
	}

	sort.Slice(agentCfg.osProp, func(i, j int) bool {
		return agentCfg.osProp[i]["index"].(string) < agentCfg.osProp[j]["index"].(string)
	})
	exp1 := map[string]interface{}{"index": "1", "name": "osp1_name", "namevalue": "osp1_value", "nametitle": "osp1_title"}
	assert.True(t, assert.ObjectsAreEqualValues(exp1, agentCfg.osProp[0]), fmt.Sprintf("the first object slice did not have correct values:\n expected %+v\n actual %+v", exp1, agentCfg.osProp[0]))
	exp2 := map[string]interface{}{"index": "2", "name": "osp2_name", "namevalue": "osp2_value", "nametitle": "osp2_title"}
	assert.True(t, assert.ObjectsAreEqualValues(exp2, agentCfg.osProp[1]), fmt.Sprintf("the second object slice did not have correct values:\n expected %+v\n actual %+v", exp2, agentCfg.osProp[1]))
	exp3 := map[string]interface{}{"index": "3", "name": "osp3_name", "namevalue": "osp3_value", "nametitle": "osp3_title"}
	assert.True(t, assert.ObjectsAreEqualValues(exp3, agentCfg.osProp[2]), fmt.Sprintf("the third object slice did not have correct values:\n expected %+v\n actual %+v", exp3, agentCfg.osProp[2]))
}

func TestRootCommandLoggerStdout(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.Nil(t, err, "An unexpected error was received")
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, "Incorrect error returned: %s", err.Error())
	}

	w.Close()

	var logData map[string]string
	scanner := bufio.NewScanner(r)

	level := "info"
	msg := "Starting test_with_non_defaults version -, Amplify Agents SDK version "

	for scanner.Scan() {
		out := scanner.Text()
		err := json.Unmarshal([]byte(out), &logData)
		assert.Nil(t, err, "failed to unmarshal log data")
		if logData["level"] == level && logData["message"] == msg {
			break
		}
	}

	os.Stdout = rescueStdout

	assert.Equal(t, level, logData["level"])
	assert.Equal(t, msg, logData["message"])
}

func TestRootCommandLoggerFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	s := newTestServer()
	defer s.Close()

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")
	rootCmd.RootCmd().SetArgs([]string{
		"--logOutput",
		"file",
		"--logFilePath",
		"./tmplogs",
		"--logFileName",
		"test_with_non_defaults.log",
	},
	)
	// Make sure to delete file
	os.RemoveAll("./tmplogs/test_with_non_defaults.log")

	fExecute := func() {
		rootCmd.Execute()
	}
	assert.NotPanics(t, fExecute)

	dat, err := ioutil.ReadFile("./tmplogs/test_with_non_defaults.log")
	assert.Nil(t, err, "failed to read file")
	scanner := bufio.NewScanner(bytes.NewReader(dat))

	var logData map[string]string
	level := "info"
	msg := "Starting test_with_non_defaults version -, Amplify Agents SDK version "

	for scanner.Scan() {
		out := scanner.Text()
		err := json.Unmarshal([]byte(out), &logData)
		assert.Nil(t, err, "failed to unmarshal log data")
		if logData["level"] == level && logData["message"] == msg {
			break
		}
	}

	assert.Equal(t, level, logData["level"])
	assert.Equal(t, msg, logData["message"])
}

func TestRootCommandLoggerStdoutAndFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	s := newTestServer()
	defer s.Close()

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")
	rootCmd.RootCmd().SetArgs([]string{
		"--logOutput",
		"both",
		"--logFilePath",
		"./tmplogs",
		"--logFileName",
		"test_with_non_defaults.log",
	},
	)
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fExecute := func() {
		rootCmd.Execute()
	}
	// Make sure to delete file
	os.Remove("./tmplogs/test_with_non_defaults.log")
	assert.NotPanics(t, fExecute)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	var logData map[string]string
	json.Unmarshal([]byte(out), &logData)

	dat, err := ioutil.ReadFile("./tmplogs/test_with_non_defaults.log")
	assert.Nil(t, err)
	assert.Equal(t, out, dat)
}

func TestRootCmdHandlerWithSecretRefProperties(t *testing.T) {
	secret := management.Secret{
		ResourceMeta: v1.ResourceMeta{Name: "agentSecret"},
		Spec: management.SecretSpec{
			Data: map[string]string{
				"secretKey":               "secretValue",
				"cachedSecretKey":         "cachedSecretValue",
				"keyElement1.keyElement2": "secretValue2",
			},
		},
	}

	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}

	environmentRes := &management.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/secrets/agentSecret") {
			buf, _ := json.Marshal(secret)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))
	defer s.Close()

	var rootCmd AgentRootCmd
	var cfg *configWithNoValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithNoValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				sProp:                 rootCmd.GetProperties().StringPropertyValue("agent.string"),
				sPropExt:              rootCmd.GetProperties().StringPropertyValue("agent.stringExt"),
				osProp:                rootCmd.GetProperties().ObjectSlicePropertyValue("agent.objectSlice"),
			},
		}
		return cfg, nil
	}
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}

	os.Setenv("CENTRAL_AUTH_URL", s.URL+"/auth")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("CENTRAL_ENVIRONMENT", "test")

	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"prop1", "prop2", "prop3"})

	// Case 1 : No secret resolution - use the value in config
	os.Setenv("AGENT_STRING", "testValue")
	os.Setenv("AGENT_STRINGEXT", "anotherTestValue")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", "osp1_1")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", "osp1_2")
	err := rootCmd.Execute()
	assert.Nil(t, err, "An unexpected error returned")
	agentCfg := cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "testValue", agentCfg.sProp)
	assert.Equal(t, "anotherTestValue", agentCfg.sPropExt)
	objectSliceProps := []string{agentCfg.osProp[0]["prop1"].(string), agentCfg.osProp[1]["prop1"].(string)}
	slices.Sort(objectSliceProps)
	assert.Equal(t, []string{"osp1_1", "osp1_2"}, objectSliceProps)
	assert.Equal(t, true, cmdHandlerInvoked)

	// Case 2 : Invalid secret resolution - secret ref with invalid secret name,
	// config value will be set to empty string
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"prop1", "prop2", "prop3"})

	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false
	os.Setenv("AGENT_STRING", "@Secret.invalidSecret.secretKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.invalidSecret.cachedSecretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", "@Secret.invalidSecret.secretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", "@Secret.invalidSecret.cachedSecretKey")

	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "agentConfig: String prop not set", err.Error())
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "", agentCfg.sProp)
	assert.Equal(t, "", agentCfg.sPropExt)
	assert.Equal(t, "", agentCfg.osProp[0]["prop1"])
	assert.Equal(t, "", agentCfg.osProp[1]["prop1"])
	assert.Equal(t, false, cmdHandlerInvoked)

	// Case 3 : Invalid secret resolution - secret ref with invalid key in secret
	// config value will be set to empty string
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"prop1", "prop2", "prop3"})

	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.invalidKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.invalidSecret.cachedSecretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", "@Secret.agentSecret.secretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", "@Secret.invalidSecret.cachedSecretKey")
	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "agentConfig: String prop not set", err.Error())
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "", agentCfg.sProp)
	assert.Equal(t, "", agentCfg.sPropExt)
	objectSliceProps = []string{agentCfg.osProp[0]["prop1"].(string), agentCfg.osProp[1]["prop1"].(string)}
	slices.Sort(objectSliceProps)
	assert.Equal(t, []string{"", "secretValue"}, objectSliceProps)
	assert.Equal(t, "", agentCfg.osProp[1]["prop1"])
	assert.Equal(t, false, cmdHandlerInvoked)

	// Case 4 : Successful secret resolution - use value in secret key
	// config value will be set to specified key in secret
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"prop1", "prop2", "prop3"})
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.secretKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.agentSecret.cachedSecretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", "@Secret.agentSecret.secretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", "@Secret.agentSecret.cachedSecretKey")
	err = rootCmd.Execute()
	assert.Nil(t, err)
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "secretValue", agentCfg.sProp)
	assert.Equal(t, "cachedSecretValue", agentCfg.sPropExt)
	objectSliceProps = []string{agentCfg.osProp[0]["prop1"].(string), agentCfg.osProp[1]["prop1"].(string)}
	slices.Sort(objectSliceProps)
	assert.Equal(t, []string{"cachedSecretValue", "secretValue"}, objectSliceProps)
	assert.Equal(t, true, cmdHandlerInvoked)

	// Case 5 : Successful secret resolution with key separate with dots(.) - use value in secret key
	// config value will be set to specified key in secret
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")
	rootCmd.GetProperties().AddObjectSliceProperty("agent.objectSlice", []string{"prop1", "prop2", "prop3"})

	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.keyElement1.keyElement2")
	os.Unsetenv("AGENT_OBJECTSLICE_PROP1_1")
	os.Unsetenv("AGENT_OBJECTSLICE_PROP1_2")
	os.Setenv("AGENT_OBJECTSLICE_PROP2_1", "@Secret.agentSecret.keyElement1.keyElement2")
	err = rootCmd.Execute()
	assert.Nil(t, err)
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "secretValue2", agentCfg.sProp)
	assert.Equal(t, true, cmdHandlerInvoked)
	assert.Equal(t, "secretValue2", agentCfg.osProp[0]["prop2"])
}

func noOpInitConfigHandler(centralConfig corecfg.CentralConfig) (interface{}, error) {
	return centralConfig, nil
}

func noOpCmdHandler() error {
	return nil
}

func newTestServer() *httptest.Server {
	teams := []definitions.PlatformTeam{
		{
			ID:      "123",
			Name:    "name",
			Default: true,
		},
	}

	environmentRes := &management.Environment{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{ID: "123"},
			Name:     "test",
			Title:    "test",
		},
	}

	secret := management.Secret{
		ResourceMeta: v1.ResourceMeta{Name: "agentSecret"},
		Spec: management.SecretSpec{
			Data: map[string]string{
				"secretKey":               "secretValue",
				"cachedSecretKey":         "cachedSecretValue",
				"keyElement1.keyElement2": "secretValue2",
			},
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/secrets/agentSecret") {
			buf, _ := json.Marshal(secret)
			resp.Write(buf)
		}

		if strings.Contains(req.RequestURI, "/realms/Broker/protocol/openid-connect/token") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			resp.Write([]byte("response"))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/environment") {
			buf, _ := json.Marshal(environmentRes)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/api/v1/platformTeams") {
			buf, _ := json.Marshal(teams)
			resp.Write(buf)
			return
		}
	}))

	return s
}

func TestLowerAndUpperLimitDurations(t *testing.T) {
	testCases := []struct {
		name             string
		durationProperty string
		defaultDuration  time.Duration
		description      string
		lowerLimit       time.Duration
		upperLimit       time.Duration
		expectPanic      bool
	}{
		{
			// valid range
			name:             "Agent Duration Property - valid range",
			durationProperty: "agent.duration",
			defaultDuration:  25 * time.Second,
			description:      "Agent Duration Property - valid range",
			lowerLimit:       20 * time.Second,
			upperLimit:       40 * time.Second,
		},
		{
			// lower limit is invalid
			/*
				{"level":"warning","message":"value 30s is lower than the supported lower limit (40s) for configuration agentDuration","time":"2022-07-26T14:42:54-07:00"}
				{"level":"warning","message":"config agentDuration has been set to the the default value of 25s.","time":"2022-07-26T14:42:54-07:00"}
			*/
			name:             "Agent Duration Property - invalid lower limit",
			durationProperty: "agent.duration",
			defaultDuration:  40 * time.Second,
			description:      "Agent Duration Property - invalid lower limit",
			lowerLimit:       40 * time.Second,
			upperLimit:       50 * time.Second,
		},
		{
			// default lower than lower limit
			name:             "Agent Duration Property - invalid upper limit",
			durationProperty: "agent.duration",
			defaultDuration:  5 * time.Second,
			description:      "Agent Duration Property - invalid upper limit",
			lowerLimit:       10 * time.Second,
			upperLimit:       20 * time.Second,
			expectPanic:      true,
		},
		{
			// upper limit is invalid
			/*
				{"level":"warning","message":"value 30s is higher than the supported higher limit (20s) for configuration agentDuration","time":"2022-07-26T14:42:54-07:00"}
				{"level":"warning","message":"config agentDuration has been set to the the default value of 30s.","time":"2022-07-26T14:42:54-07:00"}
			*/
			name:             "Agent Duration Property - invalid upper limit",
			durationProperty: "agent.duration",
			defaultDuration:  20 * time.Second,
			description:      "Agent Duration Property - invalid upper limit",
			lowerLimit:       10 * time.Second,
			upperLimit:       20 * time.Second,
		},
		{
			// default higher than upper limit
			name:             "Agent Duration Property - invalid upper limit",
			durationProperty: "agent.duration",
			defaultDuration:  40 * time.Second,
			description:      "Agent Duration Property - invalid upper limit",
			lowerLimit:       10 * time.Second,
			upperLimit:       20 * time.Second,
			expectPanic:      true,
		},
		{
			// upper lower than lower limit
			name:             "Agent Duration Property - invalid upper limit",
			durationProperty: "agent.duration",
			defaultDuration:  15 * time.Second,
			description:      "Agent Duration Property - invalid upper limit",
			lowerLimit:       10 * time.Second,
			upperLimit:       5 * time.Second,
			expectPanic:      true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			s := newTestServer()
			defer s.Close()

			var rootCmd AgentRootCmd
			var cfg *configWithValidation
			initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
				cfg = &configWithValidation{
					configValidationCalled: false,
					CentralCfg:             centralConfig,
					AgentCfg: &agentConfig{
						agentValidationCalled: false,
						dProp:                 rootCmd.GetProperties().DurationPropertyValue("agent.duration"),
					},
				}
				return cfg, nil
			}

			os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
			os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
			os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
			os.Setenv("CENTRAL_AUTH_URL", s.URL)
			os.Setenv("CENTRAL_URL", s.URL)
			os.Setenv("CENTRAL_SINGLEURL", s.URL)
			os.Setenv("AGENT_DURATION", "30s")

			rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
			viper.AddConfigPath("./testdata")
			fExecute := func() {
				rootCmd.GetProperties().AddDurationProperty(test.durationProperty, test.defaultDuration, test.description, properties.WithLowerLimit(test.lowerLimit), properties.WithUpperLimit(test.upperLimit))
			}
			if test.expectPanic {
				assert.Panics(t, fExecute)
			} else {
				assert.NotPanics(t, fExecute)
				_ = rootCmd.Execute()
			}
		})
	}
}
