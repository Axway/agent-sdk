package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/gorhill/cronexpr"
)

const urlCutSet = " /"

type Region int

const (
	US Region = iota + 1
	EU
	AP
)

var regionNamesMap = map[Region]string{
	US: "US",
	EU: "EU",
	AP: "AP",
}

var nameToRegionMap = map[string]Region{
	"US": US,
	"EU": EU,
	"AP": AP,
}

func (r Region) ToString() string {
	return regionNamesMap[r]
}

type regionalSettings struct {
	SingleURL        string
	CentralURL       string
	AuthURL          string
	PlatformURL      string
	TraceabilityHost string
	Deployment       string
}

var regionalSettingsMap = map[Region]regionalSettings{
	US: {
		SingleURL:        "https://ingestion.platform.axway.com",
		CentralURL:       "https://apicentral.axway.com",
		AuthURL:          "https://login.axway.com/auth",
		PlatformURL:      "https://platform.axway.com",
		TraceabilityHost: "ingestion.datasearch.axway.com:5044",
		Deployment:       "prod",
	},
	EU: {
		SingleURL:        "https://ingestion-eu.platform.axway.com",
		CentralURL:       "https://central.eu-fr.axway.com",
		AuthURL:          "https://login.axway.com/auth",
		PlatformURL:      "https://platform.axway.com",
		TraceabilityHost: "ingestion.visibility.eu-fr.axway.com:5044",
		Deployment:       "prod-eu",
	},
	AP: {
		SingleURL:        "https://ingestion-ap-sg.platform.axway.com",
		CentralURL:       "https://central.ap-sg.axway.com",
		AuthURL:          "https://login.axway.com/auth",
		PlatformURL:      "https://platform.axway.com",
		TraceabilityHost: "ingestion.visibility.ap-sg.axway.com:5044",
		Deployment:       "prod-ap",
	},
}

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

var agentTypeNamesMap = map[AgentType]string{
	DiscoveryAgent:    "discoveryagent",
	TraceabilityAgent: "traceabilityagent",
	GovernanceAgent:   "governanceagent",
}

var agentTypeShortNamesMap = map[AgentType]string{
	DiscoveryAgent:    "da",
	TraceabilityAgent: "ta",
	GovernanceAgent:   "ga",
}

func (agentType AgentType) ToString() string {
	return agentTypeNamesMap[agentType]
}

func (agentType AgentType) ToShortString() string {
	return agentTypeShortNamesMap[agentType]
}

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
	GetRegion() Region
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
	GetTraceabilityHost() string
	GetPlatformURL() string
	GetAPIServerURL() string
	GetEnvironmentURL() string
	GetEnvironmentACLsURL() string
	GetServicesURL() string
	GetRevisionsURL() string
	GetInstancesURL() string
	DeleteServicesURL() string
	GetAPIServerAccessRequestDefinitionURL() string
	GetAPIServerSecretsURL() string
	GetAccessRequestsURL() string
	GetAccessRequestURL(string) string
	GetAccessRequestStateURL(string) string
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
	GetTagsToPublish() string
	GetProxyURL() string
	GetPollInterval() time.Duration
	GetReportActivityFrequency() time.Duration
	GetAPIValidationCronSchedule() string
	GetJobExecutionTimeout() time.Duration
	GetClientTimeout() time.Duration
	GetPageSize() int
	GetAPIServiceRevisionPattern() string
	GetAppendEnvironmentToTitle() bool
	GetUsageReportingConfig() UsageReportingConfig
	GetMetricReportingConfig() MetricReportingConfig
	IsUsingGRPC() bool
	GetGRPCHost() string
	GetGRPCPort() int
	IsGRPCInsecure() bool
	GetCacheStoragePath() string
	GetCacheStorageInterval() time.Duration
	GetSingleURL() string
	GetMigrationSettings() MigrationConfig
	GetWatchResourceFilters() []ResourceFilter
	SetWatchResourceFilters([]ResourceFilter) error
	GetCredentialConfig() CredentialConfig
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	IConfigValidator
	AgentType                 AgentType
	RegionSettings            regionalSettings
	Region                    Region                `config:"region"`
	TenantID                  string                `config:"organizationID"`
	TeamName                  string                `config:"team"`
	APICDeployment            string                `config:"deployment"`
	Environment               string                `config:"environment"`
	EnvironmentID             string                `config:"environmentID"`
	AgentName                 string                `config:"agentName"`
	URL                       string                `config:"url"`
	SingleURL                 string                `config:"platformSingleURL"`
	PlatformURL               string                `config:"platformURL"`
	APIServerVersion          string                `config:"apiServerVersion"`
	TagsToPublish             string                `config:"additionalTags"`
	AppendEnvironmentToTitle  bool                  `config:"appendEnvironmentToTitle"`
	MigrationSettings         MigrationConfig       `config:"migration"`
	Auth                      AuthConfig            `config:"auth"`
	TLS                       TLSConfig             `config:"ssl"`
	PollInterval              time.Duration         `config:"pollInterval"`
	ReportActivityFrequency   time.Duration         `config:"reportActivityFrequency"`
	ClientTimeout             time.Duration         `config:"clientTimeout"`
	PageSize                  int                   `config:"pageSize"`
	APIValidationCronSchedule string                `config:"apiValidationCronSchedule"`
	APIServiceRevisionPattern string                `config:"apiServiceRevisionPattern"`
	ProxyURL                  string                `config:"proxyUrl"`
	UsageReporting            UsageReportingConfig  `config:"usageReporting"`
	MetricReporting           MetricReportingConfig `config:"metricReporting"`
	GRPCCfg                   GRPCConfig            `config:"grpc"`
	CacheStoragePath          string                `config:"cacheStoragePath"`
	CacheStorageInterval      time.Duration         `config:"cacheStorageInterval"`
	CredentialConfig          CredentialConfig      `config:"credential"`
	JobExecutionTimeout       time.Duration
	environmentID             string
	teamID                    string
	isSingleURLSet            bool
	isRegionSet               bool
	isAxwayManaged            bool
	WatchResourceFilters      []ResourceFilter
}

