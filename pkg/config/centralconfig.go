package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/exception"
)

// AgentType - Defines the type of agent
type AgentType int

const (
	// DiscoveryAgent - Type definition for discovery agent
	DiscoveryAgent AgentType = iota + 1
	// TraceabilityAgent - Type definition for traceability agent
	TraceabilityAgent
)

// AgentMode - Defines the agent mode
type AgentMode int

const (
	// PublishToEnvironment (formerly Connected) - publish items to Environment
	PublishToEnvironment AgentMode = iota + 1
	// PublishToEnvironmentAndCatalog - publish items to both Catalog and Environment
	PublishToEnvironmentAndCatalog
)

// subscription approval types
const (
	ManualApproval  string = "manual"
	AutoApproval    string = "auto"
	WebhookApproval string = "webhook"
)

// AgentModeStringMap - Map the Agent Mode constant to a string
var AgentModeStringMap = map[AgentMode]string{
	PublishToEnvironment:           "publishToEnvironment",
	PublishToEnvironmentAndCatalog: "publishToEnvironmentAndCatalog",
}

// StringAgentModeMap - Map the string to the Agent Mode constant. Note that the strings are lowercased. In the config parser
// we change the string to all lowers to all for mis-typing of the case
var StringAgentModeMap = map[string]AgentMode{
	"publishtoenvironment":           PublishToEnvironment,
	"publishtoenvironmentandcatalog": PublishToEnvironmentAndCatalog,
}

// AgentTypeName - Holds the name Agent type
var AgentTypeName string

// AgentVersion - Holds the version of agent
var AgentVersion string

// IConfigValidator - Interface to be implemented for config validation by agent
type IConfigValidator interface {
	ValidateCfg() error
}

// IResourceConfigCallback - Interface to be implemented by configs to apply API Server resource
// for agent and dataplane
type IResourceConfigCallback interface {
	ApplyResources(dataplaneResource *v1.ResourceInstance, agentResource *v1.ResourceInstance) error
}

// CentralConfig - Interface to get central Config
type CentralConfig interface {
	GetAgentType() AgentType
	IsPublishToEnvironmentMode() bool
	IsPublishToEnvironmentOnlyMode() bool
	IsPublishToEnvironmentAndCatalogMode() bool
	GetAgentMode() AgentMode
	GetAgentModeAsString() string
	GetTenantID() string
	GetAPICDeployment() string
	GetEnvironmentID() string
	SetEnvironmentID(environmentID string)
	GetEnvironmentName() string
	GetAgentName() string
	GetTeamName() string
	GetTeamID() string
	SetTeamID(teamID string)
	GetURL() string
	GetPlatformURL() string
	GetCatalogItemsURL() string
	GetAPIServerURL() string
	GetEnvironmentURL() string
	GetServicesURL() string
	GetRevisionsURL() string
	GetInstancesURL() string
	DeleteServicesURL() string
	GetConsumerInstancesURL() string
	GetAPIServerSubscriptionDefinitionURL() string
	GetAPIServerWebhooksURL() string
	GetAPIServerSecretsURL() string
	GetSubscriptionURL() string
	GetSubscriptionConfig() SubscriptionConfig
	GetCatalogItemSubscriptionsURL(string) string
	GetCatalogItemSubscriptionStatesURL(string, string) string
	GetCatalogItemSubscriptionPropertiesURL(string, string) string
	GetCatalogItemSubscriptionDefinitionPropertiesURL(string) string
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
	GetTagsToPublish() string
	GetProxyURL() string
	SetProxyEnvironmentVariable() error
	GetPollInterval() time.Duration
	GetCatalogItemByIDURL(catalogItemID string) string
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	IConfigValidator
	AgentType                 AgentType
	Mode                      AgentMode     `config:"mode"`
	TenantID                  string        `config:"organizationID"`
	TeamName                  string        `config:"team"`
	APICDeployment            string        `config:"deployment"`
	Environment               string        `config:"environment"`
	AgentName                 string        `config:"agentName"`
	URL                       string        `config:"url"`
	PlatformURL               string        `config:"platformURL"`
	APIServerVersion          string        `config:"apiServerVersion"`
	TagsToPublish             string        `config:"additionalTags"`
	Auth                      AuthConfig    `config:"auth"`
	TLS                       TLSConfig     `config:"ssl"`
	PollInterval              time.Duration `config:"pollInterval"`
	ProxyURL                  string        `config:"proxyUrl"`
	environmentID             string
	teamID                    string
	SubscriptionConfiguration SubscriptionConfig `config:"subscriptions"`
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	return &CentralConfiguration{
		AgentType:                 agentType,
		Mode:                      PublishToEnvironmentAndCatalog,
		APIServerVersion:          "v1alpha1",
		Auth:                      newAuthConfig(),
		TLS:                       NewTLSConfig(),
		PollInterval:              60 * time.Second,
		PlatformURL:               "https://platform.axway.com",
		SubscriptionConfiguration: NewSubscriptionConfig(),
	}
}

