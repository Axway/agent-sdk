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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/config"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
)

// central config property keys, flag names, descriptions and defaults. These are hared by the
// Discovery/Compliance/Traceability asserted in TestRootCmdFlags.
const (
	pathCentralDeployment    = "central.deployment"
	flagCentralDeployment    = "centralDeployment"
	descCentralDeployment    = "Amplify Central"
	pathCentralSSLMaxVersion = "central.ssl.maxVersion"
	flagCentralSSLMaxVersion = "centralSslMaxVersion"
	descCentralSSLMaxVersion = "Maximum acceptable SSL/TLS protocol version"
	defaultSSLMaxVersion     = "0"

	pathCentralURL                   = "central.url"
	pathCentralPlatformURL           = "central.platformURL"
	pathCentralSingleURL             = "central.singleURL"
	pathCentralOrganizationID        = "central.organizationID"
	pathCentralAuthPrivateKey        = "central.auth.privateKey"
	pathCentralAuthPublicKey         = "central.auth.publicKey"
	pathCentralAuthKeyPassword       = "central.auth.keyPassword"
	pathCentralAuthURL               = "central.auth.url"
	pathCentralAuthRealm             = "central.auth.realm"
	pathCentralAuthClientID          = "central.auth.clientId"
	pathCentralAuthTimeout           = "central.auth.timeout"
	pathCentralSSLNextProtos         = "central.ssl.nextProtos"
	pathCentralSSLInsecureSkipVerify = "central.ssl.insecureSkipVerify"
	pathCentralSSLCipherSuites       = "central.ssl.cipherSuites"
	pathCentralSSLMinVersion         = "central.ssl.minVersion"

	flagCentralURL                   = "centralUrl"
	flagCentralPlatformURL           = "centralPlatformURL"
	flagCentralSingleURL             = "centralSingleURL"
	flagCentralOrganizationID        = "centralOrganizationID"
	flagCentralAuthPrivateKey        = "centralAuthPrivateKey"
	flagCentralAuthPublicKey         = "centralAuthPublicKey"
	flagCentralAuthKeyPassword       = "centralAuthKeyPassword"
	flagCentralAuthURL               = "centralAuthUrl"
	flagCentralAuthRealm             = "centralAuthRealm"
	flagCentralAuthClientID          = "centralAuthClientId"
	flagCentralAuthTimeout           = "centralAuthTimeout"
	flagCentralSSLNextProtos         = "centralSslNextProtos"
	flagCentralSSLInsecureSkipVerify = "centralSslInsecureSkipVerify"
	flagCentralSSLCipherSuites       = "centralSslCipherSuites"
	flagCentralSSLMinVersion         = "centralSslMinVersion"

	descCentralURL                   = "URL of Amplify Central"
	descCentralPlatformURL           = "URL of the platform"
	descCentralSingleURL             = "Alternate Connection for Agent if using static IP"
	descCentralOrganizationID        = "Tenant ID for the owner of the environment"
	descCentralAuthPrivateKey        = "Path to the private key for Amplify Central Authentication"
	descCentralAuthPublicKey         = "Path to the public key for Amplify Central Authentication"
	descCentralAuthKeyPassword       = "Path to the password file required by the private key for Amplify Central Authentication"
	descCentralAuthURL               = "Amplify Central authentication URL"
	descCentralAuthRealm             = "Amplify Central authentication Realm"
	descCentralAuthClientID          = "Client ID for the service account"
	descCentralAuthTimeout           = "Timeout waiting for AxwayID response"
	descCentralSSLNextProtos         = "List of supported application level protocols, comma separated"
	descCentralSSLInsecureSkipVerify = "Controls whether a client verifies the server's certificate chain and host name"
	descCentralSSLCipherSuites       = "List of supported cipher suites, comma separated"
	descCentralSSLMinVersion         = "Minimum acceptable SSL/TLS protocol version"

	defaultAuthPrivateKeyPath = "/etc/private_key.pem"
	defaultAuthPublicKeyPath  = "/etc/public_key"

	errStringPropNotSet       = "agentConfig: String prop not set"
	errOrganizationIDUnset    = "Error central.organizationID not set in config"
	errIncorrectErrorReturned = "Incorrect error returned: %s"

	testDataPath = "./testdata"

	pathAgentBool        = "agent.bool"
	pathAgentDuration    = "agent.duration"
	pathAgentInt         = "agent.int"
	pathAgentString      = "agent.string"
	pathAgentStringSlice = "agent.stringSlice"
	pathAgentObjectSlice = "agent.objectSlice"

	descAgentBoolProperty         = "Agent Bool Property"
	descAgentDurationProperty     = "Agent Duration Property"
	descAgentDurationInvalidUpper = "Agent Duration Property - invalid upper limit"
	descAgentIntProperty          = "Agent Int Property"
	descAgentStringProperty       = "Agent String Property"
	descAgentStringSliceProperty  = "Agent String Slice Property"

	testPrivateKeyPath = "../transaction/testdata/private_key.pem"
	testPublicKeyPath  = "../transaction/testdata/public_key"
	testLogPath        = "./tmplogs/test_with_non_defaults.log"

	secretInvalidCachedKey = "@Secret.invalidSecret.cachedSecretKey"
	secretAgentKey         = "@Secret.agentSecret.secretKey"

	testAuthToken = "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
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
		return errors.New(errStringPropNotSet)
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

type newCmdConfigValidation struct {
	configValidationCalled bool
	CentralCfg             corecfg.CentralConfig
}

type configWithNoValidation struct {
	configValidationCalled bool
	CentralCfg             corecfg.CentralConfig
	AgentCfg               corecfg.IConfigValidator
}

func TestRootCmdFlags(t *testing.T) {
	// Discovery Agent
	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)
	assertStringCmdFlag(t, rootCmd, pathCentralURL, flagCentralURL, "", descCentralURL)                         // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralPlatformURL, flagCentralPlatformURL, "", descCentralPlatformURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralSingleURL, flagCentralSingleURL, "", descCentralSingleURL)
	assertStringCmdFlag(t, rootCmd, pathCentralOrganizationID, flagCentralOrganizationID, "", descCentralOrganizationID)
	assertStringCmdFlag(t, rootCmd, "central.team", "centralTeam", "", "Team name for creating catalog")
	assertStringCmdFlag(t, rootCmd, "central.environment", "centralEnvironment", "", "The Environment that the APIs will be associated with in Amplify Central")
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPrivateKey, flagCentralAuthPrivateKey, defaultAuthPrivateKeyPath, descCentralAuthPrivateKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPublicKey, flagCentralAuthPublicKey, defaultAuthPublicKeyPath, descCentralAuthPublicKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthKeyPassword, flagCentralAuthKeyPassword, "", descCentralAuthKeyPassword)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthURL, flagCentralAuthURL, "", descCentralAuthURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralAuthRealm, flagCentralAuthRealm, "Broker", descCentralAuthRealm)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthClientID, flagCentralAuthClientID, "", descCentralAuthClientID)
	assertDurationCmdFlag(t, rootCmd, pathCentralAuthTimeout, flagCentralAuthTimeout, 10*time.Second, descCentralAuthTimeout)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLNextProtos, flagCentralSSLNextProtos, []string{}, descCentralSSLNextProtos)
	assertBooleanCmdFlag(t, rootCmd, pathCentralSSLInsecureSkipVerify, flagCentralSSLInsecureSkipVerify, false, descCentralSSLInsecureSkipVerify)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLCipherSuites, flagCentralSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), descCentralSSLCipherSuites)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMinVersion, flagCentralSSLMinVersion, corecfg.TLSDefaultMinVersionString(), descCentralSSLMinVersion)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMaxVersion, flagCentralSSLMaxVersion, defaultSSLMaxVersion, descCentralSSLMaxVersion)
	assertBooleanCmdFlag(t, rootCmd, "central.migration.cleanInstances", "centralMigrationCleanInstances", false, "Set this to clean all but latest instance, per stage, within an API Service")

	// Compliance Agent
	rootCmd = NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.ComplianceAgent)
	assertStringCmdFlag(t, rootCmd, pathCentralDeployment, flagCentralDeployment, "", descCentralDeployment)    // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralURL, flagCentralURL, "", descCentralURL)                         // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralPlatformURL, flagCentralPlatformURL, "", descCentralPlatformURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralSingleURL, flagCentralSingleURL, "", descCentralSingleURL)
	assertStringCmdFlag(t, rootCmd, pathCentralOrganizationID, flagCentralOrganizationID, "", descCentralOrganizationID)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPrivateKey, flagCentralAuthPrivateKey, defaultAuthPrivateKeyPath, descCentralAuthPrivateKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPublicKey, flagCentralAuthPublicKey, defaultAuthPublicKeyPath, descCentralAuthPublicKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthKeyPassword, flagCentralAuthKeyPassword, "", descCentralAuthKeyPassword)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthURL, flagCentralAuthURL, "", descCentralAuthURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralAuthRealm, flagCentralAuthRealm, "Broker", descCentralAuthRealm)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthClientID, flagCentralAuthClientID, "", descCentralAuthClientID)
	assertDurationCmdFlag(t, rootCmd, pathCentralAuthTimeout, flagCentralAuthTimeout, 10*time.Second, descCentralAuthTimeout)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLNextProtos, flagCentralSSLNextProtos, []string{}, descCentralSSLNextProtos)
	assertBooleanCmdFlag(t, rootCmd, pathCentralSSLInsecureSkipVerify, flagCentralSSLInsecureSkipVerify, false, descCentralSSLInsecureSkipVerify)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLCipherSuites, flagCentralSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), descCentralSSLCipherSuites)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMinVersion, flagCentralSSLMinVersion, corecfg.TLSDefaultMinVersionString(), descCentralSSLMinVersion)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMaxVersion, flagCentralSSLMaxVersion, defaultSSLMaxVersion, descCentralSSLMaxVersion)

	// Traceability Agent
	rootCmd = NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.TraceabilityAgent)
	assertStringCmdFlag(t, rootCmd, pathCentralDeployment, flagCentralDeployment, "", descCentralDeployment)    // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralURL, flagCentralURL, "", descCentralURL)                         // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralPlatformURL, flagCentralPlatformURL, "", descCentralPlatformURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralSingleURL, flagCentralSingleURL, "", descCentralSingleURL)
	assertStringCmdFlag(t, rootCmd, pathCentralOrganizationID, flagCentralOrganizationID, "", descCentralOrganizationID)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPrivateKey, flagCentralAuthPrivateKey, defaultAuthPrivateKeyPath, descCentralAuthPrivateKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthPublicKey, flagCentralAuthPublicKey, defaultAuthPublicKeyPath, descCentralAuthPublicKey)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthKeyPassword, flagCentralAuthKeyPassword, "", descCentralAuthKeyPassword)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthURL, flagCentralAuthURL, "", descCentralAuthURL) // assert to empty "" - set by region settings
	assertStringCmdFlag(t, rootCmd, pathCentralAuthRealm, flagCentralAuthRealm, "Broker", descCentralAuthRealm)
	assertStringCmdFlag(t, rootCmd, pathCentralAuthClientID, flagCentralAuthClientID, "", descCentralAuthClientID)
	assertDurationCmdFlag(t, rootCmd, pathCentralAuthTimeout, flagCentralAuthTimeout, 10*time.Second, descCentralAuthTimeout)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLNextProtos, flagCentralSSLNextProtos, []string{}, descCentralSSLNextProtos)
	assertBooleanCmdFlag(t, rootCmd, pathCentralSSLInsecureSkipVerify, flagCentralSSLInsecureSkipVerify, false, descCentralSSLInsecureSkipVerify)
	assertStringSliceCmdFlag(t, rootCmd, pathCentralSSLCipherSuites, flagCentralSSLCipherSuites, corecfg.TLSDefaultCipherSuitesStringSlice(), descCentralSSLCipherSuites)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMinVersion, flagCentralSSLMinVersion, corecfg.TLSDefaultMinVersionString(), descCentralSSLMinVersion)
	assertStringCmdFlag(t, rootCmd, pathCentralSSLMaxVersion, flagCentralSSLMaxVersion, defaultSSLMaxVersion, descCentralSSLMaxVersion)

	// Log yaml properties and command flags
	assertStringCmdFlag(t, rootCmd, "log.level", "logLevel", "info", "Log level (trace, debug, info, warn, error)")
	assertStringCmdFlag(t, rootCmd, "log.format", "logFormat", "json", "Log format (json, line)")
	assertStringCmdFlag(t, rootCmd, "log.output", "logOutput", "stdout", "Log output type (stdout, file, both)")
	assertStringCmdFlag(t, rootCmd, "log.file.path", "logFilePath", "logs", "Log file path if output type is file or both")
}