// GRPCConfig - Represents the grpc config
type GRPCConfig struct {
	Enabled  bool   `config:"enabled"`
	Host     string `config:"host"`
	Port     int    `config:"port"`
	Insecure bool   `config:"insecure"`
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	platformURL := "https://platform.axway.com"
	return &CentralConfiguration{
		AgentType:                 agentType,
		Region:                    US,
		TeamName:                  "",
		APIServerVersion:          "v1alpha1",
		Auth:                      newAuthConfig(),
		TLS:                       NewTLSConfig(),
		PollInterval:              60 * time.Second,
		ClientTimeout:             60 * time.Second,
		PageSize:                  100,
		PlatformURL:               platformURL,
		SingleURL:                 "",
		AppendEnvironmentToTitle:  true,
		ReportActivityFrequency:   5 * time.Minute,
		APIValidationCronSchedule: "@daily",
		UsageReporting:            NewUsageReporting(platformURL),
		MetricReporting:           NewMetricReporting(),
		JobExecutionTimeout:       5 * time.Minute,
		CacheStorageInterval:      10 * time.Second,
		GRPCCfg: GRPCConfig{
			Enabled: true,
		},
		MigrationSettings: newMigrationConfig(),
		CredentialConfig:  newCredentialConfig(),
	}
}

// NewTestCentralConfig - Creates the default central config
func NewTestCentralConfig(agentType AgentType) CentralConfig {
	config := NewCentralConfig(agentType).(*CentralConfiguration)
	config.TenantID = "1234567890"
	config.Region = US
	config.URL = "https://central.com"
	config.PlatformURL = "https://platform.axway.com"
	config.Environment = "environment"
	config.environmentID = "env-id"
	config.Auth = newTestAuthConfig()
	config.MigrationSettings = newTestMigrationConfig()
	if agentType == TraceabilityAgent {
		config.APICDeployment = "deployment"
	}
	return config
}

// GetPlatformURL - Returns the central base URL
func (c *CentralConfiguration) GetPlatformURL() string {
	if c.PlatformURL == "" {
		return c.RegionSettings.PlatformURL
	}
	return c.PlatformURL
}

// GetAgentType - Returns the agent type
func (c *CentralConfiguration) GetAgentType() AgentType {
	return c.AgentType
}

// GetRegion - Returns the region
func (c *CentralConfiguration) GetRegion() Region {
	return c.Region
}

// GetTenantID - Returns the tenant ID
func (c *CentralConfiguration) GetTenantID() string {
	return c.TenantID
}

// GetAPICDeployment - Returns the Central deployment type 'prod', 'preprod', team ('beano')
func (c *CentralConfiguration) GetAPICDeployment() string {
	if c.APICDeployment == "" {
		return c.RegionSettings.Deployment
	}
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
	if c.URL == "" {
		return c.RegionSettings.CentralURL
	}
	return c.URL
}

