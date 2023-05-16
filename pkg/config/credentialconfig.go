package config

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
