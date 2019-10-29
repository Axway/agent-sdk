package apic

import (
	"github.com/Axway/ace-golang-sdk/config"
	log "github.com/sirupsen/logrus"
)

// Config - Represents the APIC config
type Config struct {
	TenantID     string `cfg:"APIC_TENANT_ID"`
	ApicURL      string `cfg:"APIC_URL"`
	APIServerURL string `cfg:"API_SERVER_URL"`
	TeamID       string `cfg:"APIC_TEAM_ID"`
}

// ApicConfig - Holds the apicentral configuration
var apicConfig Config

func init() {
	config.ReadConfigFromEnv(&apicConfig)
	log.Debug("APIC config: ", apicConfig)
}

// GetApicConfig - Returns the auth config
func GetApicConfig() *Config {
	return &apicConfig
}

// GetApicURL - Returns the configured base apicentral URL
func (apicConfig *Config) GetApicURL() string {
	return apicConfig.ApicURL
}

// GetAPIServerURL - Returns the configured base apicentral URL
func (apicConfig *Config) GetAPIServerURL() string {
	return apicConfig.APIServerURL
}

// GetTenantID - Returns the configured apicentral tenantID
func (apicConfig *Config) GetTenantID() string {
	return apicConfig.TenantID
}

// GetTeamID - Returns the configured apicentral Default team id
func (apicConfig *Config) GetTeamID() string {
	return apicConfig.TeamID
}