// GetTraceabilityHost - Returns the central traceability host
func (c *CentralConfiguration) GetTraceabilityHost() string {
	if c.isRegionSet {
		return c.RegionSettings.TraceabilityHost
	}
	return ""
}

// GetProxyURL - Returns the central Proxy URL
func (c *CentralConfiguration) GetProxyURL() string {
	return c.ProxyURL
}

// GetAccessRequestsURL - Returns the accessrequest URL for access request API
func (c *CentralConfiguration) GetAccessRequestsURL() string {
	return c.GetEnvironmentURL() + "/accessrequests"
}

// GetAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) GetAPIServerURL() string {
	return c.GetURL() + "/apis/management/" + c.APIServerVersion + "/environments/"
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

// GetAPIServerAccessRequestDefinitionURL - Returns the APIServer URL for access request definitions
func (c *CentralConfiguration) GetAPIServerAccessRequestDefinitionURL() string {
	return c.GetEnvironmentURL() + "/accessrequestdefinitions"
}

// GetAPIServerSecretsURL - Returns the APIServer URL for secrets
func (c *CentralConfiguration) GetAPIServerSecretsURL() string {
	return c.GetEnvironmentURL() + "/secrets"
}

// GetAccessRequestURL - Returns the access request URL for catalog item subscription states
func (c *CentralConfiguration) GetAccessRequestURL(accessRequestName string) string {
	return fmt.Sprintf("%s/%s", c.GetAccessRequestsURL(), accessRequestName)
}

// GetAccessRequestStateURL - Returns the access request URL to update the state
func (c *CentralConfiguration) GetAccessRequestStateURL(accessRequestName string) string {
	return fmt.Sprintf("%s/state", c.GetAccessRequestURL(accessRequestName))
}

// GetAuthConfig - Returns the Auth Config
func (c *CentralConfiguration) GetAuthConfig() AuthConfig {
	return c.Auth
}

// GetMigrationSettings - Returns the Migration Config
func (c *CentralConfiguration) GetMigrationSettings() MigrationConfig {
	return c.MigrationSettings
}

// GetTLSConfig - Returns the TLS Config
func (c *CentralConfiguration) GetTLSConfig() TLSConfig {
	return c.TLS
}

// GetTagsToPublish - Returns tags to publish
func (c *CentralConfiguration) GetTagsToPublish() string {
	return c.TagsToPublish
}

// GetPollInterval - Returns the interval for polling subscriptions
func (c *CentralConfiguration) GetPollInterval() time.Duration {
	return c.PollInterval
}

// GetReportActivityFrequency - Returns the interval between running periodic status updater
func (c *CentralConfiguration) GetReportActivityFrequency() time.Duration {
	return c.ReportActivityFrequency
}

// GetAPIValidationCronSchedule - Returns the cron schedule running the api validator
func (c *CentralConfiguration) GetAPIValidationCronSchedule() string {
	return c.APIValidationCronSchedule
}

// GetJobExecutionTimeout - Returns the max time a job execution can run before considered failed
func (c *CentralConfiguration) GetJobExecutionTimeout() time.Duration {
	return c.JobExecutionTimeout
}

// GetClientTimeout - Returns the interval for http client timeouts
func (c *CentralConfiguration) GetClientTimeout() time.Duration {
	return c.ClientTimeout
}

// GetPageSize - Returns the page size for api server calls
func (c *CentralConfiguration) GetPageSize() int {
	return c.PageSize
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
	// Some paths in DA are checking usage reporting .  So return an empty usage reporting config if nil
	// Find All References to see DA scenarios checking for this config
	if c.UsageReporting == nil {
		return NewUsageReporting(c.GetPlatformURL())
	}
	return c.UsageReporting
}

// GetMetricReportingConfig -
func (c *CentralConfiguration) GetMetricReportingConfig() MetricReportingConfig {
	// Some paths in DA are checking usage reporting .  So return an empty usage reporting config if nil
	// Find All References to see DA scenarios checking for this config
	if c.MetricReporting == nil {
		return NewMetricReporting()
	}
	return c.MetricReporting
}

// GetCredentialConfig -
func (c *CentralConfiguration) GetCredentialConfig() CredentialConfig {
	return c.CredentialConfig
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
	if c.SingleURL == "" && !c.isSingleURLSet {
		if c.isRegionSet {
			return c.RegionSettings.SingleURL
		}

	}
	return c.SingleURL
}

