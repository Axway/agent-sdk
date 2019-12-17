package config

import (
	"errors"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/exception"
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
	GetAPIServerServicesURL() string
	GetAPIServerServicesRevisionsURL() string
	GetAPIServerServicesInstancesURL() string
	Validate() error
	GetAuthConfig() AuthConfig
	GetTLSConfig() TLSConfig
}

// CentralConfiguration - Structure to hold the central config
type CentralConfiguration struct {
	CentralConfig
	AgentType            AgentType
	Mode                 AgentMode  `config:"mode"`
	TenantID             string     `config:"tenantID"`
	TeamID               string     `config:"teamID" `
	APICDeployment       string     `config:"deployment"`
	APIServerEnvironment string     `config:"apiServerEnvironment"`
	EnvironmentID        string     `config:"environmentID"`
	URL                  string     `config:"url"`
	APIServerVersion     string     `config:"apiServerVersion"`
	Auth                 AuthConfig `config:"auth"`
	TLS                  TLSConfig  `config:"ssl"`
}

// NewCentralConfig - Creates the default central config
func NewCentralConfig(agentType AgentType) CentralConfig {
	return &CentralConfiguration{
		AgentType:        agentType,
		Mode:             Disconnected,
		APIServerVersion: "v1alpha1",
		Auth:             newAuthConfig(),
		TLS:              newTLSConfig(),
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

// GetCatalogItemsURL - Returns the URL for catalog items API
func (c *CentralConfiguration) GetCatalogItemsURL() string {
	return c.URL + "/api/unifiedCatalog/v1/catalogItems"
}

// GetCatalogItemImageURL - Returns the image based on catalogItemID
func (c *CentralConfiguration) GetCatalogItemImageURL(catalogItemID string) string {
	return c.URL + "/api/unifiedCatalog/v1/catalogItems/" + catalogItemID + "/image"
}

// getAPIServerURL - Returns the base path for the API server
func (c *CentralConfiguration) getAPIServerURL() string {
	return c.URL + "/apis/management/" + c.APIServerVersion + "/environments/"
}

// GetAPIServerServicesURL - Returns the APIServer URL for services API
func (c *CentralConfiguration) GetAPIServerServicesURL() string {
	return c.getAPIServerURL() + c.APIServerEnvironment + "/apiservices"
}

// GetAPIServerServicesRevisionsURL - Returns the APIServer URL for services API revisions
func (c *CentralConfiguration) GetAPIServerServicesRevisionsURL() string {
	return c.getAPIServerURL() + c.APIServerEnvironment + "/apiservicerevisions"
}

// GetAPIServerServicesInstancesURL - Returns the APIServer URL for services API instances
func (c *CentralConfiguration) GetAPIServerServicesInstancesURL() string {
	return c.getAPIServerURL() + c.APIServerEnvironment + "/apiserviceinstances"
}

// GetAuthConfig - Returns the Auth Config
func (c *CentralConfiguration) GetAuthConfig() AuthConfig {
	return c.Auth
}

// GetTLSConfig - Returns the TLS Config
func (c *CentralConfiguration) GetTLSConfig() TLSConfig {
	return c.TLS
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
