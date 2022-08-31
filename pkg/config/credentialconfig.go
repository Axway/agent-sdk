package config

import (
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

// SubscriptionConfig - Interface to get subscription config
type CredentialConfig interface {
	ShouldDeprovisionExpired() bool
	GetExpirationDays() int
}

// NotificationConfig -
type CredentialConfiguration struct {
	ExpirationDays      int  `config:"expirationDays"`
	DeprovisionOnExpire bool `config:"deprovisionOnExpire"`
}

// These constants are the paths that the settings are at in a config file
const (
	pathExpirationDays = "central.credentials.expirationDays"
	pathExpireAction   = "central.credentials.deprovisionOnExpire"
)

// AddCredentialConfigProperties -
func AddCredentialConfigProperties(props properties.Properties) {
	props.AddBoolProperty(pathExpireAction, false, "Set to true if an expired credential should be deprovisioned from the data plane")
	props.AddIntProperty(pathExpirationDays, 0, "The number of days until a provisioned credential expires, enforced on data plane if possible (default: 0 = Never) ")
}

// ParseCredentialConfig -
func ParseCredentialConfig(props properties.Properties) CredentialConfig {
	return &CredentialConfiguration{
		ExpirationDays:      props.IntPropertyValue(pathExpirationDays),
		DeprovisionOnExpire: props.BoolPropertyValue(pathExpireAction),
	}
}

// newCredentialConfig - Creates the default credential config
func newCredentialConfig() CredentialConfig {
	return &CredentialConfiguration{
		ExpirationDays:      0,
		DeprovisionOnExpire: false,
	}
}

// ExpireAction -
func (s *CredentialConfiguration) ShouldDeprovisionExpired() bool {
	return s.DeprovisionOnExpire
}

// GetTimeToLive -
func (s *CredentialConfiguration) GetExpirationDays() int {
	return s.ExpirationDays
}

// ValidateCfg - Validates the config, implementing IConfigInterface
func (s *CredentialConfiguration) ValidateCfg() error {
	// TODO - validate time to live
	return nil
}