// GetWatchResourceFilters - returns the custom watch filter config
func (c *CentralConfiguration) GetWatchResourceFilters() []ResourceFilter {
	if c.WatchResourceFilters == nil {
		c.WatchResourceFilters = make([]ResourceFilter, 0)
	}
	return c.WatchResourceFilters
}

// SetWatchResourceFilters - sets the custom watch filter config
func (c *CentralConfiguration) SetWatchResourceFilters(filters []ResourceFilter) error {
	c.WatchResourceFilters = make([]ResourceFilter, 0)
	for _, filter := range filters {
		if filter.Group == "" || filter.Kind == "" {
			return errors.New("invalid watch filter configuration, group and kind are required")
		}

		if filter.Name == "" {
			filter.Name = "*"
		}
		if len(filter.EventTypes) == 0 {
			filter.EventTypes = []ResourceEventType{ResourceEventCreated, ResourceEventUpdated, ResourceEventDeleted}
		}

		if filter.Scope == nil {
			filter.Scope = &ResourceScope{
				Kind: mv1.EnvironmentGVK().Kind,
				Name: c.GetEnvironmentName(),
			}
		} else {
			if filter.Scope.Kind == "" || filter.Scope.Name == "" {
				return errors.New("invalid watch filter configuration, scope kind and name are required")
			}
		}

		c.WatchResourceFilters = append(c.WatchResourceFilters, filter)
	}

	return nil
}

const (
	pathRegion                    = "central.region"
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
	pathPageSize                  = "central.pageSize"
	pathAPIServiceRevisionPattern = "central.apiServiceRevisionPattern"
	pathProxyURL                  = "central.proxyUrl"
	pathAPIServerVersion          = "central.apiServerVersion"
	pathAdditionalTags            = "central.additionalTags"
	pathAppendEnvironmentToTitle  = "central.appendEnvironmentToTitle"
	pathAPIValidationCronSchedule = "central.apiValidationCronSchedule"
	pathJobTimeout                = "central.jobTimeout"
	pathGRPCEnabled               = "central.grpc.enabled"
	pathGRPCHost                  = "central.grpc.host"
	pathGRPCPort                  = "central.grpc.port"
	pathGRPCInsecure              = "central.grpc.insecure"
	pathCacheStoragePath          = "central.cacheStoragePath"
	pathCacheStorageInterval      = "central.cacheStorageInterval"
	pathCredentialsOAuthMethods   = "central.credentials.oauthMethods"
)