func TestRootCmdConfigFileLoad(t *testing.T) {

	rootCmd := NewRootCmd("Test", "TestRootCmd", nil, nil, corecfg.DiscoveryAgent)

	err := rootCmd.Execute()

	// a missing config file is now ok. The resulting error, if any, comes from downstream
	// config validation (nothing is set here), not from ConfigFileNotFoundError
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", nil, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)

	assert.Contains(t, errOrganizationIDUnset, errBuf.String())
}

func TestRootCmdConfigDefault(t *testing.T) {
	discoveryInitConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		assert.Equal(t, "https://apicentral.axway.com", centralConfig.GetURL())
		assert.Equal(t, "222222", centralConfig.GetTeamName())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker", centralConfig.GetAuthConfig().GetAudience())
		assert.Equal(t, "https://login.axway.com/auth/realms/Broker/protocol/openid-connect/token", centralConfig.GetAuthConfig().GetTokenURL())
		assert.Equal(t, "cccc", centralConfig.GetAuthConfig().GetClientID())
		assert.Equal(t, "Broker", centralConfig.GetAuthConfig().GetRealm())
		assert.Equal(t, defaultAuthPrivateKeyPath, centralConfig.GetAuthConfig().GetPrivateKey())
		assert.Equal(t, defaultAuthPublicKeyPath, centralConfig.GetAuthConfig().GetPublicKey())
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
		assert.Equal(t, defaultAuthPrivateKeyPath, centralConfig.GetAuthConfig().GetPrivateKey())
		assert.Equal(t, defaultAuthPublicKeyPath, centralConfig.GetAuthConfig().GetPublicKey())
		assert.Equal(t, "", centralConfig.GetAuthConfig().GetKeyPassword())
		assert.Equal(t, 10*time.Second, centralConfig.GetAuthConfig().GetTimeout())
		return centralConfig, errors.New("Test return error from init config handler")
	}

	// Discovery
	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", discoveryInitConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)
	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "Test return error from init config handler, Discovery Agent", errBuf.String())

	// Compliance
	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", traceabilityInitConfigHandler, nil, corecfg.ComplianceAgent)
	viper.AddConfigPath(testDataPath)
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	errBuf = new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, "Test return error from init config handler, Compliance Agent", errBuf.String())

	// Traceability
	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", traceabilityInitConfigHandler, nil, corecfg.TraceabilityAgent)
	viper.AddConfigPath(testDataPath)
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
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
				bProp:                 rootCmd.GetProperties().BoolPropertyValue(pathAgentBool),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue(pathAgentDuration),
				iProp:                 rootCmd.GetProperties().IntPropertyValue(pathAgentInt),
				sProp:                 rootCmd.GetProperties().StringPropertyValue(pathAgentString),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue(pathAgentStringSlice),
			},
		}
		return cfg, nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("CENTRAL_PLATFORMURL", s.URL)

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddBoolProperty(pathAgentBool, false, descAgentBoolProperty)
	rootCmd.GetProperties().AddDurationProperty(pathAgentDuration, 10*time.Second, descAgentDurationProperty, properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty(pathAgentInt, 0, descAgentIntProperty)
	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
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
				bProp:                 rootCmd.GetProperties().BoolPropertyValue(pathAgentBool),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue(pathAgentDuration),
				iProp:                 rootCmd.GetProperties().IntPropertyValue(pathAgentInt),
				sProp:                 rootCmd.GetProperties().StringPropertyValue(pathAgentString),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue(pathAgentStringSlice),
			},
		}
		return cfg, nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddBoolProperty(pathAgentBool, false, descAgentBoolProperty)
	rootCmd.GetProperties().AddDurationProperty(pathAgentDuration, 10*time.Second, descAgentDurationProperty, properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty(pathAgentInt, 0, descAgentIntProperty)
	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	errBuf := new(bytes.Buffer)
	rootCmd.RootCmd().SetErr(errBuf)
	assert.Contains(t, errStringPropNotSet, errBuf.String())
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

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("Test", "TestRootCmd", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	err := rootCmd.Execute()

	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	rootCmd = NewRootCmd("test_no_overide", "test_no_overide", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)
	err = rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.NotNil(t, err, err.Error())
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
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
				bProp:                 rootCmd.GetProperties().BoolPropertyValue(pathAgentBool),
				dProp:                 rootCmd.GetProperties().DurationPropertyValue(pathAgentDuration),
				iProp:                 rootCmd.GetProperties().IntPropertyValue(pathAgentInt),
				sProp:                 rootCmd.GetProperties().StringPropertyValue(pathAgentString),
				ssProp:                rootCmd.GetProperties().StringSlicePropertyValue(pathAgentStringSlice),
				osProp:                rootCmd.GetProperties().ObjectSlicePropertyValue(pathAgentObjectSlice),
			},
		}
		return cfg, nil
	}
	var cmdHandlerInvoked bool
	cmdHandler := func() error {
		cmdHandlerInvoked = true
		return nil
	}

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
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
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddBoolProperty(pathAgentBool, false, descAgentBoolProperty)
	rootCmd.GetProperties().AddDurationProperty(pathAgentDuration, 10*time.Second, descAgentDurationProperty, properties.WithLowerLimit(10*time.Second))
	rootCmd.GetProperties().AddIntProperty(pathAgentInt, 0, descAgentIntProperty)
	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"index", "name", "namevalue", "nametitle"})
	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.Nil(t, err, "An unexpected error returned")
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
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

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)

	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()

	// should NOT be FileNotFound error
	assert.Nil(t, err, "An unexpected error was received")
	if err != nil {
		_, ok := err.(viper.ConfigFileNotFoundError)
		assert.False(t, ok, errIncorrectErrorReturned, err.Error())
	}

	w.Close()

	var logData map[string]string
	scanner := bufio.NewScanner(r)

	level := "info"
	msg := "Starting test_with_non_defaults"

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
	defer func() {
		config.AgentVersion = ""
		SDKBuildVersion = ""
	}()

	config.AgentVersion = "1.2.3-abc123"
	SDKBuildVersion = "1.0.0"

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)
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
	os.RemoveAll(testLogPath)

	fExecute := func() {
		rootCmd.Execute()
	}
	assert.NotPanics(t, fExecute)

	dat, err := ioutil.ReadFile(testLogPath)
	assert.Nil(t, err, "failed to read file")
	scanner := bufio.NewScanner(bytes.NewReader(dat))

	var logData map[string]string
	level := "info"
	msg := "Starting test_with_non_defaults"

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
	assert.Equal(t, "1.2.3-abc123", logData["version"])
	assert.Equal(t, "1.0.0", logData["sdkVersion"])
}

