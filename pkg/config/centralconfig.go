package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
)

// AgentType - Defines the type of agent
type AgentType int

const (
	// DiscoveryAgent - Type definition for discovery agent
	DiscoveryAgent AgentType = iota + 1
	// TraceabilityAgent - Type definition for traceability agent
	TraceabilityAgent
	// GovernanceAgent - Type definition for governance agent
	GovernanceAgent
	// GenericService - Type for a generic service
	GenericService
)

// subscription approval types
const (
	ManualApproval  string = "manual"
	AutoApproval    string = "auto"
	WebhookApproval string = "webhook"
)

// AgentTypeName - Holds the name Agent type
var AgentTypeName string

// AgentDisplayName - Holds the name Agent name for displaying in version command or elsewhere
var AgentDisplayName string

// AgentVersion - Holds the version of agent
var AgentVersion string

// AgentLatestVersion - Holds the latest version of the agent
var AgentLatestVersion string

// AgentDataPlaneType - Holds the data plane type of agent
var AgentDataPlaneType string

// SDKVersion - Holds the version of SDK
var SDKVersion string

// IConfigValidator - Interface to be implemented for config validation by agent
type IConfigValidator interface {
	ValidateCfg() error
}

// IResourceConfigCallback - Interface to be implemented by configs to apply API Server resource for agent
type IResourceConfigCallback interface {
	ApplyResources(agentResource *v1.ResourceInstance) error
}

// CentralConfig - Interface to get central Config
type CentralConfig interface {
	GetAgentType() AgentType
	GetTenantID() string
	GetAPICDeployment() string
	GetEnvironmentID() string
	SetEnvironmentID(environmentID string)
	IsAxwayManaged() bool
	SetAxwayManaged(isAxwayManaged bool)
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
	GetEnvironmentACLsURL() string
	GetServicesURL() string
	GetRevisionsURL() string
	GetInstancesURL() string
	DeleteServicesURL() string
	GetConsumerInstancesURL() string
	GetAPIServerSubscriptionDefinitionURL() string
	GetAPIServerAccessRequestDefinitionURL() string
	GetAPIServerWebhooksURL() string
	GetAPIServerSecretsURL() string
	GetCategoriesURL() string
	GetSubscriptionURL() string
	GetSubscriptionConfig() SubscriptionConfig
	GetAccessRequestsURL() string
	GetAccessRequestURL(string) string
	GetAccessRequestStateURL(string) string
	GetCatalogItemSubscriptionsURL(string) string
	GetCatalogItemSubscriptionStatesURL(string, string) string
	GetCatalogItemSubscriptionPropertiesURL(string, string) string
	GetCatalogItemSubscriptionRelationshipURL(string, string) string
	GetCatalogItemSubscriptionDefinitionPropertiesURL(string) string
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
	GetTagsToPublish() string
	GetProxyURL() string
	GetPollInterval() time.Duration
	GetReportActivityFrequency() time.Duration
	GetJobExecutionTimeout() time.Duration
	GetClientTimeout() time.Duration
	GetAPIServiceRevisionPattern() string
	GetCatalogItemByIDURL(catalogItemID string) string
	GetAppendEnvironmentToTitle() bool
	GetUsageReportingConfig() UsageReportingConfig
	IsUsingGRPC() bool
	GetGRPCHost() string
	GetGRPCPort() int
	IsGRPCInsecure() bool
	IsFetchOnStartupEnabled() bool
	GetFetchOnStartupPageSize() int
	GetFetchOnStartupRetention() time.Duration
	GetCacheStoragePath() string
	GetCacheStorageInterval() time.Duration
	SetIsMarketplaceSubsEnabled(enabled bool)
	IsMarketplaceSubsEnabled() bool
	GetSingleURL() string
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	IConfigValidator
	AgentType                 AgentType
	TenantID                  string               `config:"organizationID"`
	TeamName                  string               `config:"team"`
	APICDeployment            string               `config:"deployment"`
	Environment               string               `config:"environment"`
	EnvironmentID             string               `config:"environmentID"`
	AgentName                 string               `config:"agentName"`
	URL                       string               `config:"url"`
	SingleURL                 string               `config:"platformSingleURL"`
	PlatformURL               string               `config:"platformURL"`
	APIServerVersion          string               `config:"apiServerVersion"`
	TagsToPublish             string               `config:"additionalTags"`
	AppendEnvironmentToTitle  bool                 `config:"appendEnvironmentToTitle"`
	Auth                      AuthConfig           `config:"auth"`
	TLS                       TLSConfig            `config:"ssl"`
	PollInterval              time.Duration        `config:"pollInterval"`
	ReportActivityFrequency   time.Duration        `config:"reportActivityFrequency"`
	ClientTimeout             time.Duration        `config:"clientTimeout"`
	APIServiceRevisionPattern string               `config:"apiServiceRevisionPattern"`
	ProxyURL                  string               `config:"proxyUrl"`
	SubscriptionConfiguration SubscriptionConfig   `config:"subscriptions"`
	UsageReporting            UsageReportingConfig `config:"usageReporting"`
	GRPCCfg                   GRPCConfig           `config:"grpc"`
	CacheStoragePath          string               `config:"cacheStoragePath"`
	CacheStorageInterval      time.Duration        `config:"cacheStorageInterval"`
	JobExecutionTimeout       time.Duration
	environmentID             string
	teamID                    string
	isAxwayManaged            bool
	isMarketplaceSubs         bool
}