// GetPlatformURL - Returns the central base URL
func (c *CentralConfiguration) GetPlatformURL() string {
	return c.PlatformURL
}

// GetAgentType - Returns the agent type
func (c *CentralConfiguration) GetAgentType() AgentType {
	return c.AgentType
}

// IsPublishToEnvironmentOnlyMode -
func (c *CentralConfiguration) IsPublishToEnvironmentOnlyMode() bool {
	return c.Mode == PublishToEnvironment
}

// IsPublishToEnvironmentAndCatalogMode -
func (c *CentralConfiguration) IsPublishToEnvironmentAndCatalogMode() bool {
	return c.Mode == PublishToEnvironmentAndCatalog
}

// GetAgentMode - Returns the agent mode
func (c *CentralConfiguration) GetAgentMode() AgentMode {
	return c.Mode
}

// GetAgentModeAsString - Returns the agent mode
func (c *CentralConfiguration) GetAgentModeAsString() string {
	return AgentModeStringMap[c.Mode]
}

// GetTenantID - Returns the tenant ID
func (c *CentralConfiguration) GetTenantID() string {
	return c.TenantID
}

// GetAPICDeployment - Returns the Central deployment type 'prod', 'preprod', team ('beano')
func (c *CentralConfiguration) GetAPICDeployment() string {
	return c.APICDeployment
}

// GetEnvironmentID - Returns the environment ID
func (c *CentralConfiguration) GetEnvironmentID() string {
	return c.environmentID
}

// SetEnvironmentID - Sets the environment ID
func (c *CentralConfiguration) SetEnvironmentID(environmentID string) {
	c.environmentID = environmentID
}

// GetEnvironmentName - Returns the environment name
func (c *CentralConfiguration) GetEnvironmentName() string {
	return c.Environment
}

// GetAgentName - Returns the agent name
func (c *CentralConfiguration) GetAgentName() string {
	return c.AgentName
}

// GetTeamName - Returns the team name
func (c *CentralConfiguration) GetTeamName() string {
	return c.TeamName
}

// GetTeamID - Returns the team ID
func (c *CentralConfiguration) GetTeamID() string {
	return c.teamID
}

// SetTeamID - Sets the team ID
func (c *CentralConfiguration) SetTeamID(teamID string) {
	c.teamID = teamID
}

// GetURL - Returns the central base URL
func (c *CentralConfiguration) GetURL() string {
	return c.URL
}

// GetProxyURL - Returns the central Proxy URL
func (c *CentralConfiguration) GetProxyURL() string {
	return c.ProxyURL
}

// SetProxyEnvironmentVariable - Set the proxy environment variable so the APIC auth uses the same proxy
func (c *CentralConfiguration) SetProxyEnvironmentVariable() (err error) {
	if c.GetProxyURL() != "" {
		urlInfo, err := url.Parse(c.GetProxyURL())
		if err == nil {
			if urlInfo.Scheme == "https" {
				os.Setenv("HTTPS_PROXY", c.GetProxyURL())
			} else if urlInfo.Scheme == "http" {
				os.Setenv("HTTP_PROXY", c.GetProxyURL())
			}
		}
	}
	return
}

// GetCatalogItemsURL - Returns the unifiedcatalog URL for catalog items API
func (c *CentralConfiguration) GetCatalogItemsURL() string {
	return c.URL + "/api/unifiedCatalog/v1/catalogItems"
}

// GetAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) GetAPIServerURL() string {
	return c.URL + "/apis/management/" + c.APIServerVersion + "/environments/"
}

// GetEnvironmentURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetEnvironmentURL() string {
	return c.GetAPIServerURL() + c.Environment
}

// GetServicesURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetServicesURL() string {
	return c.GetEnvironmentURL() + "/apiservices"
}

// GetRevisionsURL - Returns the APIServer URL for services API revisions
func (c *CentralConfiguration) GetRevisionsURL() string {
	return c.GetEnvironmentURL() + "/apiservicerevisions"
}

// GetInstancesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetInstancesURL() string {
	return c.GetEnvironmentURL() + "/apiserviceinstances"
}

// DeleteServicesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) DeleteServicesURL() string {
	return c.GetEnvironmentURL() + "/apiservices"
}

// GetConsumerInstancesURL - Returns the APIServer URL for services API consumer instances
func (c *CentralConfiguration) GetConsumerInstancesURL() string {
	return c.GetEnvironmentURL() + "/consumerinstances"
}

// GetAPIServerSubscriptionDefinitionURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerSubscriptionDefinitionURL() string {
	return c.GetEnvironmentURL() + "/consumersubscriptiondefs"
}

// GetAPIServerWebhooksURL - Returns the APIServer URL for webhooks instances
func (c *CentralConfiguration) GetAPIServerWebhooksURL() string {
	return c.GetEnvironmentURL() + "/webhooks"
}

// GetAPIServerSecretsURL - Returns the APIServer URL for secrets
func (c *CentralConfiguration) GetAPIServerSecretsURL() string {
	return c.GetEnvironmentURL() + "/secrets"
}

// GetSubscriptionURL - Returns the unifiedcatalog URL for subscriptions list
func (c *CentralConfiguration) GetSubscriptionURL() string {
	return c.URL + "/api/unifiedCatalog/v1/subscriptions"
}

// GetCatalogItemSubscriptionsURL - Returns the unifiedcatalog URL for catalog item subscriptions
func (c *CentralConfiguration) GetCatalogItemSubscriptionsURL(catalogItemID string) string {
	return fmt.Sprintf("%s/%s/subscriptions", c.GetCatalogItemsURL(), catalogItemID)
}

// GetCatalogItemSubscriptionStatesURL - Returns the unifiedcatalog URL for catalog item subscription states
func (c *CentralConfiguration) GetCatalogItemSubscriptionStatesURL(catalogItemID, subscriptionID string) string {
	return fmt.Sprintf("%s/%s/states", c.GetCatalogItemSubscriptionsURL(catalogItemID), subscriptionID)
}

// GetCatalogItemSubscriptionPropertiesURL - Returns the unifiedcatalog URL for catalog item subscription properties
func (c *CentralConfiguration) GetCatalogItemSubscriptionPropertiesURL(catalogItemID, subscriptionID string) string {
	return fmt.Sprintf("%s/%s/properties", c.GetCatalogItemSubscriptionsURL(catalogItemID), subscriptionID)
}

// GetCatalogItemSubscriptionDefinitionPropertiesURL - Returns the unifiedcatalog URL for catalog item subscription definition properties
func (c *CentralConfiguration) GetCatalogItemSubscriptionDefinitionPropertiesURL(catalogItemID string) string {
	return fmt.Sprintf("%s/%s/%s/properties", c.GetCatalogItemsURL(), catalogItemID, "subscriptionDefinition")
}

// GetAuthConfig - Returns the Auth Config
func (c *CentralConfiguration) GetAuthConfig() AuthConfig {
	return c.Auth
}

// GetTLSConfig - Returns the TLS Config
func (c *CentralConfiguration) GetTLSConfig() TLSConfig {
	return c.TLS
}

// GetSubscriptionConfig - Returns the Config for the subscription webhook
func (c *CentralConfiguration) GetSubscriptionConfig() SubscriptionConfig {
	return c.SubscriptionConfiguration
}

// GetTagsToPublish - Returns tags to publish
func (c *CentralConfiguration) GetTagsToPublish() string {
	return c.TagsToPublish
}

// GetCatalogItemByIDURL - Returns URL to get catalog item by id
func (c *CentralConfiguration) GetCatalogItemByIDURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID
}

// GetPollInterval - Returns the interval for polling subscriptions
func (c *CentralConfiguration) GetPollInterval() time.Duration {
	return c.PollInterval
}