func TestRootCommandLoggerStdoutAndFile(t *testing.T) {
	initConfigHandler := noOpInitConfigHandler
	cmdHandler := noOpCmdHandler

	s := newTestServer()
	defer s.Close()

	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)

	rootCmd := NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)
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
	os.Remove(testLogPath)
	assert.NotPanics(t, fExecute)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout
	var logData map[string]string
	json.Unmarshal([]byte(out), &logData)

	dat, err := ioutil.ReadFile(testLogPath)
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
			token := testAuthToken
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/test/secrets/agentSecret") {
			buf, _ := json.Marshal(secret)
			resp.Write(buf)
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/test") {
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
				sProp:                 rootCmd.GetProperties().StringPropertyValue(pathAgentString),
				sPropExt:              rootCmd.GetProperties().StringPropertyValue("agent.stringExt"),
				osProp:                rootCmd.GetProperties().ObjectSlicePropertyValue(pathAgentObjectSlice),
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
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("CENTRAL_ENVIRONMENT", "test")

	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"prop1", "prop2", "prop3"})

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
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"prop1", "prop2", "prop3"})

	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false
	os.Setenv("AGENT_STRING", "@Secret.invalidSecret.secretKey")
	os.Setenv("AGENT_STRINGEXT", secretInvalidCachedKey)
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", "@Secret.invalidSecret.secretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", secretInvalidCachedKey)

	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, errStringPropNotSet, err.Error())
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
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"prop1", "prop2", "prop3"})

	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", "@Secret.agentSecret.invalidKey")
	os.Setenv("AGENT_STRINGEXT", secretInvalidCachedKey)
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", secretAgentKey)
	os.Setenv("AGENT_OBJECTSLICE_PROP1_2", secretInvalidCachedKey)
	err = rootCmd.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, errStringPropNotSet, err.Error())
	agentCfg = cfg.AgentCfg.(*agentConfig)
	assert.Equal(t, true, agentCfg.agentValidationCalled)
	assert.Equal(t, "", agentCfg.sProp)
	assert.Equal(t, "", agentCfg.sPropExt)
	objectSliceProps = []string{agentCfg.osProp[0]["prop1"].(string), agentCfg.osProp[1]["prop1"].(string)}
	slices.Sort(objectSliceProps)
	assert.Equal(t, []string{"", "secretValue"}, objectSliceProps)
	assert.Equal(t, false, cmdHandlerInvoked)

	// Case 4 : Successful secret resolution - use value in secret key
	// config value will be set to specified key in secret
	rootCmd = NewRootCmd("test_with_agent_cfg", "test_with_agent_cfg", initConfigHandler, cmdHandler, corecfg.DiscoveryAgent)
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"prop1", "prop2", "prop3"})
	cfg = nil
	agentCfg.agentValidationCalled = false
	cmdHandlerInvoked = false

	os.Setenv("AGENT_STRING", secretAgentKey)
	os.Setenv("AGENT_STRINGEXT", "@Secret.agentSecret.cachedSecretKey")
	os.Setenv("AGENT_OBJECTSLICE_PROP1_1", secretAgentKey)
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
	viper.AddConfigPath(testDataPath)

	rootCmd.GetProperties().AddStringProperty(pathAgentString, "", descAgentStringProperty)
	rootCmd.GetProperties().AddStringSliceProperty(pathAgentStringSlice, nil, descAgentStringSliceProperty)
	rootCmd.GetProperties().AddObjectSliceProperty(pathAgentObjectSlice, []string{"prop1", "prop2", "prop3"})

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
			token := testAuthToken
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/test/secrets/agentSecret") {
			buf, _ := json.Marshal(secret)
			resp.Write(buf)
		}

		if strings.Contains(req.RequestURI, "/realms/Broker/protocol/openid-connect/token") {
			token := testAuthToken
			resp.Write([]byte(token))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/test/apiservices") {
			resp.Write([]byte("response"))
			return
		}

		if strings.Contains(req.RequestURI, "/apis/management/v1/environments/environment") {
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
			durationProperty: pathAgentDuration,
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
			durationProperty: pathAgentDuration,
			defaultDuration:  40 * time.Second,
			description:      "Agent Duration Property - invalid lower limit",
			lowerLimit:       40 * time.Second,
			upperLimit:       50 * time.Second,
		},
		{
			// default lower than lower limit
			name:             descAgentDurationInvalidUpper,
			durationProperty: pathAgentDuration,
			defaultDuration:  5 * time.Second,
			description:      descAgentDurationInvalidUpper,
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
			name:             descAgentDurationInvalidUpper,
			durationProperty: pathAgentDuration,
			defaultDuration:  20 * time.Second,
			description:      descAgentDurationInvalidUpper,
			lowerLimit:       10 * time.Second,
			upperLimit:       20 * time.Second,
		},
		{
			// default higher than upper limit
			name:             descAgentDurationInvalidUpper,
			durationProperty: pathAgentDuration,
			defaultDuration:  40 * time.Second,
			description:      descAgentDurationInvalidUpper,
			lowerLimit:       10 * time.Second,
			upperLimit:       20 * time.Second,
			expectPanic:      true,
		},
		{
			// upper lower than lower limit
			name:             descAgentDurationInvalidUpper,
			durationProperty: pathAgentDuration,
			defaultDuration:  15 * time.Second,
			description:      descAgentDurationInvalidUpper,
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
						dProp:                 rootCmd.GetProperties().DurationPropertyValue(pathAgentDuration),
					},
				}
				return cfg, nil
			}

			os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
			os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
			os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
			os.Setenv("CENTRAL_AUTH_URL", s.URL)
			os.Setenv("CENTRAL_URL", s.URL)
			os.Setenv("CENTRAL_SINGLEURL", s.URL)
			os.Setenv("AGENT_DURATION", "30s")

			rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
			viper.AddConfigPath(testDataPath)
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