// FetchOnStartup - Fetch on startup config
type FetchOnStartup struct {
	Enabled   bool          `config:"enabled"`
	PageSize  int           `config:"pageSize"`
	Retention time.Duration `config:"retention"`
}

// GRPCConfig - Represents the grpc config
type GRPCConfig struct {
	Enabled        bool           `config:"enabled"`
	Host           string         `config:"host"`
	Port           int            `config:"port"`
	Insecure       bool           `config:"insecure"`
	FetchOnStartup FetchOnStartup `config:"fetchOnStartup"`
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	return &CentralConfiguration{
		AgentType:                 agentType,
		TeamName:                  "",
		APIServerVersion:          "v1alpha1",
		Auth:                      newAuthConfig(),
		TLS:                       NewTLSConfig(),
		PollInterval:              60 * time.Second,
		ClientTimeout:             60 * time.Second,
		PlatformURL:               "https://platform.axway.com",
		SingleURL:                 "",
		SubscriptionConfiguration: NewSubscriptionConfig(),
		AppendEnvironmentToTitle:  true,
		ReportActivityFrequency:   5 * time.Minute,
		UsageReporting:            NewUsageReporting(),
		JobExecutionTimeout:       5 * time.Minute,
		CacheStorageInterval:      10 * time.Second,
		GRPCCfg: GRPCConfig{
			FetchOnStartup: FetchOnStartup{
				Retention: 10 * time.Minute,
				PageSize:  20,
			},
		},
	}
}

// NewTestCentralConfig - Creates the default central config
func NewTestCentralConfig(agentType AgentType) CentralConfig {
	config := NewCentralConfig(agentType).(*CentralConfiguration)
	config.TenantID = "1234567890"
	config.URL = "https://central.com"
	config.Environment = "environment"
	config.Auth = newTestAuthConfig()
	return config
}

// GetPlatformURL - Returns the central base URL
func (c *CentralConfiguration) GetPlatformURL() string {
	return c.PlatformURL
}

