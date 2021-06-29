package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

func TestRootCmdFlags(t *testing.T) {

	// Discovery Agent
	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	assertStringCmdFlag(t, rootCmd, "central.mode", "centralMode", "publishToEnvironmentAndCatalog", "Agent Mode")
	assertStringCmdFlag(t, rootCmd, "central.url", "centralUrl", "https://apicentral.axway.com", "URL of Amplify Central")
	assertStringCmdFlag(t, rootCmd, "central.platformURL", "centralPlatformURL", "https://platform.axway.com", "URL of the platform")
	assertStringCmdFlag(t, rootCmd, "central.organizationID", "centralOrganizationID", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.team", "centralTeam", "", "Team name for creating catalog")
	assertStringCmdFlag(t, rootCmd, "central.environment", "centralEnvironment", "", "The Environment that the APIs will be associated with in Amplify Central")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "centralAuthPrivateKey", "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "centralAuthPublicKey", "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.keyPassword", "centralAuthKeyPassword", "", "Password for the private key, if needed")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "centralAuthUrl", "https://login.axway.com/auth", "Amplify Central authentication URL")
	assertStringCmdFlag(t, rootCmd, "central.auth.realm", "centralAuthRealm", "Broker", "Amplify Central authentication Realm")
	assertStringCmdFlag(t, rootCmd, "central.auth.clientId", "centralAuthClientId", "", "Client ID for the service account")
	assertDurationCmdFlag(t, rootCmd, "central.auth.timeout", "centralAuthTimeout", 10*time.Second, "Timeout waiting for AxwayID response")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.nextProtos", "centralSslNextProtos", []string{}, "List of supported application level protocols, comma separated")
	assertBooleanCmdFlag(t, rootCmd, "central.ssl.insecureSkipVerify", "centralSslInsecureSkipVerify", false, "Controls whether a client verifies the server's certificate chain and host name")
	assertStringSliceCmdFlag(t, rootCmd, "central.ssl.cipherSuites", "centralSslCipherSuites", corecfg.TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	assertStringCmdFlag(t, rootCmd, "central.ssl.minVersion", "centralSslMinVersion", corecfg.TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	assertStringCmdFlag(t, rootCmd, "central.ssl.maxVersion", "centralSslMaxVersion", "0", "Maximum acceptable SSL/TLS protocol version")

	// Traceability Agent
	rootCmd = NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	assertStringCmdFlag(t, rootCmd, "central.deployment", "centralDeployment", "prod", "Amplify Central")
	assertStringCmdFlag(t, rootCmd, "central.url", "centralUrl", "https://apicentral.axway.com", "URL of Amplify Central")
	assertStringCmdFlag(t, rootCmd, "central.platformURL", "centralPlatformURL", "https://platform.axway.com", "URL of the platform")
	assertStringCmdFlag(t, rootCmd, "central.organizationID", "centralOrganizationID", "", "Tenant ID for the owner of the environment")
	assertStringCmdFlag(t, rootCmd, "central.auth.privateKey", "centralAuthPrivateKey", "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.publicKey", "centralAuthPublicKey", "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	assertStringCmdFlag(t, rootCmd, "central.auth.keyPassword", "centralAuthKeyPassword", "", "Password for the private key, if needed")
	assertStringCmdFlag(t, rootCmd, "central.auth.url", "centralAuthUrl", "https://login.axway.com/auth", "Amplify Central authentication URL")
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
		assert.Equal(t, corecfg.PublishToEnvironmentAndCatalog, centralConfig.GetAgentMode())
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

type agentConfig struct {
	bProp                 bool
	dProp                 time.Duration
	iProp                 int
	sProp                 string
	sPropExt              string
	ssProp                []string
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

func TestRootCmdAgentConfigValidation(t *testing.T) {
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

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())
	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property")
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
	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
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
				bProp:                 rootCmd.GetProperties().BoolPropertyValue("agent.bool"),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue("agent.duration"),
				iProp:                 rootCmd.GetProperties().IntPropertyValue("agent.int"),
				sProp:                 rootCmd.GetProperties().StringPropertyValue("agent.string"),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue("agent.stringSlice"),
			},
		}
		return cfg, nil
	}

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property")
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
	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
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
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())

	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddBoolProperty("agent.bool", false, "Agent Bool Property")
	rootCmd.GetProperties().AddDurationProperty("agent.duration", 10*time.Second, "Agent Duration Property")
	rootCmd.GetProperties().AddIntProperty("agent.int", 0, "Agent Int Property")
	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")

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

	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
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

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())

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
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	var logData map[string]string
	json.Unmarshal([]byte(out), &logData)

	assert.Equal(t, "info", logData["level"])
	assert.Equal(t, "Starting test_with_non_defaults (-)", logData["msg"])

	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
}

func TestRootCommandLoggerFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())

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
	assert.Nil(t, err)
	var logData map[string]string
	json.Unmarshal([]byte(dat), &logData)

	assert.Equal(t, "info", logData["level"])
	assert.Equal(t, "Starting test_with_non_defaults (-)", logData["msg"])

	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
}

func TestRootCommandLoggerStdoutAndFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	tmpFile, _ := ioutil.TempFile("./", "key*")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "./"+tmpFile.Name())
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "./"+tmpFile.Name())

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

	// Remove the test keys file
	os.Remove("./" + tmpFile.Name())
}

func TestRootCmdHandlerWithSecretRefProperties(t *testing.T) {
	secret := v1alpha1.Secret{
		ResourceMeta: v1.ResourceMeta{Name: "agentSecret"},
		Spec: v1alpha1.SecretSpec{
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
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/secrets/agentSecret") {
			buf, _ := json.Marshal(secret)
			resp.Write(buf)
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
			},
		}
		return cfg, nil
	}
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}

	tmpFile, _ := ioutil.TempFile("./", "key*")
	defer os.Remove("./" + tmpFile.Name())

	os.Setenv("CENTRAL_AUTH_URL", s.URL+"/auth")
	os.Setenv("CENTRAL_AUTH_CLIENTID", "DOSA_1111")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", "../transaction/testdata/private_key.pem")
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", "../transaction/testdata/public_key")
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_ENVIRONMENT", "test")

	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath("./testdata")

	rootCmd.GetProperties().AddStringProperty("agent.string", "", "Agent String Property")
	rootCmd.GetProperties().AddStringSliceProperty("agent.stringSlice", nil, "Agent String Slice Property")

	// Case 1 : No secret resolution - use the value in config
	os.Setenv("AGENT_STRING", "testValue")
	os.Setenv("AGENT_STRINGEXT", "anotherTestValue")
	err := rootCmd.Execute()
	assert.Nil(t, err, "An unexpected error returned")
	agentCfg := cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "testValue", agentCfg.sProp)
	assert.Equal(t, "anotherTestValue", agentCfg.sPropExt)
	assert.Equal(t, true, cmdHandlerInvoked)

	// Case 2 : Invalid secret resolution - secret ref with invalid secret name,
	// config value will be set to empty string
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false
	os.Setenv("AGENT_STRING", "@Secret.invalidSecret.secretKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.invalidSecret.cachedSecretKey")
	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "agentConfig: String prop not set", err.Error())
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "", agentCfg.sProp)
	assert.Equal(t, "", agentCfg.sPropExt)
	assert.Equal(t, false, cmdHandlerInvoked)

	// Case 3 : Invalid secret resolution - secret ref with invalid key in secret
	// config value will be set to empty string
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.invalidKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.invalidSecret.cachedSecretKey")
	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "agentConfig: String prop not set", err.Error())
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "", agentCfg.sProp)
	assert.Equal(t, "", agentCfg.sPropExt)
	assert.Equal(t, false, cmdHandlerInvoked)

	// Case 4 : Successful secret resolution - use value in secret key
	// config value will be set to specified key in secret
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.secretKey")
	os.Setenv("AGENT_STRINGEXT", "@Secret.agentSecret.cachedSecretKey")
	err = rootCmd.Execute()
	assert.Nil(t, err)
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "secretValue", agentCfg.sProp)
	assert.Equal(t, "cachedSecretValue", agentCfg.sPropExt)
	assert.Equal(t, true, cmdHandlerInvoked)

	// Case 5 : Successful secret resolution with key separate with dots(.) - use value in secret key
	// config value will be set to specified key in secret
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.keyElement1.keyElement2")
	err = rootCmd.Execute()
	assert.Nil(t, err)
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "secretValue2", agentCfg.sProp)
	assert.Equal(t, true, cmdHandlerInvoked)
}
