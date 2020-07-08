package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/exception"
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
	// PublishToCatalog (formerly Disconnected) - publish items to Catalog
	PublishToCatalog AgentMode = iota + 1
	// PublishToEnvironment (formerly Connected) - publish items to Environment
	PublishToEnvironment
	// PublishToEnvironmentAndCatalog - publish items to both Catalog and Environment
	PublishToEnvironmentAndCatalog
)

// AgentModeStringMap - Map the Agent Mode constant to a string
var AgentModeStringMap = map[AgentMode]string{
	PublishToCatalog:               "publishToCatalog",
	PublishToEnvironment:           "publishToEnvironment",
	PublishToEnvironmentAndCatalog: "publishToEnvironmentAndCatalog",
}

// StringAgentModeMap - Map the string to the Agent Mode constant. Note that the strings are lowercased. In the config parser
// we change the string to all lowers to all for mis-typing of the case
var StringAgentModeMap = map[string]AgentMode{
	"publishtocatalog":               PublishToCatalog,
	"publishtoenvironment":           PublishToEnvironment,
	"publishtoenvironmentandcatalog": PublishToEnvironmentAndCatalog,
}

// CentralConfig - Interface to get central Config
type CentralConfig interface {
	GetAgentType() AgentType
	IsPublishToCatalogMode() bool
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
	GetTeamID() string
	GetURL() string
	GetPlatformURL() string
	GetCatalogItemsURL() string
	GetCatalogItemImageURL(catalogItemID string) string
	GetCatalogItemRelationshipsURL(catalogItemID string) string
	GetEnvironmentURL() string
	GetAPIServerURL() string
	GetAPIServerEnvironmentURL() string
	GetAPIServerServicesURL() string
	GetAPIServerServicesRevisionsURL() string
	GetAPIServerServicesInstancesURL() string
	DeleteAPIServerServicesURL() string
	GetAPIServerConsumerInstancesURL() string
	GetAPIServerSubscriptionDefinitionURL() string
	GetSubscriptionURL() string
	GetCatalogItemSubscriptionsURL(string) string
	Validate() error
	GetSubscriptionConfig() SubscriptionConfig
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
	GetTagsToPublish() string
	GetProxyURL() string
	SetProxyEnvironmentVariable() error
	GetPollInterval() time.Duration
	UpdateCatalogItemRevisionsURL(catalogItemID string) string
	GetCatalogItemByIDURL(catalogItemID string) string
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	AgentType        AgentType
	Mode             AgentMode     `config:"mode"`
	TenantID         string        `config:"tenantID"`
	TeamID           string        `config:"teamID" `
	APICDeployment   string        `config:"deployment"`
	Environment      string        `config:"environment"`
	URL              string        `config:"url"`
	PlatformURL      string        `config:"platformURL"`
	APIServerVersion string        `config:"apiServerVersion"`
	TagsToPublish    string        `config:"additionalTags"`
	Auth             AuthConfig    `config:"auth"`
	TLS              TLSConfig     `config:"ssl"`
	PollInterval     time.Duration `config:"pollInterval"`
	ProxyURL         string        `config:"proxyUrl"`
	environmentID    string
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	return &CentralConfiguration{
		AgentType:        agentType,
		Mode:             PublishToCatalog,
		APIServerVersion: "v1alpha1",
		Auth:             newAuthConfig(),
		TLS:              NewTLSConfig(),
		PollInterval:     60 * time.Second,
		PlatformURL:      "https://platform.axwaytest.net",
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

// IsPublishToCatalogMode -
func (c *CentralConfiguration) IsPublishToCatalogMode() bool {
	return c.Mode == PublishToCatalog
}

// IsPublishToEnvironmentMode -
func (c *CentralConfiguration) IsPublishToEnvironmentMode() bool {
	return c.Mode == PublishToEnvironment || c.IsPublishToEnvironmentAndCatalogMode()
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

// GetTeamID - Returns the team ID
func (c *CentralConfiguration) GetTeamID() string {
	return c.TeamID
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

// GetCatalogItemsURL - Returns the URL for catalog items API
func (c *CentralConfiguration) GetCatalogItemsURL() string {
	return c.URL + "/api/unifiedCatalog/v1/catalogItems"
}

// GetCatalogItemImageURL - Returns the image based on catalogItemID
func (c *CentralConfiguration) GetCatalogItemImageURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID + "/image"
}

// GetCatalogItemRelationshipsURL - Returns the image based on catalogItemID
func (c *CentralConfiguration) GetCatalogItemRelationshipsURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID + "/relationships"
}

// GetEnvironmentURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetEnvironmentURL() string {
	return c.URL + "/api/v1/environments"
}

// GetAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) GetAPIServerURL() string {
	return c.URL + "/apis/management/" + c.APIServerVersion + "/environments/"
}

// GetAPIServerEnvironmentURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetAPIServerEnvironmentURL() string {
	return c.GetAPIServerURL() + c.Environment
}

// GetAPIServerServicesURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetAPIServerServicesURL() string {
	return c.GetAPIServerEnvironmentURL() + "/apiservices"
}

// GetAPIServerServicesRevisionsURL - Returns the APIServer URL for services API revisions
func (c *CentralConfiguration) GetAPIServerServicesRevisionsURL() string {
	return c.GetAPIServerEnvironmentURL() + "/apiservicerevisions"
}

// GetAPIServerServicesInstancesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerServicesInstancesURL() string {
	return c.GetAPIServerEnvironmentURL() + "/apiserviceinstances"
}

// DeleteAPIServerServicesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) DeleteAPIServerServicesURL() string {
	return c.GetAPIServerEnvironmentURL() + "/apiservices"
}

