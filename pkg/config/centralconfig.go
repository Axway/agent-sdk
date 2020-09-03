package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/exception"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
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
	GetTeamName() string
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
	GetCatalogItemSubscriptionsURL(string) string
	Validate() error
	GetSubscriptionConfig() SubscriptionConfig
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
	AgentType                 AgentType
	Mode                      AgentMode     `config:"mode"`
	TenantID                  string        `config:"organizationID"`
	TeamName                  string        `config:"team"`
	APICDeployment            string        `config:"deployment"`
	Environment               string        `config:"environment"`
	URL                       string        `config:"url"`
	PlatformURL               string        `config:"platformURL"`
	APIServerVersion          string        `config:"apiServerVersion"`
	TagsToPublish             string        `config:"additionalTags"`
	Auth                      AuthConfig    `config:"auth"`
	TLS                       TLSConfig     `config:"ssl"`
	PollInterval              time.Duration `config:"pollInterval"`
	ProxyURL                  string        `config:"proxyUrl"`
	environmentID             string
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

// GetTeamName - Returns the team name
func (c *CentralConfiguration) GetTeamName() string {
	return c.TeamName
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

// ConfKey - int
type ConfKey int

// IMPORTANT NOTE : constants much match order and count of func (confKey ConfKey)
// Any adds or deletions to these constants MUST be reflected in func (confKey ConfKey) to keep keys and values in sync.
// Failure to keep the constants and confKey in sync will result in a failure to load the central configs
const (
	pathTenantID ConfKey = iota
	pathURL
	pathPlatformURL
	pathAuthPrivateKey
	pathAuthPublicKey
	pathAuthKeyPassword
	pathAuthURL
	pathAuthRealm
	pathAuthClientID
	pathAuthTimeout
	pathSSLNextProtos
	pathSSLInsecureSkipVerify
	pathSSLCipherSuites
	pathSSLMinVersion
	pathSSLMaxVersion
	pathEnvironment
	pathDeployment
	pathMode
	pathTeam
	pathPollInterval
	pathProxyURL
	pathAPIServerVersion
	pathAdditionalTags
)

// IMPORTANT NOTE : these need to be in sync with constants (above)
func (confKey ConfKey) String() string {
	keys := [...]string{
		"central.organizationID",
		"central.url",
		"central.platformURL",
		"central.auth.privateKey",
		"central.auth.publicKey",
		"central.auth.keyPassword",
		"central.auth.url",
		"central.auth.realm",
		"central.auth.clientId",
		"central.auth.timeout",
		"central.ssl.nextProtos",
		"central.ssl.insecureSkipVerify",
		"central.ssl.cipherSuites",
		"central.ssl.minVersion",
		"central.ssl.maxVersion",
		"central.environment",
		"central.deployment",
		"central.mode",
		"central.team",
		"central.pollInterval",
		"central.proxyUrl",
		"central.apiServerVersion",
		"central.additionalTags",
	}
	return keys[confKey]
}

// AddCentralConfigProperties - Adds the command properties needed for Central Config
func AddCentralConfigProperties(props properties.Properties, agentType AgentType) {
	props.AddStringProperty(fmt.Sprint(pathTenantID), "", "Tenant ID for the owner of the environment")
	props.AddStringProperty(fmt.Sprint(pathURL), "https://apicentral.axway.com", "URL of AMPLIFY Central")
	props.AddStringProperty(fmt.Sprint(pathPlatformURL), "https://platform.axway.com", "URL of the platform")
	props.AddStringProperty(fmt.Sprint(pathAuthPrivateKey), "/etc/private_key.pem", "Path to the private key for AMPLIFY Central Authentication")
	props.AddStringProperty(fmt.Sprint(pathAuthPublicKey), "/etc/public_key", "Path to the public key for AMPLIFY Central Authentication")
	props.AddStringProperty(fmt.Sprint(pathAuthKeyPassword), "", "Password for the private key, if needed")
	props.AddStringProperty(fmt.Sprint(pathAuthURL), "https://login.axway.com/auth", "AMPLIFY Central authentication URL")
	props.AddStringProperty(fmt.Sprint(pathAuthRealm), "Broker", "AMPLIFY Central authentication Realm")
	props.AddStringProperty(fmt.Sprint(pathAuthClientID), "", "Client ID for the service account")
	props.AddDurationProperty(fmt.Sprint(pathAuthTimeout), 10*time.Second, "Timeout waiting for AxwayID response")
	// ssl properties and command flags
	props.AddStringSliceProperty(fmt.Sprint(pathSSLNextProtos), []string{}, "List of supported application level protocols, comma separated")
	props.AddBoolProperty(fmt.Sprint(pathSSLInsecureSkipVerify), false, "Controls whether a client verifies the server's certificate chain and host name")
	props.AddStringSliceProperty(fmt.Sprint(pathSSLCipherSuites), TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	props.AddStringProperty(fmt.Sprint(pathSSLMinVersion), TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	props.AddStringProperty(fmt.Sprint(pathSSLMaxVersion), "0", "Maximum acceptable SSL/TLS protocol version")
	props.AddStringProperty(fmt.Sprint(pathEnvironment), "", "The Environment that the APIs will be associated with in AMPLIFY Central")
	props.AddStringProperty(fmt.Sprint(pathProxyURL), "", "The Proxy URL to use for communication to AMPLIFY Central")

	if agentType == TraceabilityAgent {
		props.AddStringProperty(fmt.Sprint(pathDeployment), "prod", "AMPLIFY Central")
	} else {
		props.AddStringProperty(fmt.Sprint(pathMode), "publishToEnvironmentAndCatalog", "Agent Mode")
		props.AddStringProperty(fmt.Sprint(pathTeam), "", "Team name for creating catalog")
		props.AddDurationProperty(fmt.Sprint(pathPollInterval), 60*time.Second, "The time interval at which the central will be polled for subscription processing.")
		props.AddStringProperty(fmt.Sprint(pathAPIServerVersion), "v1alpha1", "Version of the API Server")
		props.AddStringProperty(fmt.Sprint(pathAdditionalTags), "", "Additional Tags to Add to discovered APIs when publishing to AMPLIFY Central")
		AddApprovalConfigProperties(props)
	}
}

// ParseCentralConfig - Parses the Central Config values from the command line
func ParseCentralConfig(props properties.Properties, agentType AgentType) (CentralConfig, error) {
	proxyURL := props.StringPropertyValue(fmt.Sprint(pathProxyURL))
	cfg := &CentralConfiguration{
		AgentType:    agentType,
		TenantID:     props.StringPropertyValue(fmt.Sprint(pathTenantID)),
		PollInterval: props.DurationPropertyValue(fmt.Sprint(pathPollInterval)),
		Environment:  props.StringPropertyValue(fmt.Sprint(pathEnvironment)),
		Auth: &AuthConfiguration{
			URL:        props.StringPropertyValue(fmt.Sprint(pathAuthURL)),
			Realm:      props.StringPropertyValue(fmt.Sprint(pathAuthRealm)),
			ClientID:   props.StringPropertyValue(fmt.Sprint(pathAuthClientID)),
			PrivateKey: props.StringPropertyValue(fmt.Sprint(pathAuthPrivateKey)),
			PublicKey:  props.StringPropertyValue(fmt.Sprint(pathAuthPublicKey)),
			KeyPwd:     props.StringPropertyValue(fmt.Sprint(pathAuthKeyPassword)),
			Timeout:    props.DurationPropertyValue(fmt.Sprint(pathAuthTimeout)),
		},
		TLS: &TLSConfiguration{
			NextProtos:         props.StringSlicePropertyValue(fmt.Sprint(pathSSLNextProtos)),
			InsecureSkipVerify: props.BoolPropertyValue(fmt.Sprint(pathSSLInsecureSkipVerify)),
			CipherSuites:       NewCipherArray(props.StringSlicePropertyValue(fmt.Sprint(pathSSLCipherSuites))),
			MinVersion:         TLSVersionAsValue(props.StringPropertyValue(fmt.Sprint(pathSSLMinVersion))),
			MaxVersion:         TLSVersionAsValue(props.StringPropertyValue(fmt.Sprint(pathSSLMaxVersion))),
		},
		ProxyURL: proxyURL,
	}

	// Set the Proxy Environment Variable
	cfg.SetProxyEnvironmentVariable()

	if agentType == TraceabilityAgent {
		cfg.APICDeployment = props.StringPropertyValue(fmt.Sprint(pathDeployment))
	} else {
		cfg.URL = props.StringPropertyValue(fmt.Sprint(pathURL))
		cfg.PlatformURL = props.StringPropertyValue(fmt.Sprint(pathPlatformURL))
		cfg.Mode = StringAgentModeMap[strings.ToLower(props.StringPropertyValue(fmt.Sprint(pathMode)))]
		cfg.APIServerVersion = props.StringPropertyValue(fmt.Sprint(pathAPIServerVersion))
		cfg.TeamName = props.StringPropertyValue(fmt.Sprint(pathTeam))
		cfg.TagsToPublish = props.StringPropertyValue(fmt.Sprint(pathAdditionalTags))

		// set the notifications
		subscriptionConfig, err := ParseSubscriptionConfig(props)
		if err != nil {
			return nil, err
		}
		cfg.SubscriptionConfiguration = subscriptionConfig
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	LogCentralConfig(cfg)
	return cfg, nil
}

// LogCentralConfig - log debug config values
func LogCentralConfig(cfg *CentralConfiguration) {
	log.Debug("---------- Agent Config START ----------")
	log.Debug(fmt.Sprint(pathTenantID) + ": [" + cfg.TenantID + "]")
	log.Debug(fmt.Sprint(pathURL) + ": [" + cfg.URL + "]")
	log.Debug(fmt.Sprint(pathPlatformURL) + ": [" + cfg.PlatformURL + "]")
	log.Debug(fmt.Sprint(pathAuthPrivateKey) + ": [" + cfg.Auth.GetPrivateKey() + "]")
	log.Debug(fmt.Sprint(pathAuthPublicKey) + ": [" + cfg.Auth.GetPublicKey() + "]")
	log.Debug(fmt.Sprint(pathAuthKeyPassword) + ": [" + cfg.maskValue(cfg.Auth.GetKeyPassword()) + "]")
	log.Debug(fmt.Sprint(pathAuthURL) + ": [" + cfg.Auth.GetTokenURL() + "]")
	log.Debug(fmt.Sprint(pathAuthRealm) + ": [" + cfg.Auth.GetRealm() + "]")
	log.Debug(fmt.Sprint(pathAuthClientID) + ": [" + cfg.Auth.GetClientID() + "]")
	log.Debug(fmt.Sprint(pathAuthTimeout) + ": [" + cfg.Auth.GetTimeout().String() + "]")
	log.Debug(fmt.Sprint(pathSSLNextProtos) + ":[" + cfg.nextProtosArrayToString() + "]")
	log.Debug(fmt.Sprint(pathSSLInsecureSkipVerify) + ":[" + strconv.FormatBool(cfg.TLS.IsInsecureSkipVerify()) + "]")
	log.Debug(fmt.Sprint(pathSSLCipherSuites) + ":[" + cfg.cipherSuitesArrayToString() + "]")
	log.Debug(fmt.Sprint(pathSSLMinVersion) + ":[" + tlsVersionsInverse[cfg.GetTLSConfig().GetMinVersion()] + "]")
	log.Debug(fmt.Sprint(pathSSLMaxVersion) + ":[" + tlsVersionsInverse[cfg.GetTLSConfig().GetMaxVersion()] + "]")
	log.Debug(fmt.Sprint(pathEnvironment) + ": [" + cfg.Environment + "]")
	log.Debug(fmt.Sprint(pathDeployment) + ": [" + cfg.APICDeployment + "]")
	log.Debug(fmt.Sprint(pathMode) + ": [" + cfg.GetAgentModeAsString() + "]")
	log.Debug(fmt.Sprint(pathTeam) + ": [" + cfg.TeamName + "]")
	log.Debug(fmt.Sprint(pathPollInterval) + ": [" + cfg.PollInterval.String() + "]")
	log.Debug(fmt.Sprint(pathProxyURL) + ": [" + cfg.ProxyURL + "]")
	log.Debug(fmt.Sprint(pathAPIServerVersion) + ": [" + cfg.APIServerVersion + "]")
	log.Debug(fmt.Sprint(pathAdditionalTags) + ": [" + cfg.TagsToPublish + "]")
	log.Debug("---------- Agent Config FINISH ----------")
}

// maskValue - mask sensitive information with * (asterisk).  Length of sensitiveData to match returning maskedValue
func (c *CentralConfiguration) maskValue(sensitiveData string) string {
	var maskedValue string
	for i := 0; i < len(sensitiveData); i++ {
		maskedValue += "*"
	}
	return maskedValue
}

// nextProtosArrayToString - return a string of concatenated next protos
func (c *CentralConfiguration) nextProtosArrayToString() string {
	var proto string
	for _, str := range c.GetTLSConfig().GetNextProtos() {
		proto += " " + str
	}
	return proto
}

// cipherSuitesArrayToString - return a string of concatenated tls cipher suites
func (c *CentralConfiguration) cipherSuitesArrayToString() string {
	var tlsCipherSuites string
	for _, tlsCipherSuite := range c.GetTLSConfig().GetCipherSuites() {
		tlsCipherSuites += " " + tlsCipherSuite.String()
	}
	return tlsCipherSuites
}