func TestIntLowerAndUpperLimits(t *testing.T) {
	cases := map[string]struct {
		intProp     string
		defaultInt  int
		lowerLimit  int
		upperLimit  int
		useDefault  bool
		expectPanic bool
	}{
		"valid limits range - value out of limits": {
			intProp:     "10",
			defaultInt:  5,
			lowerLimit:  2,
			upperLimit:  8,
			useDefault:  true,
			expectPanic: false,
		},
		"valid limits range - value within limits": {
			intProp:     "6",
			defaultInt:  5,
			lowerLimit:  2,
			upperLimit:  8,
			useDefault:  false,
			expectPanic: false,
		},
		"invalid limits range - lower > upper": {
			intProp:     "5",
			defaultInt:  5,
			lowerLimit:  6,
			upperLimit:  5,
			useDefault:  false,
			expectPanic: true,
		},
		"default value out of limits": {
			intProp:     "5",
			defaultInt:  10,
			lowerLimit:  2,
			upperLimit:  8,
			useDefault:  false,
			expectPanic: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
						iProp:                 rootCmd.GetProperties().IntPropertyValue(pathAgentInt),
					},
				}
				return cfg, nil
			}

			os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
			os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
			os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
			os.Setenv("CENTRAL_AUTH_URL", s.URL)
			os.Setenv("CENTRAL_URL", s.URL)
			os.Setenv("CENTRAL_SINGLEURL", s.URL)
			os.Setenv("AGENT_INT", tc.intProp)

			rootCmd = NewRootCmd("test_with_non_defaults", "test_with_non_defaults", initConfigHandler, nil, corecfg.DiscoveryAgent)
			viper.AddConfigPath(testDataPath)
			fExecute := func() {
				rootCmd.GetProperties().AddIntProperty(pathAgentInt, tc.defaultInt, "", properties.WithLowerLimitInt(tc.lowerLimit), properties.WithUpperLimitInt(tc.upperLimit))
			}
			if tc.expectPanic {
				assert.Panics(t, fExecute)
			} else {
				assert.NotPanics(t, fExecute)
				_ = rootCmd.Execute()
				if tc.useDefault {
					assert.Equal(t, tc.defaultInt, cfg.AgentCfg.iProp)
				} else {
					assert.Equal(t, tc.intProp, strconv.Itoa(cfg.AgentCfg.iProp))
				}
			}
		})
	}
}