// GetAPIServerConsumerInstancesURL - Returns the APIServer URL for services API consumer instances
func (c *CentralConfiguration) GetAPIServerConsumerInstancesURL() string {
	return c.GetAPIServerEnvironmentURL() + "/consumerinstances"
}

// GetAPIServerSubscriptionDefinitionURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerSubscriptionDefinitionURL() string {
	return c.GetAPIServerEnvironmentURL() + "/consumersubscriptiondefs"
}

// GetSubscriptionURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetSubscriptionURL() string {
	return c.URL + "/api/unifiedCatalog/v1/subscriptions"
}

// GetCatalogItemSubscriptionsURL - Returns the APIServer URL for catalog item subscriptions
func (c *CentralConfiguration) GetCatalogItemSubscriptionsURL(catalogItemID string) string {
	return fmt.Sprintf("%s/%s/subscriptions", c.GetCatalogItemsURL(), catalogItemID)
}

// GetAuthConfig - Returns the Auth Config
func (c *CentralConfiguration) GetAuthConfig() AuthConfig {
	return c.Auth
}

// GetTLSConfig - Returns the TLS Config
func (c *CentralConfiguration) GetTLSConfig() TLSConfig {
	return c.TLS
}

// GetTagsToPublish - Returns tags to publish
func (c *CentralConfiguration) GetTagsToPublish() string {
	return c.TagsToPublish
}

// GetCatalogItemByIDURL - Returns URL to get catalog item by id
func (c *CentralConfiguration) GetCatalogItemByIDURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID
}

// UpdateCatalogItemRevisionsURL - Returns URL to update catalog revision
func (c *CentralConfiguration) UpdateCatalogItemRevisionsURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID + "/revisions"
}

// GetPollInterval - Returns the interval for polling subscriptions
func (c *CentralConfiguration) GetPollInterval() time.Duration {
	return c.PollInterval
}

// Validate - Validates the config
func (c *CentralConfiguration) Validate() (err error) {
	exception.Block{
		Try: func() {
			c.validateConfig()
			c.Auth.validate()
			c.TLS.Validate()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (c *CentralConfiguration) validateConfig() {
	if c.GetTenantID() == "" {
		exception.Throw(errors.New("Error central.tenantID not set in config"))
	}

	if c.GetURL() == "" {
		exception.Throw(errors.New("Error central.url not set in config"))
	}

	if c.GetPlatformURL() == "" {
		exception.Throw(errors.New("Error central.platformURL not set in config"))
	}

	if c.GetAgentType() == TraceabilityAgent {
		c.validateTraceabilityAgentConfig()
	} else {
		c.validateDiscoveryAgentConfig()
	}
}

func (c *CentralConfiguration) validateDiscoveryAgentConfig() {
	if c.GetTeamID() == "" {
		exception.Throw(errors.New("Error central.teamID not set in config"))
	}

	if c.IsPublishToEnvironmentMode() {
		c.validatePublishToEnvironmentModeConfig()
	}

	if c.GetPollInterval() <= 0 {
		exception.Throw(errors.New("Error central.pollInterval not set in config"))
	}
}

func (c *CentralConfiguration) validatePublishToEnvironmentModeConfig() {
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
	pathTenantID              = "central.tenantId"
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
	pathDeployment            = "central.deployment"
	pathMode                  = "central.mode"
	pathTeamID                = "central.teamId"
	pathPollInterval          = "central.pollInterval"
	pathProxyURL              = "central.proxyUrl"
	pathAPIServerVersion      = "central.apiServerVersion"
	pathAdditionalTags        = "central.additionalTags"
)

// AddCentralConfigProperties - Adds the command properties needed for Central Config
func AddCentralConfigProperties(props properties.Properties, agentType AgentType) {
	props.AddStringProperty(pathTenantID, "", "Tenant ID for the owner of the environment")
	props.AddStringProperty(pathURL, "https://apicentral.axway.com", "URL of AMPLIFY Central")
	props.AddStringProperty(pathPlatformURL, "https://platform.axwaytest.net", "URL of the platform")
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
	props.AddStringProperty(pathProxyURL, "", "The Proxy URL to use for communication to AMPLIFY Central")

	if agentType == TraceabilityAgent {
		props.AddStringProperty(pathDeployment, "prod", "AMPLIFY Central")
	} else {
		props.AddStringProperty(pathMode, "publishToCatalog", "Agent Mode")
		props.AddStringProperty(pathTeamID, "", "Team ID for the current default team for creating catalog")
		props.AddDurationProperty(pathPollInterval, 60*time.Second, "The time interval at which the central will be polled for subscription processing.")
		props.AddStringProperty(pathAPIServerVersion, "v1alpha1", "Version of the API Server")
		props.AddStringProperty(pathAdditionalTags, "", "Additional Tags to Add to discovered APIs when publishing to AMPLIFY Central")
	}
}

// ParseCentralConfig - Parses the Central Config values form teh command line
func ParseCentralConfig(props properties.Properties, agentType AgentType) (CentralConfig, error) {
	proxyURL := props.StringPropertyValue(pathProxyURL)
	cfg := &CentralConfiguration{
		AgentType:    agentType,
		TenantID:     props.StringPropertyValue(pathTenantID),
		PollInterval: props.DurationPropertyValue(pathPollInterval),
		Environment:  props.StringPropertyValue(pathEnvironment),
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
		cfg.TeamID = props.StringPropertyValue(pathTeamID)
		cfg.TagsToPublish = props.StringPropertyValue(pathAdditionalTags)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
