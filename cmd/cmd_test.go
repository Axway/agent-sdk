package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func getPFlag(cmd AgentRootCmd, flagName string) *flag.Flag {
	return cmd.RootCmd().Flags().Lookup(flagName)
}

func assertCmdFlag(t *testing.T, cmd AgentRootCmd, flagName, fType, description string) {
	pflag := getPFlag(cmd, flagName)
	assert.NotNil(t, pflag)
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
func TestRootCmdFlags(t *testing.T) {

	// Discovery Agent
	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	assertStringCmdFlag(t, rootCmd, "central.mode", "centralMode", "disconnected", "Agent Mode")
	assertStringCmdFlag(t, rootCmd, "central.url", "centralUrl", "https://apicentral.preprod.k8s.axwayamplify.com", "URL of API Central")
	assertStringCmdFlag(t, rootCmd, "central.tenantId", "centralTenantId", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.teamId", "centralTeamId", "", "Team ID for the current default team for creating catalog")
	assertStringCmdFlag(t, rootCmd, "central.apiServerEnvironment", "apiServerEnvironment", "", "The Environment that the APIs will be associated with in API Central")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "authPrivateKey", "/etc/private_key.pem", "Path to the private key for API Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "authPublicKey", "/etc/public_key", "Path to the public key for API Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.keyPassword", "authKeyPassword", "", "Password for the private key, if needed")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "authUrl", "https://login-preprod.axway.com/auth", "API Central authentication URL")
	assertStringCmdFlag(t, rootCmd, "central.auth.realm", "authRealm", "Broker", "API Central authentication Realm")
	assertStringCmdFlag(t, rootCmd, "central.auth.clientId", "authClientId", "", "Client ID for the service account")
	assertDurationCmdFlag(t, rootCmd, "central.auth.timeout", "authTimeout", 10*time.Second, "Timeout waiting for AxwayID response")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.nextProtos", "centralSSLNextProtos", []string{}, "List of supported application level protocols, comma separated")
	assertBooleanCmdFlag(t, rootCmd, "central.ssl.insecureSkipVerify", "centralSSLInsecureSkipVerify", false, "Controls whether a client verifies the server's certificate chain and host name")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.cipherSuites", "centralSSLCipherSuites", corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	assertStringCmdFlag(t, rootCmd, "central.ssl.minVersion", "centralSSLMinVersion", corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	assertStringCmdFlag(t, rootCmd, "central.ssl.maxVersion", "centralSSLMaxVersion", "0", "Maximum acceptable SSL/TLS protocol version")

	// Traceability Agent
	rootCmd = NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	assertStringCmdFlag(t, rootCmd, "central.deployment", "centralDeployment", "preprod", "API Central")
	assertStringCmdFlag(t, rootCmd, "central.tenantId", "centralTenantId", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.environmentId", "centralEnvironmentId", "", "Environment ID for the current environment")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "authPrivateKey", "/etc/private_key.pem", "Path to the private key for API Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "authPublicKey", "/etc/public_key", "Path to the public key for API Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.keyPassword", "authKeyPassword", "", "Password for the private key, if needed")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "authUrl", "https://login-preprod.axway.com/auth", "API Central authentication URL")
	assertStringCmdFlag(t, rootCmd, "central.auth.realm", "authRealm", "Broker", "API Central authentication Realm")
	assertStringCmdFlag(t, rootCmd, "central.auth.clientId", "authClientId", "", "Client ID for the service account")
	assertDurationCmdFlag(t, rootCmd, "central.auth.timeout", "authTimeout", 10*time.Second, "Timeout waiting for AxwayID response")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.nextProtos", "centralSSLNextProtos", []string{}, "List of supported application level protocols, comma separated")
	assertBooleanCmdFlag(t, rootCmd, "central.ssl.insecureSkipVerify", "centralSSLInsecureSkipVerify", false, "Controls whether a client verifies the server's certificate chain and host name")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.cipherSuites", "centralSSLCipherSuites", corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	assertStringCmdFlag(t, rootCmd, "central.ssl.minVersion", "centralSSLMinVersion", corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	assertStringCmdFlag(t, rootCmd, "central.ssl.maxVersion", "centralSSLMaxVersion", "0", "Maximum acceptable SSL/TLS protocol version")

	// Log yaml properties and command flags
	assertStringCmdFlag(t, rootCmd, "log.level", "logLevel", "info", "Log level (debug, info, warn, error)")
	assertStringCmdFlag(t, rootCmd, "log.format", "logFormat", "json", "Log format (json, line, package)")
	assertStringCmdFlag(t, rootCmd, "log.output", "logOutput", "stdout", "Log output type (stdout, file, both)")
	assertStringCmdFlag(t, rootCmd, "log.path", "logPath", "logs", "Log file path if output type is file or both")

}

func TestRootCmdConfigFileLoad(t *testing.T) {

	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	fExecute := func() {
		rootCmd.Execute()
	}
	assert.Panics(t, fExecute)

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", nil, nil, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	fExecute = func() {
		rootCmd.Execute()
	}
	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Contains(t, "Error central.tenantID not set in config", errBuf.String())
}