// ValidateCfg - Validates the config, implementing IConfigInterface
func (c *CentralConfiguration) ValidateCfg() (err error) {
	exception.Block{
		Try: func() {
			if supportsTraceability(c.AgentType) && c.GetUsageReportingConfig().IsOfflineMode() {
				// only validate certain things when a traceability agent is in offline mode
				c.validateOfflineConfig()
				c.GetUsageReportingConfig().Validate()
				return
			}
			c.validateConfig()
			c.Auth.validate()

			// Check that platform service account is used with market place provisioning
			if strings.HasPrefix(c.Auth.GetClientID(), "DOSA_") {
				exception.Throw(ErrServiceAccount)
			}

			if supportsTraceability(c.AgentType) {
				c.GetMetricReportingConfig().Validate()
				c.GetUsageReportingConfig().Validate()
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

	cron, err := cronexpr.Parse(c.GetAPIValidationCronSchedule())
	if err != nil {
		exception.Throw(ErrBadConfig.FormatError(pathAPIValidationCronSchedule))
	}
	checks := 5
	nextRuns := cron.NextN(time.Now(), uint(checks))
	if len(nextRuns) != checks {
		exception.Throw(ErrBadConfig.FormatError(pathAPIValidationCronSchedule))
	}
	for i := 1; i < checks-1; i++ {
		delta := nextRuns[i].Sub(nextRuns[i-1])
		if delta < time.Hour {
			log.Tracef("%s must be at least 1 hour apart", pathAPIValidationCronSchedule)
			exception.Throw(ErrBadConfig.FormatError(pathAPIValidationCronSchedule))
		}
	}

	if c.GetClientTimeout() <= 0 {
		exception.Throw(ErrBadConfig.FormatError(pathClientTimeout))
	}
	if c.GetJobExecutionTimeout() < 0 {
		exception.Throw(ErrBadConfig.FormatError(pathJobTimeout))
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
	props.AddStringProperty(pathURL, "", "URL of Amplify Central")
	props.AddStringProperty(pathTeam, "", "Team name for creating catalog")
	props.AddStringProperty(pathPlatformURL, "", "URL of the platform")
	props.AddStringProperty(pathSingleURL, "", "Alternate Connection for Agent if using static IP")
	props.AddStringProperty(pathAuthPrivateKey, "/etc/private_key.pem", "Path to the private key for Amplify Central Authentication")
	props.AddStringProperty(pathAuthPublicKey, "/etc/public_key", "Path to the public key for Amplify Central Authentication")
	props.AddStringProperty(pathAuthKeyPassword, "", "Path to the password file required by the private key for Amplify Central Authentication")
	props.AddStringProperty(pathAuthURL, "", "Amplify Central authentication URL")
	props.AddStringProperty(pathAuthRealm, "Broker", "Amplify Central authentication Realm")
	props.AddStringProperty(pathAuthClientID, "", "Client ID for the service account")
	props.AddDurationProperty(pathAuthTimeout, 10*time.Second, "Timeout waiting for AxwayID response", properties.WithLowerLimit(10*time.Second))
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
	props.AddStringProperty(pathAPIValidationCronSchedule, "@daily", "The cron schedule at which the agent validates API Services with the dataplane")
	props.AddDurationProperty(pathClientTimeout, 60*time.Second, "The time interval at which the http client times out making HTTP requests and processing the response", properties.WithLowerLimit(15*time.Second), properties.WithUpperLimit(120*time.Second))
	props.AddIntProperty(pathPageSize, 100, "The max page size the agent will use while retrieving API Server resources", properties.WithLowerLimitInt(10), properties.WithUpperLimitInt(100))
	props.AddStringProperty(pathAPIServiceRevisionPattern, "", "The naming pattern for APIServiceRevision Title")
	props.AddStringProperty(pathAPIServerVersion, "v1alpha1", "Version of the API Server")
	props.AddDurationProperty(pathJobTimeout, 5*time.Minute, "The max time a job execution can run before being considered as failed")
	// Watch stream config
	props.AddBoolProperty(pathGRPCEnabled, true, "Controls whether an agent uses a gRPC connection")
	props.AddStringProperty(pathGRPCHost, "", "Host name for Amplify Central gRPC connection")
	props.AddIntProperty(pathGRPCPort, 0, "Port for Amplify Central gRPC connection")
	props.AddBoolProperty(pathGRPCInsecure, false, "Controls whether an agent uses a gRPC connection with TLS")
	props.AddStringProperty(pathCacheStoragePath, "", "The directory path where agent cache will be persisted to file")
	props.AddDurationProperty(pathCacheStorageInterval, 10*time.Second, "The interval to persist agent caches to file", properties.WithLowerLimit(10*time.Second))
	props.AddStringSliceProperty(pathCredentialsOAuthMethods, []string{}, "Allowed OAuth credential types")

	if supportsTraceability(agentType) {
		props.AddStringProperty(pathEnvironmentID, "", "Offline Usage Reporting Only. The Environment ID the usage is associated with on Amplify Central")
		props.AddStringProperty(pathDeployment, "", "Amplify Central")
		AddMetricReportingProperties(props)
		AddUsageReportingProperties(props)
	} else {
		props.AddStringProperty(pathAdditionalTags, "", "Additional Tags to Add to discovered APIs when publishing to Amplify Central")
		props.AddBoolProperty(pathAppendEnvironmentToTitle, true, "When true API titles and descriptions will be appended with environment name")
		AddMigrationConfigProperties(props)
	}
}

// ParseCentralConfig - Parses the Central Config values from the command line
func ParseCentralConfig(props properties.Properties, agentType AgentType) (CentralConfig, error) {
	region := US
	regionSet := false
	if r, ok := nameToRegionMap[props.StringPropertyValue(pathRegion)]; ok {
		region = r
		regionSet = true
	}

	regSet := regionalSettingsMap[region]

	// check if CENTRAL_SINGLEURL is explicitly empty
	_, set := os.LookupEnv("CENTRAL_SINGLEURL")

	var metricReporting MetricReportingConfig
	var usageReporting UsageReportingConfig
	if supportsTraceability(agentType) {
		metricReporting = ParseMetricReportingConfig(props)
		usageReporting = ParseUsageReportingConfig(props)
		if usageReporting.IsOfflineMode() {
			// Check if this is offline usage reporting only
			cfg := &CentralConfiguration{
				AgentName:       props.StringPropertyValue(pathAgentName),
				AgentType:       agentType,
				UsageReporting:  usageReporting,
				MetricReporting: metricReporting,
			}
			// only need the environment ID in offline mode
			cfg.EnvironmentID = props.StringPropertyValue(pathEnvironmentID)
			return cfg, nil
		}
	}

	proxyURL := props.StringPropertyValue(pathProxyURL)

	cfg := &CentralConfiguration{
		AgentType:                 agentType,
		RegionSettings:            regSet,
		Region:                    region,
		TenantID:                  props.StringPropertyValue(pathTenantID),
		PollInterval:              props.DurationPropertyValue(pathPollInterval),
		ReportActivityFrequency:   props.DurationPropertyValue(pathReportActivityFrequency),
		APIValidationCronSchedule: props.StringPropertyValue(pathAPIValidationCronSchedule),
		JobExecutionTimeout:       props.DurationPropertyValue(pathJobTimeout),
		ClientTimeout:             props.DurationPropertyValue(pathClientTimeout),
		PageSize:                  props.IntPropertyValue(pathPageSize),
		APIServiceRevisionPattern: props.StringPropertyValue(pathAPIServiceRevisionPattern),
		Environment:               props.StringPropertyValue(pathEnvironment),
		TeamName:                  props.StringPropertyValue(pathTeam),
		AgentName:                 props.StringPropertyValue(pathAgentName),
		Auth: &AuthConfiguration{
			RegionSettings: regSet,
			URL:            strings.TrimRight(props.StringPropertyValue(pathAuthURL), urlCutSet),
			Realm:          props.StringPropertyValue(pathAuthRealm),
			ClientID:       props.StringPropertyValue(pathAuthClientID),
			PrivateKey:     props.StringPropertyValue(pathAuthPrivateKey),
			PublicKey:      props.StringPropertyValue(pathAuthPublicKey),
			KeyPwd:         props.StringPropertyValue(pathAuthKeyPassword),
			Timeout:        props.DurationPropertyValue(pathAuthTimeout),
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
		},
		CacheStoragePath:     props.StringPropertyValue(pathCacheStoragePath),
		CacheStorageInterval: props.DurationPropertyValue(pathCacheStorageInterval),
	}
	cfg.URL = strings.TrimRight(props.StringPropertyValue(pathURL), urlCutSet)
	cfg.SingleURL = strings.TrimRight(props.StringPropertyValue(pathSingleURL), urlCutSet)
	cfg.isSingleURLSet = set
	cfg.isRegionSet = regionSet
	cfg.PlatformURL = strings.TrimRight(props.StringPropertyValue(pathPlatformURL), urlCutSet)
	cfg.APIServerVersion = props.StringPropertyValue(pathAPIServerVersion)
	cfg.APIServiceRevisionPattern = props.StringPropertyValue(pathAPIServiceRevisionPattern)
	cfg.CredentialConfig = newCredentialConfig()
	if supportsTraceability(agentType) {
		cfg.APICDeployment = props.StringPropertyValue(pathDeployment)
		cfg.UsageReporting = usageReporting
		cfg.MetricReporting = metricReporting
	} else {
		cfg.TeamName = props.StringPropertyValue(pathTeam)
		cfg.TagsToPublish = props.StringPropertyValue(pathAdditionalTags)
		cfg.AppendEnvironmentToTitle = props.BoolPropertyValue(pathAppendEnvironmentToTitle)
		cfg.MigrationSettings = ParseMigrationConfig(props)
		cfg.CredentialConfig = newCredentialConfig()
		cfg.CredentialConfig.SetAllowedOAuthMethods(props.StringSlicePropertyValue(pathCredentialsOAuthMethods))
	}
	if cfg.AgentName == "" && cfg.Environment != "" && agentType.ToShortString() != "" {
		cfg.AgentName = cfg.Environment + "-" + agentType.ToShortString()
	}
	if regionSet {
		regSet := regionalSettingsMap[region]
		cfg.RegionSettings = regSet
		authCfg, ok := cfg.Auth.(*AuthConfiguration)
		if ok {
			authCfg.RegionSettings = regSet
			authCfg.URL = regSet.AuthURL
		}

		cfg.URL = regSet.CentralURL
		cfg.PlatformURL = regSet.PlatformURL
		cfg.APICDeployment = regSet.Deployment
	}

	return cfg, nil
}

func supportsTraceability(agentType AgentType) bool {
	return agentType == TraceabilityAgent
}