// GetAgentType - Returns the agent type
func (c *CentralConfiguration) GetAgentType() AgentType {
	return c.AgentType
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

// SetIsMarketplaceSubsEnabled - Sets the isMarketplaceSubs boolean
func (c *CentralConfiguration) SetIsMarketplaceSubsEnabled(enabled bool) {
	c.isMarketplaceSubs = enabled
}

// IsMarketplaceSubsEnabled - Returns the isMarketplaceSubs boolean
func (c *CentralConfiguration) IsMarketplaceSubsEnabled() bool {
	return c.isMarketplaceSubs
}

// IsAxwayManaged - Returns the environment ID
func (c *CentralConfiguration) IsAxwayManaged() bool {
	return c.isAxwayManaged
}

// SetAxwayManaged - Sets the environment ID
func (c *CentralConfiguration) SetAxwayManaged(isManaged bool) {
	c.isAxwayManaged = isManaged
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

// GetCatalogItemsURL - Returns the unifiedcatalog URL for catalog items API
func (c *CentralConfiguration) GetCatalogItemsURL() string {
	return c.URL + "/api/unifiedCatalog/v1/catalogItems"
}

// GetAccessRequestsURL - Returns the accessrequest URL for access request API
func (c *CentralConfiguration) GetAccessRequestsURL() string {
	return c.GetEnvironmentURL() + "/accessrequests"
}

// GetAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) GetAPIServerURL() string {
	return c.URL + "/apis/management/" + c.APIServerVersion + "/environments/"
}

// GetAPIServerCatalogURL - Returns the base path for the API server for catalog resources
func (c *CentralConfiguration) GetAPIServerCatalogURL() string {
	return c.URL + "/apis/catalog/" + c.APIServerVersion
}

// GetEnvironmentURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetEnvironmentURL() string {
	return c.GetAPIServerURL() + c.Environment
}