func TestRootCmdConfigDefault(t *testing.T) {
	discoveryInitConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		assert.Equal(t, corecfg.Disconnected, centralConfig.GetAgentMode())
		assert.Equal(t, "https://apicentral.preprod.k8s.axwayamplify.com", centralConfig.GetURL())
		assert.Equal(t, "222222", centralConfig.GetTeamID())
		assert.Equal(t, "https://login-preprod.axway.com/auth/realms/Broker", centralConfig.GetAuthConfig().GetAudience())
		assert.Equal(t, "https://login-preprod.axway.com/auth/realms/Broker/protocol/openid-connect/token", centralConfig.GetAuthConfig().GetTokenURL())
		assert.Equal(t, "cccc", centralConfig.GetAuthConfig().GetClientID())
		assert.Equal(t, "Broker", centralConfig.GetAuthConfig().GetRealm())
		assert.Equal(t, "/etc/private_key.pem", centralConfig.GetAuthConfig().GetPrivateKey())
		assert.Equal(t, "/etc/public_key", centralConfig.GetAuthConfig().GetPublicKey())
		assert.Equal(t, "", centralConfig.GetAuthConfig().GetKeyPassword())
		assert.Equal(t, 10*time.Second, centralConfig.GetAuthConfig().GetTimeout())
		return centralConfig, errors.New("Test return error from init config handler")
	}

	traceabilityInitConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		assert.Equal(t, "preprod", centralConfig.GetAPICDeployment())
		assert.Equal(t, "https://login-preprod.axway.com/auth/realms/Broker", centralConfig.GetAuthConfig().GetAudience())
		assert.Equal(t, "https://login-preprod.axway.com/auth/realms/Broker/protocol/openid-connect/token", centralConfig.GetAuthConfig().GetTokenURL())
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
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	fExecute := func() {
		rootCmd.Execute()
	}
	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Contains(t, "Test return error from init config handler, Discovery Agent", errBuf.String())

	// Traceability
	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", traceabilityInitConfigHandler, nil, corecfg.TraceabilityAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	fExecute = func() {
		rootCmd.Execute()
	}
	errBuf = new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Contains(t, "Test return error from init config handler, Traceability Agent", errBuf.String())
}

type IAgentCfgWithValidate interface {
	Validate() error
}

type agentConfig struct {
	bProp                 bool
	dProp                 time.Duration
	iProp                 int
	sProp                 string
	ssProp                []string
	agentValidationCalled bool
}