// ValidateCfg - Validates the config, implementing IConfigInterface
func (c *CentralConfiguration) ValidateCfg() (err error) {
	exception.Block{
		Try: func() {
			c.validateConfig()
			c.Auth.validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (c *CentralConfiguration) validateConfig() {
	if c.GetTenantID() == "" {
		exception.Throw(errors.New("Error central.organizationID not set in config"))
	}

	if c.GetURL() == "" {
		exception.Throw(errors.New("Error central.url not set in config"))
	}

	if c.GetPlatformURL() == "" {
		exception.Throw(errors.New("Error central.platformURL not set in config"))
	}

	c.validatePublishToEnvironmentModeConfig()

	if c.GetAgentType() == TraceabilityAgent {
		c.validateTraceabilityAgentConfig()
	} else {
		c.validateDiscoveryAgentConfig()
	}
}

func (c *CentralConfiguration) validateDiscoveryAgentConfig() {
	if c.GetPollInterval() <= 0 {
		exception.Throw(errors.New("Error central.pollInterval not set in config"))
	}
}

func (c *CentralConfiguration) validatePublishToEnvironmentModeConfig() {
	if !c.IsPublishToEnvironmentOnlyMode() && !c.IsPublishToEnvironmentAndCatalogMode() {
		exception.Throw(errors.New("Error central.mode not configured for publishToEnvironment or publishToEnvironmentAndCatalog"))
	}

	if c.GetEnvironmentName() == "" {
		exception.Throw(errors.New("Error central.environment not set in config"))
	}

	if c.APIServerVersion == "" {
		exception.Throw(errors.New("Error central.apiServerVersion not set in config"))
	}
}

func (c *CentralConfiguration) validateTraceabilityAgentConfig() {
	if c.GetAPICDeployment() == "" {
		exception.Throw(errors.New("Error central.apicDeployment not set in config"))
	}
	if c.GetEnvironmentName() == "" {
		exception.Throw(errors.New("Error central.environment not set in config"))
	}
}

const (
	pathTenantID              = "central.organizationID"
	pathURL                   = "central.url"
	pathPlatformURL           = "central.platformURL"
	pathAuthPrivateKey        = "central.auth.privateKey"
	pathAuthPublicKey         = "central.auth.publicKey"
	pathAuthKeyPassword       = "central.auth.keyPassword"
	pathAuthURL               = "central.auth.url"
	pathAuthRealm             = "central.auth.realm"
	pathAuthClientID          = "central.auth.clientId"
	pathAuthTimeout           = "central.auth.timeout"
	pathSSLNextProtos         = "central.ssl.nextProtos"
	pathSSLInsecureSkipVerify = "central.ssl.insecureSkipVerify"
	pathSSLCipherSuites       = "central.ssl.cipherSuites"
	pathSSLMinVersion         = "central.ssl.minVersion"
	pathSSLMaxVersion         = "central.ssl.maxVersion"
	pathEnvironment           = "central.environment"
	pathAgentName             = "central.agentName"
	pathDeployment            = "central.deployment"
	pathMode                  = "central.mode"
	pathTeam                  = "central.team"
	pathPollInterval          = "central.pollInterval"
	pathProxyURL              = "central.proxyUrl"
	pathAPIServerVersion      = "central.apiServerVersion"
	pathAdditionalTags        = "central.additionalTags"
)

// AddCentralConfigProperties - Adds the command properties needed for Central Config
func AddCentralConfigProperties(props properties.Properties, agentType AgentType) {
	props.AddStringProperty(pathTenantID, "", "Tenant ID for the owner of the environment")
	props.AddStringProperty(pathURL, "https://apicentral.axway.com", "URL of AMPLIFY Central")
	props.AddStringProperty(pathTeam, "", "Team name for creating catalog")
	props.AddStringProperty(pathPlatformURL, "https://platform.axway.com", "URL of the platform")
	props.AddStringProperty(pathAuthPrivateKey, "/etc/private_key.pem", "Path to the private key for AMPLIFY Central Authentication")
	props.AddStringProperty(pathAuthPublicKey, "/etc/public_key", "Path to the public key for AMPLIFY Central Authentication")
	props.AddStringProperty(pathAuthKeyPassword, "", "Password for the private key, if needed")
	props.AddStringProperty(pathAuthURL, "https://login.axway.com/auth", "AMPLIFY Central authentication URL")
	props.AddStringProperty(pathAuthRealm, "Broker", "AMPLIFY Central authentication Realm")
	props.AddStringProperty(pathAuthClientID, "", "Client ID for the service account")
	props.AddDurationProperty(pathAuthTimeout, 10*time.Second, "Timeout waiting for AxwayID response")
	// ssl properties and command flags
	props.AddStringSliceProperty(pathSSLNextProtos, []string{}, "List of supported application level protocols, comma separated")
	props.AddBoolProperty(pathSSLInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name")
	props.AddStringSliceProperty(pathSSLCipherSuites, TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	props.AddStringProperty(pathSSLMinVersion, TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	props.AddStringProperty(pathSSLMaxVersion, "0", "Maximum acceptable SSL/TLS protocol version")
	props.AddStringProperty(pathEnvironment, "", "The Environment that the APIs will be associated with in AMPLIFY Central")
	props.AddStringProperty(pathAgentName, "", "The name of the asociated agent resource in AMPLIFY Central")
	props.AddStringProperty(pathProxyURL, "", "The Proxy URL to use for communication to AMPLIFY Central")

	if agentType == TraceabilityAgent {
		props.AddStringProperty(pathDeployment, "prod", "AMPLIFY Central")
	} else {
		props.AddStringProperty(pathMode, "publishToEnvironmentAndCatalog", "Agent Mode")
		props.AddDurationProperty(pathPollInterval, 60*time.Second, "The time interval at which the central will be polled for subscription processing.")
		props.AddStringProperty(pathAPIServerVersion, "v1alpha1", "Version of the API Server")
		props.AddStringProperty(pathAdditionalTags, "", "Additional Tags to Add to discovered APIs when publishing to AMPLIFY Central")
		AddSubscriptionConfigProperties(props)
	}
}

// ParseCentralConfig - Parses the Central Config values from the command line
func ParseCentralConfig(props properties.Properties, agentType AgentType) (CentralConfig, error) {
	proxyURL := props.StringPropertyValue(pathProxyURL)
	cfg := &CentralConfiguration{
		AgentType:    agentType,
		TenantID:     props.StringPropertyValue(pathTenantID),
		PollInterval: props.DurationPropertyValue(pathPollInterval),
		Environment:  props.StringPropertyValue(pathEnvironment),
		AgentName:    props.StringPropertyValue(pathAgentName),
		Auth: &AuthConfiguration{
			URL:        props.StringPropertyValue(pathAuthURL),
			Realm:      props.StringPropertyValue(pathAuthRealm),
			ClientID:   props.StringPropertyValue(pathAuthClientID),
			PrivateKey: props.StringPropertyValue(pathAuthPrivateKey),
			PublicKey:  props.StringPropertyValue(pathAuthPublicKey),
			KeyPwd:     props.StringPropertyValue(pathAuthKeyPassword),
			Timeout:    props.DurationPropertyValue(pathAuthTimeout),
		},
		TLS: &TLSConfiguration{
			NextProtos:         props.StringSlicePropertyValue(pathSSLNextProtos),
			InsecureSkipVerify: props.BoolPropertyValue(pathSSLInsecureSkipVerify),
			CipherSuites:       NewCipherArray(props.StringSlicePropertyValue(pathSSLCipherSuites)),
			MinVersion:         TLSVersionAsValue(props.StringPropertyValue(pathSSLMinVersion)),
			MaxVersion:         TLSVersionAsValue(props.StringPropertyValue(pathSSLMaxVersion)),
		},
		ProxyURL: proxyURL,
	}

	// Set the Proxy Environment Variable
	cfg.SetProxyEnvironmentVariable()

	if agentType == TraceabilityAgent {
		cfg.APICDeployment = props.StringPropertyValue(pathDeployment)
	} else {
		cfg.URL = props.StringPropertyValue(pathURL)
		cfg.PlatformURL = props.StringPropertyValue(pathPlatformURL)
		cfg.Mode = StringAgentModeMap[strings.ToLower(props.StringPropertyValue(pathMode))]
		cfg.APIServerVersion = props.StringPropertyValue(pathAPIServerVersion)
		cfg.TeamName = props.StringPropertyValue(pathTeam)
		cfg.TagsToPublish = props.StringPropertyValue(pathAdditionalTags)

		// set the notifications
		subscriptionConfig := ParseSubscriptionConfig(props)
		cfg.SubscriptionConfiguration = subscriptionConfig
	}

	return cfg, nil
}