// GetEnvironmentACLsURL - Returns the APIServer URL for ACLs in Environments
func (c *CentralConfiguration) GetEnvironmentACLsURL() string {
	return c.GetEnvironmentURL() + "/accesscontrollists"
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

// GetAPIServerAccessRequestDefinitionURL - Returns the APIServer URL for access request definitions
func (c *CentralConfiguration) GetAPIServerAccessRequestDefinitionURL() string {
	return c.GetEnvironmentURL() + "/accessrequestdefinitions"
}

// GetCategoriesURL - Returns the Categories URL
func (c *CentralConfiguration) GetCategoriesURL() string {
	return c.GetAPIServerCatalogURL() + "/categories"
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

// GetAccessRequestURL - Returns the access request URL for catalog item subscription states
func (c *CentralConfiguration) GetAccessRequestURL(accessRequestName string) string {
	return fmt.Sprintf("%s/%s", c.GetAccessRequestsURL(), accessRequestName)
}

// GetAccessRequestStateURL - Returns the access request URL to update the state
func (c *CentralConfiguration) GetAccessRequestStateURL(accessRequestName string) string {
	return fmt.Sprintf("%s/state", c.GetAccessRequestURL(accessRequestName))
}

// GetCatalogItemSubscriptionPropertiesURL - Returns the unifiedcatalog URL for catalog item subscription properties
func (c *CentralConfiguration) GetCatalogItemSubscriptionPropertiesURL(catalogItemID, subscriptionID string) string {
	return fmt.Sprintf("%s/%s/properties", c.GetCatalogItemSubscriptionsURL(catalogItemID), subscriptionID)
}

// GetAccessRequestSubscriptionPropertiesURL - Returns the access request URL for subscription properties
func (c *CentralConfiguration) GetAccessRequestSubscriptionPropertiesURL(accessRequestName string) string {
	return fmt.Sprintf("%s/%s", c.GetAccessRequestsURL(), accessRequestName)
}

// GetCatalogItemSubscriptionRelationshipURL - Returns the relationships URL for catalog item subscription
func (c *CentralConfiguration) GetCatalogItemSubscriptionRelationshipURL(catalogItemID, subscriptionID string) string {
	return fmt.Sprintf("%s/%s/relationships", c.GetCatalogItemSubscriptionsURL(catalogItemID), subscriptionID)
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

// GetReportActivityFrequency - Returns the interval between running periodic status updater
func (c *CentralConfiguration) GetReportActivityFrequency() time.Duration {
	return c.ReportActivityFrequency
}

// GetJobExecutionTimeout - Returns the max time a job execution can run before considered failed
func (c *CentralConfiguration) GetJobExecutionTimeout() time.Duration {
	return c.JobExecutionTimeout
}

// GetClientTimeout - Returns the interval for http client timeouts
func (c *CentralConfiguration) GetClientTimeout() time.Duration {
	return c.ClientTimeout
}

// GetAPIServiceRevisionPattern - Returns the naming pattern for APIServiceRevition title
func (c *CentralConfiguration) GetAPIServiceRevisionPattern() string {
	return c.APIServiceRevisionPattern
}

// GetAppendEnvironmentToTitle - Returns the value of append environment name to title attribute
func (c *CentralConfiguration) GetAppendEnvironmentToTitle() bool {
	return c.AppendEnvironmentToTitle
}

// GetUsageReportingConfig -
func (c *CentralConfiguration) GetUsageReportingConfig() UsageReportingConfig {
	return c.UsageReporting
}

// IsUsingGRPC -
func (c *CentralConfiguration) IsUsingGRPC() bool {
	return c.GRPCCfg.Enabled
}

// GetGRPCHost -
func (c *CentralConfiguration) GetGRPCHost() string {
	return c.GRPCCfg.Host
}

// GetGRPCPort -
func (c *CentralConfiguration) GetGRPCPort() int {
	return c.GRPCCfg.Port
}

// IsGRPCInsecure -
func (c *CentralConfiguration) IsGRPCInsecure() bool {
	return c.GRPCCfg.Insecure
}

// IsFetchOnStartupEnabled -
func (c *CentralConfiguration) IsFetchOnStartupEnabled() bool {
	return c.GRPCCfg.FetchOnStartup.Enabled
}

// GetFetchOnStartupPageSize -
func (c *CentralConfiguration) GetFetchOnStartupPageSize() int {
	return c.GRPCCfg.FetchOnStartup.PageSize
}

// GetFetchOnStartupRetention -
func (c *CentralConfiguration) GetFetchOnStartupRetention() time.Duration {
	return c.GRPCCfg.FetchOnStartup.Retention
}

// GetCacheStoragePath -
func (c *CentralConfiguration) GetCacheStoragePath() string {
	return c.CacheStoragePath
}

// GetCacheStorageInterval -
func (c *CentralConfiguration) GetCacheStorageInterval() time.Duration {
	return c.CacheStorageInterval
}

// GetSingleURL - Returns the Alternate base URL
func (c *CentralConfiguration) GetSingleURL() string {
	return c.SingleURL
}

const (
	pathTenantID                  = "central.organizationID"
	pathURL                       = "central.url"
	pathPlatformURL               = "central.platformURL"
	pathAuthPrivateKey            = "central.auth.privateKey"
	pathAuthPublicKey             = "central.auth.publicKey"
	pathAuthKeyPassword           = "central.auth.keyPassword"
	pathAuthURL                   = "central.auth.url"
	pathSingleURL                 = "central.singleURL"
	pathAuthRealm                 = "central.auth.realm"
	pathAuthClientID              = "central.auth.clientId"
	pathAuthTimeout               = "central.auth.timeout"
	pathSSLNextProtos             = "central.ssl.nextProtos"
	pathSSLInsecureSkipVerify     = "central.ssl.insecureSkipVerify"
	pathSSLCipherSuites           = "central.ssl.cipherSuites"
	pathSSLMinVersion             = "central.ssl.minVersion"
	pathSSLMaxVersion             = "central.ssl.maxVersion"
	pathEnvironment               = "central.environment"
	pathEnvironmentID             = "central.environmentID"
	pathAgentName                 = "central.agentName"
	pathDeployment                = "central.deployment"
	pathMode                      = "central.mode"
	pathTeam                      = "central.team"
	pathPollInterval              = "central.pollInterval"
	pathReportActivityFrequency   = "central.reportActivityFrequency"
	pathClientTimeout             = "central.clientTimeout"
	pathAPIServiceRevisionPattern = "central.apiServiceRevisionPattern"
	pathProxyURL                  = "central.proxyUrl"
	pathAPIServerVersion          = "central.apiServerVersion"
	pathAdditionalTags            = "central.additionalTags"
	pathAppendEnvironmentToTitle  = "central.appendEnvironmentToTitle"
	pathJobTimeout                = "central.jobTimeout"
	pathGRPCEnabled               = "central.grpc.enabled"
	pathGRPCHost                  = "central.grpc.host"
	pathGRPCPort                  = "central.grpc.port"
	pathGRPCInsecure              = "central.grpc.insecure"
	pathFetchOnStartupEnable      = "central.grpc.fetchOnStartup.enabled"
	pathFetchOnStartupPageSize    = "central.grpc.fetchOnStartup.pageSize"
	pathFetchOnStartupRetention   = "central.grpc.fetchOnStartup.retention"
	pathCacheStoragePath          = "central.cacheStoragePath"
	pathCacheStorageInterval      = "central.cacheStorageInterval"
)

// ValidateCfg - Validates the config, implementing IConfigInterface
func (c *CentralConfiguration) ValidateCfg() (err error) {
	exception.Block{
		Try: func() {
			if supportsTraceability(c.AgentType) && c.UsageReporting.IsOfflineMode() {
				// only validate certain things when a traceability agent is in offline mode
				c.validateOfflineConfig()
				c.UsageReporting.validate()
				return
			}
			c.validateConfig()
			c.Auth.validate()

			// Check that platform service account is used with market place provisioning
			if c.IsMarketplaceSubsEnabled() && strings.HasPrefix(c.Auth.GetClientID(), "DOSA_") {
				exception.Throw(ErrServiceAccount)
			}

			if supportsTraceability(c.AgentType) {
				c.UsageReporting.validate()
			}
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (c *CentralConfiguration) validateConfig() {
	if c.GetTenantID() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathTenantID))
	}

	c.validateURL(c.GetURL(), pathURL, true)

	c.validateURL(c.GetPlatformURL(), pathPlatformURL, true)

	if c.GetSingleURL() != "" {
		c.validateURL(c.GetSingleURL(), pathSingleURL, true)
	}

	// proxyURL
	c.validateURL(c.GetProxyURL(), pathProxyURL, false)

	if supportsTraceability(c.AgentType) {
		c.validateTraceabilityAgentConfig()
	} else {
		c.validateEnvironmentConfig()
		c.validateDiscoveryAgentConfig()
	}

	if c.GetReportActivityFrequency() <= 0 {
		exception.Throw(ErrBadConfig.FormatError(pathReportActivityFrequency))
	}
	if c.GetClientTimeout() <= 0 {
		exception.Throw(ErrBadConfig.FormatError(pathClientTimeout))
	}
	if c.GetJobExecutionTimeout() < 0 {
		exception.Throw(ErrBadConfig.FormatError(pathJobTimeout))
	}

	c.validateGRPC()

}

func (c *CentralConfiguration) validateGRPC() {

	if c.IsFetchOnStartupEnabled() {
		if c.GetFetchOnStartupPageSize() <= 0 {
			exception.Throw(ErrBadConfig.FormatError(pathFetchOnStartupPageSize))
		}

		if c.GetFetchOnStartupRetention() <= 0 {
			exception.Throw(ErrBadConfig.FormatError(pathFetchOnStartupRetention))
		}
	}
}

func (c *CentralConfiguration) validateURL(urlString, configPath string, isURLRequired bool) {
	if isURLRequired && urlString == "" {
		exception.Throw(ErrBadConfig.FormatError(configPath))
	}
	if urlString != "" {
		if _, err := url.ParseRequestURI(urlString); err != nil {
			exception.Throw(ErrBadConfig.FormatError(configPath))
		}
	}
}

func (c *CentralConfiguration) validateDiscoveryAgentConfig() {
	if c.GetPollInterval() <= 0 {
		exception.Throw(ErrBadConfig.FormatError(pathPollInterval))
	}
}

func (c *CentralConfiguration) validateEnvironmentConfig() {
	if c.GetEnvironmentName() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathEnvironment))
	}

	if c.APIServerVersion == "" {
		exception.Throw(ErrBadConfig.FormatError(pathAPIServerVersion))
	}
}
func (c *CentralConfiguration) validateTraceabilityAgentConfig() {
	if c.GetAPICDeployment() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathDeployment))
	}
	if c.GetEnvironmentName() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathEnvironment))
	}
}