func (a *agentConfig) Validate() error {
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

func (c *configWithValidation) Validate() error {
	c.configValidationCalled = true
	if c.AgentCfg.sProp == "" {
		return errors.New("configWithValidation: String prop not set")
	}
	return nil
}

type configWithNoValidation struct {
	configValidationCalled bool
	CentralCfg             corecfg.CentralConfig
	AgentCfg               IAgentCfgWithValidate
}

func TestRootCmdAgentConfigValidation(t *testing.T) {
	var rootCmd AgentRootCmd
	var cfg *configWithValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.IntPropertyValue("agent.int"),
				sProp:                 rootCmd.StringPropertyValue("agent.string"),
				ssProp:                rootCmd.StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"

	rootCmd.AddBoolProperty("agent.bool", "agentBool", false, "Agent Bool Property")
	rootCmd.AddDurationProperty("agent.duration", "agentDuration", 10*time.Second, "Agent Duration Property")
	rootCmd.AddIntProperty("agent.int", "agentInt", 0, "Agent Int Property")
	rootCmd.AddStringProperty("agent.string", "agentString", "", "Agent String Property")
	rootCmd.AddStringSliceProperty("agent.stringSlice", "agentStringSlice", nil, "Agent String Slice Property")

	fExecute := func() {
		rootCmd.Execute()
	}
	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Contains(t, "configWithValidation: String prop not set", errBuf.String())
	assert.Equal(t, true, cfg.configValidationCalled)
	assert.Equal(t, false, cfg.AgentCfg.agentValidationCalled)
}

func TestRootCmdAgentConfigChildValidation(t *testing.T) {
	var rootCmd AgentRootCmd
	var cfg *configWithNoValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithNoValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.IntPropertyValue("agent.int"),
				sProp:                 rootCmd.StringPropertyValue("agent.string"),
				ssProp:                rootCmd.StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"

	rootCmd.AddBoolProperty("agent.bool", "agentBool", false, "Agent Bool Property")
	rootCmd.AddDurationProperty("agent.duration", "agentDuration", 10*time.Second, "Agent Duration Property")
	rootCmd.AddIntProperty("agent.int", "agentInt", 0, "Agent Int Property")
	rootCmd.AddStringProperty("agent.string", "agentString", "", "Agent String Property")
	rootCmd.AddStringSliceProperty("agent.stringSlice", "agentStringSlice", nil, "Agent String Slice Property")

	fExecute := func() {
		rootCmd.Execute()
	}
	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Contains(t, "agentConfig: String prop not set", errBuf.String())
	assert.Equal(t, false, cfg.configValidationCalled)
	assert.Equal(t, true, cfg.AgentCfg.(*agentConfig).agentValidationCalled)
}

func TestRootCmdHandlersWithError(t *testing.T) {
	var centralCfg corecfg.CentralConfig
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		centralCfg = centralConfig
		return centralConfig, nil
	}
	cmdHandler := func() error {
		centralCfg.GetAgentMode()
		return nil
	}
	rootCmd := NewRootCmd("Test", "TestRootCmd", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	fExecute := func() {
		rootCmd.Execute()
	}
	assert.Panics(t, fExecute)

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	fExecute = func() {
		str := rootCmd.(*agentRootCommand).configPath
		assert.Equal(t, "./testdata", str)
		rootCmd.Execute()
	}
	assert.NotPanics(t, fExecute)
}

func TestRootCmdHandlers(t *testing.T) {
	var rootCmd AgentRootCmd
	var cfg *configWithNoValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &configWithNoValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
			AgentCfg: &agentConfig{
				agentValidationCalled: false,
				bProp:                 rootCmd.BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.IntPropertyValue("agent.int"),
				sProp:                 rootCmd.StringPropertyValue("agent.string"),
				ssProp:                rootCmd.StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"

	rootCmd.AddBoolProperty("agent.bool", "agentBool", false, "Agent Bool Property")
	rootCmd.AddDurationProperty("agent.duration", "agentDuration", 10*time.Second, "Agent Duration Property")
	rootCmd.AddIntProperty("agent.int", "agentInt", 0, "Agent Int Property")
	rootCmd.AddStringProperty("agent.string", "agentString", "", "Agent String Property")
	rootCmd.AddStringSliceProperty("agent.stringSlice", "agentStringSlice", nil, "Agent String Slice Property")

	fExecute := func() {
		rootCmd.Execute()
	}
	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.NotPanics(t, fExecute)

	assert.Empty(t, "", errBuf.String())
	assert.Equal(t, false, cfg.configValidationCalled)
	agentCfg := cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, true, agentCfg.bProp)
	assert.Equal(t, 30*time.Second, agentCfg.dProp)
	assert.Equal(t, 555, agentCfg.iProp)
	assert.Equal(t, true, cmdHandlerInvoked)
}

func noOpInitConfigHandler(centralConfig corecfg.CentralConfig) (interface{}, error) {
	return centralConfig, nil
}

func noOpCmdHandler() error {
	return nil
}

func TestRootCommandLoggerStdout(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fExecute := func() {
		rootCmd.Execute()
	}
	assert.NotPanics(t, fExecute)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	var logData map[string]string
	json.Unmarshal([]byte(out), &logData)

	assert.Equal(t, "info", logData["level"])
	assert.Equal(t, "Starting test_with_non_defaults ()", logData["msg"])
}

func TestRootCommandLoggerFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	rootCmd.RootCmd().SetArgs([]string{
		"--logOutput",
		"file",
		"--logPath",
		"./tmplogs",
	},
	)
	// Make sure to delete file
	os.RemoveAll("./tmplogs/test_with_non_defaults.log")

	fExecute := func() {
		rootCmd.Execute()
	}
	assert.NotPanics(t, fExecute)

	dat, err := ioutil.ReadFile("./tmplogs/test_with_non_defaults.log")
	assert.Nil(t, err)
	var logData map[string]string
	json.Unmarshal([]byte(dat), &logData)

	assert.Equal(t, "info", logData["level"])
	assert.Equal(t, "Starting test_with_non_defaults ()", logData["msg"])
}

func TestRootCommandLoggerStdoutAndFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	rootCmd.(*agentRootCommand).configPath = "./testdata"
	rootCmd.RootCmd().SetArgs([]string{
		"--logOutput",
		"both",
		"--logPath",
		"./tmplogs",
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