func TestNewCmd(t *testing.T) {
	s := newTestServer()
	defer s.Close()

	rootCmd := &cobra.Command{}
	var cfg *newCmdConfigValidation
	initConfigHandler := func(centralConfig corecfg.CentralConfig) (interface{}, error) {
		cfg = &newCmdConfigValidation{
			configValidationCalled: false,
			CentralCfg:             centralConfig,
		}
		return cfg, nil
	}
	cmdHandler := func() error {
		return nil
	}
	newCmd := NewCmd(rootCmd, "traceability", "TestRootCmd", initConfigHandler, cmdHandler, corecfg.TraceabilityAgent)
	viper.AddConfigPath(testDataPath)
	assert.NotNil(t, newCmd)

	os.Setenv("CENTRAL_AUTH_URL", s.URL)
	os.Setenv("CENTRAL_AUTH_CLIENTID", "serviceaccount_1234")
	os.Setenv("CENTRAL_AUTH_PRIVATEKEY", testPrivateKeyPath)
	os.Setenv("CENTRAL_AUTH_PUBLICKEY", testPublicKeyPath)
	os.Setenv("CENTRAL_URL", s.URL)
	os.Setenv("CENTRAL_SINGLEURL", s.URL)
	os.Setenv("CENTRAL_ORGANIZATIONID", " orgid")
	os.Setenv("CENTRAL_ENVIRONMENT", "environment ")
	defer os.Setenv("CENTRAL_ORGANIZATIONID", "")

	err := rootCmd.Execute()

	assert.Nil(t, err)
	assert.Equal(t, "environment", cfg.CentralCfg.GetEnvironmentName())
	assert.Equal(t, "orgid", cfg.CentralCfg.GetTenantID())

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	assert.Contains(t, errOrganizationIDUnset, errBuf.String())
}