func (c *CentralConfiguration) validateOfflineConfig() {
	// validate environment ID
	c.SetEnvironmentID(c.EnvironmentID)
	if c.GetEnvironmentID() == "" {
		exception.Throw(ErrBadConfig.FormatError(pathEnvironmentID))
	}
}

// AddCentralConfigProperties - Adds the command properties needed for Central Config
func AddCentralConfigProperties(props properties.Properties, agentType AgentType) {
	props.AddStringProperty(pathTenantID, "", "Tenant ID for the owner of the environment")
	props.AddStringProperty(pathURL, "https://apicentral.axway.com", "URL of Amplify Central")
	props.AddStringProperty(pathTeam, "", "Team name for creating catalog")
	props.AddStringProperty(pathPlatformURL, "https://platform.axway.com", "URL of the platform")
	props.AddStringProperty(pathSingleURL, "", "Alternate Connection for Agent if using static IP")
	props.AddStringProperty(pathAuthPrivateKey, "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	props.AddStringProperty(pathAuthPublicKey, "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	props.AddStringProperty(pathAuthKeyPassword, "", "Password for the private key, if needed")
	props.AddStringProperty(pathAuthURL, "https://login.axway.com/auth", "Amplify Central authentication URL")
	props.AddStringProperty(pathAuthRealm, "Broker", "Amplify Central authentication Realm")
	props.AddStringProperty(pathAuthClientID, "", "Client ID for the service account")
	props.AddDurationProperty(pathAuthTimeout, 10*time.Second, "Timeout waiting for AxwayID response")
	// ssl properties and command flags
	props.AddStringSliceProperty(pathSSLNextProtos, []string{}, "List of supported application level protocols, comma separated")
	props.AddBoolProperty(pathSSLInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name")
	props.AddStringSliceProperty(pathSSLCipherSuites, TLSDefaultCipherSuitesStringSlice(), "List of supported cipher suites, comma separated")
	props.AddStringProperty(pathSSLMinVersion, TLSDefaultMinVersionString(), "Minimum acceptable SSL/TLS protocol version")
	props.AddStringProperty(pathSSLMaxVersion, "0", "Maximum acceptable SSL/TLS protocol version")
	props.AddStringProperty(pathEnvironment, "", "The Environment that the APIs will be associated with in Amplify Central")
	props.AddStringProperty(pathAgentName, "", "The name of the asociated agent resource in Amplify Central")
	props.AddStringProperty(pathProxyURL, "", "The Proxy URL to use for communication to Amplify Central")
	props.AddDurationProperty(pathPollInterval, 60*time.Second, "The time interval at which the central will be polled for subscription processing")
	props.AddDurationProperty(pathReportActivityFrequency, 5*time.Minute, "The time interval at which the agent polls for event changes for the periodic agent status updater")
	props.AddDurationProperty(pathClientTimeout, 60*time.Second, "The time interval at which the http client times out making HTTP requests and processing the response")
	props.AddStringProperty(pathAPIServiceRevisionPattern, "", "The naming pattern for APIServiceRevision Title")
	props.AddStringProperty(pathAPIServerVersion, "v1alpha1", "Version of the API Server")
	props.AddDurationProperty(pathJobTimeout, 5*time.Minute, "The max time a job execution can run before being considered as failed")
	// Watch stream config
	props.AddBoolProperty(pathGRPCEnabled, false, "Controls whether an agent uses a gRPC connection")
	props.AddStringProperty(pathGRPCHost, "", "Host name for Amplify Central gRPC connection")
	props.AddIntProperty(pathGRPCPort, 0, "Port for Amplify Central gRPC connection")
	props.AddBoolProperty(pathGRPCInsecure, false, "Controls whether an agent uses a gRPC connection with TLS")
	props.AddBoolProperty(pathFetchOnStartupEnable, false, "Enable fetching all watched resources on startup")
	props.AddIntProperty(pathFetchOnStartupPageSize, 20, "When fetching watched resources on startup, how many resource can be returned at once")
	props.AddDurationProperty(pathFetchOnStartupRetention, 10*time.Minute, "When fetching watched resources on startup, how much time shall they be hold in the cache before evicted")
	props.AddStringProperty(pathCacheStoragePath, "", "The directory path where agent cache will be persisted to file")
	props.AddDurationProperty(pathCacheStorageInterval, 10*time.Second, "The interval to persist agent caches to file")

	if supportsTraceability(agentType) {
		props.AddStringProperty(pathEnvironmentID, "", "Offline Usage Reporting Only. The Environment ID the usage is associated with on Amplify Central")
		props.AddStringProperty(pathDeployment, "prod", "Amplify Central")
		AddUsageReportingProperties(props)
	} else {
		props.AddStringProperty(pathAdditionalTags, "", "Additional Tags to Add to discovered APIs when publishing to Amplify Central")
		props.AddBoolProperty(pathAppendEnvironmentToTitle, true, "When true API titles and descriptions will be appended with environment name")
		AddSubscriptionConfigProperties(props)
	}
}

// ParseCentralConfig - Parses the Central Config values from the command line
func ParseCentralConfig(props properties.Properties, agentType AgentType) (CentralConfig, error) {
	if supportsTraceability(agentType) {
		// Check if this is offline usage reporting only
		cfg := &CentralConfiguration{
			AgentName: props.StringPropertyValue(pathAgentName),
			AgentType: agentType,
		}
		cfg.UsageReporting = ParseUsageReportingConfig(props)
		if cfg.UsageReporting.IsOfflineMode() {
			// only need the environment ID in offline mode
			cfg.EnvironmentID = props.StringPropertyValue(pathEnvironmentID)
			return cfg, nil
		}
	}

	proxyURL := props.StringPropertyValue(pathProxyURL)
	cfg := &CentralConfiguration{
		AgentType:                 agentType,
		TenantID:                  props.StringPropertyValue(pathTenantID),
		PollInterval:              props.DurationPropertyValue(pathPollInterval),
		ReportActivityFrequency:   props.DurationPropertyValue(pathReportActivityFrequency),
		JobExecutionTimeout:       props.DurationPropertyValue(pathJobTimeout),
		ClientTimeout:             props.DurationPropertyValue(pathClientTimeout),
		APIServiceRevisionPattern: props.StringPropertyValue(pathAPIServiceRevisionPattern),
		Environment:               props.StringPropertyValue(pathEnvironment),
		TeamName:                  props.StringPropertyValue(pathTeam),
		AgentName:                 props.StringPropertyValue(pathAgentName),
		UsageReporting:            ParseUsageReportingConfig(props),
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
		GRPCCfg: GRPCConfig{
			Enabled:  props.BoolPropertyValue(pathGRPCEnabled),
			Host:     props.StringPropertyValue(pathGRPCHost),
			Port:     props.IntPropertyValue(pathGRPCPort),
			Insecure: props.BoolPropertyValue(pathGRPCInsecure),
			FetchOnStartup: FetchOnStartup{
				Enabled:   props.BoolPropertyValue(pathFetchOnStartupEnable),
				PageSize:  props.IntPropertyValue(pathFetchOnStartupPageSize),
				Retention: props.DurationPropertyValue(pathFetchOnStartupRetention),
			},
		},
		CacheStoragePath:     props.StringPropertyValue(pathCacheStoragePath),
		CacheStorageInterval: props.DurationPropertyValue(pathCacheStorageInterval),
	}

	cfg.URL = props.StringPropertyValue(pathURL)
	cfg.SingleURL = props.StringPropertyValue(pathSingleURL)
	cfg.PlatformURL = props.StringPropertyValue(pathPlatformURL)
	cfg.APIServerVersion = props.StringPropertyValue(pathAPIServerVersion)
	cfg.APIServiceRevisionPattern = props.StringPropertyValue(pathAPIServiceRevisionPattern)

	if supportsTraceability(agentType) {
		cfg.APICDeployment = props.StringPropertyValue(pathDeployment)
	} else {
		cfg.TeamName = props.StringPropertyValue(pathTeam)
		cfg.TagsToPublish = props.StringPropertyValue(pathAdditionalTags)
		cfg.AppendEnvironmentToTitle = props.BoolPropertyValue(pathAppendEnvironmentToTitle)

		// set the notifications
		subscriptionConfig := ParseSubscriptionConfig(props)
		cfg.SubscriptionConfiguration = subscriptionConfig
	}

	return cfg, nil
}

func supportsTraceability(agentType AgentType) bool {
	return agentType == TraceabilityAgent
}
