package config

import (
	"errors"
	"time"

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
	// Disconnected - Mode definition for disconnected mode
	Disconnected AgentMode = iota + 1
	// Connected - Mode definition for connected mode
	Connected
)

// AgentModeStringMap - Map the Agent Mode constant to a string
var AgentModeStringMap = map[AgentMode]string{
	Connected:    "connected",
	Disconnected: "disconnected",
}

// StringAgentModeMap - Map the string to the Agent Mode constant
var StringAgentModeMap = map[string]AgentMode{
	"connected":    Connected,
	"disconnected": Disconnected,
}

// CentralConfig - Interface to get central Config
type CentralConfig interface {
	GetAgentType() AgentType
	GetAgentMode() AgentMode
	GetAgentModeString() string
	GetTenantID() string
	GetAPICDeployment() string
	GetEnvironmentID() string
	GetEnvironmentName() string
	GetTeamID() string
	GetURL() string
	GetCatalogItemsURL() string
	GetCatalogItemImageURL(catalogItemID string) string
	GetAPIServerURL() string
	GetAPIServerServicesURL() string
	GetAPIServerServicesRevisionsURL() string
	GetAPIServerServicesInstancesURL() string
	DeleteAPIServerServicesURL() string
	GetAPIServerConsumerInstancesURL() string
	GetAPIServerSubscriptionDefinitionURL() string
	GetSubscriptionURL() string
	Validate() error
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
	GetTagsToPublish() string
	GetProxyURL() string
	GetPollInterval() time.Duration
	UpdateCatalogItemRevisions(catalogItemID string) string
	GetCatalogItemByID(catalogItemID string) string
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	AgentType            AgentType
	Mode                 AgentMode     `config:"mode"`
	TenantID             string        `config:"tenantID"`
	TeamID               string        `config:"teamID" `
	APICDeployment       string        `config:"deployment"`
	APIServerEnvironment string        `config:"apiServerEnvironment"`
	EnvironmentID        string        `config:"environmentID"`
	URL                  string        `config:"url"`
	APIServerVersion     string        `config:"apiServerVersion"`
	TagsToPublish        string        `config:"additionalTags"`
	Auth                 AuthConfig    `config:"auth"`
	TLS                  TLSConfig     `config:"ssl"`
	PollInterval         time.Duration `config:"pollInterval"`
	ProxyURL             string        `config:"proxyUrl"`
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	return &CentralConfiguration{
		AgentType:        agentType,
		Mode:             Disconnected,
		APIServerVersion: "v1alpha1",
		Auth:             newAuthConfig(),
		TLS:              NewTLSConfig(),
		PollInterval:     60 * time.Second,
	}
}

// GetAgentType - Returns the agent type
func (c *CentralConfiguration) GetAgentType() AgentType {
	return c.AgentType
}

// GetAgentMode - Returns the agent mode
func (c *CentralConfiguration) GetAgentMode() AgentMode {
	return c.Mode
}

// GetAgentModeString - Returns the agent mode
func (c *CentralConfiguration) GetAgentModeString() string {
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
	return c.EnvironmentID
}

// GetEnvironmentName - Returns the environment name
func (c *CentralConfiguration) GetEnvironmentName() string {
	return c.APIServerEnvironment
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

// GetCatalogItemsURL - Returns the URL for catalog items API
func (c *CentralConfiguration) GetCatalogItemsURL() string {
	return c.URL + baseCatalogItemsURL
}

// GetCatalogItemImageURL - Returns the image based on catalogItemID
func (c *CentralConfiguration) GetCatalogItemImageURL(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID + "/image"
}

// GetAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) GetAPIServerURL() string {
	return c.URL + "/apis/management/" + c.APIServerVersion + "/environments/"
}

// GetAPIServerServicesURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetAPIServerServicesURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/apiservices"
}

// GetAPIServerServicesRevisionsURL - Returns the APIServer URL for services API revisions
func (c *CentralConfiguration) GetAPIServerServicesRevisionsURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/apiservicerevisions"
}

// GetAPIServerServicesInstancesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerServicesInstancesURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/apiserviceinstances"
}

// DeleteAPIServerServicesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) DeleteAPIServerServicesURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/apiservices"
}

// GetAPIServerConsumerInstancesURL - Returns the APIServer URL for services API consumer instance representing the catalog item
func (c *CentralConfiguration) GetAPIServerConsumerInstancesURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/consumerinstances"
}

// GetAPIServerSubscriptionDefinitionURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerSubscriptionDefinitionURL() string {
	return c.GetAPIServerURL() + c.APIServerEnvironment + "/consumersubscriptiondefs"
}

// GetSubscriptionURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetSubscriptionURL() string {
	return c.URL + "/api/unifiedCatalog/v1/subscriptions"
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

// UpdateCatalogItemRevisions - Returns URL to update catalog revision
func (c *CentralConfiguration) UpdateCatalogItemRevisions(catalogItemID string) string {
	return c.GetCatalogItemsURL() + "/" + catalogItemID + "/revisions"
}

// GetCatalogItemByID - Returns URL to get catalog item by id
func (c *CentralConfiguration) GetCatalogItemByID(catalogItemID string) string {
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
		exception.Throw(errors.New("Error central.tenantID not set in config"))
	}

	if c.GetAgentType() == TraceabilityAgent {
		c.validateTraceabilityAgentConfig()
	} else {
		c.validateDiscoveryAgentConfig()
	}
}

func (c *CentralConfiguration) validateDiscoveryAgentConfig() {
	if c.GetURL() == "" {
		exception.Throw(errors.New("Error central.url not set in config"))
	}

	if c.GetTeamID() == "" {
		exception.Throw(errors.New("Error central.teamID not set in config"))
	}

	if c.GetAgentMode() == Connected {
		c.validateConnectedModeConfig()
	}

	if c.GetPollInterval() <= 0 {
		exception.Throw(errors.New("Error central.pollInterval not set in config"))
	}
}

func (c *CentralConfiguration) validateConnectedModeConfig() {
	if c.GetEnvironmentName() == "" {
		exception.Throw(errors.New("Error central.apiServerEnvironment not set in config"))
	}

	if c.APIServerVersion == "" {
		exception.Throw(errors.New("Error central.apiServerVersion not set in config"))
	}
}

func (c *CentralConfiguration) validateTraceabilityAgentConfig() {
	if c.GetAPICDeployment() == "" {
		exception.Throw(errors.New("Error central.apicDeployment not set in config"))
	}

	if c.GetEnvironmentID() == "" {
		exception.Throw(errors.New("Error central.environmentID not set in config"))
	}
}
