package config

import "errors"

var supportedOAuthMethods = map[string]bool{
	"oauth-secret":     true,
	"oauth-public-key": true,
}

// SubscriptionConfig - Interface to get subscription config
type CredentialConfig interface {
	SetAllowedOAuthMethods(allowedMethods []string)
	GetAllowedOAuthMethods() []string
	ShouldDeprovisionExpired() bool
	SetShouldDeprovisionExpired(deprovisionExpired bool)
	GetExpirationDays() int
	SetExpirationDays(expirationDays int)
}

// NotificationConfig -
type CredentialConfiguration struct {
	AllowedOAuthMethods []string `config:"allowedOAuthMethods"`
	ExpirationDays      int      `config:"expirationDays"`
	DeprovisionOnExpire bool     `config:"deprovisionOnExpire"`
}

// newCredentialConfig - Creates the default credential config
func newCredentialConfig() CredentialConfig {
	return &CredentialConfiguration{
		AllowedOAuthMethods: make([]string, 0),
		ExpirationDays:      0,
		DeprovisionOnExpire: false,
	}
}

// SetAllowedOAuthMethods -
func (s *CredentialConfiguration) SetAllowedOAuthMethods(allowedOAuthMethods []string) {
	s.AllowedOAuthMethods = allowedOAuthMethods
}

// GetAllowedOAuthMethods -
func (s *CredentialConfiguration) GetAllowedOAuthMethods() []string {
	return s.AllowedOAuthMethods
}

// ExpireAction -
func (s *CredentialConfiguration) ShouldDeprovisionExpired() bool {
	return s.DeprovisionOnExpire
}

// Set ExpireAction -
func (s *CredentialConfiguration) SetShouldDeprovisionExpired(deprovisionExpired bool) {
	s.DeprovisionOnExpire = deprovisionExpired
}

// GetTimeToLive -
func (s *CredentialConfiguration) GetExpirationDays() int {
	return s.ExpirationDays
}

// Set GetTimeToLive -
func (s *CredentialConfiguration) SetExpirationDays(expirationDays int) {
	s.ExpirationDays = expirationDays
}

// ValidateCfg - Validates the config, implementing IConfigInterface
func (s *CredentialConfiguration) ValidateCfg() error {
	for _, method := range s.AllowedOAuthMethods {
		if _, ok := supportedOAuthMethods[method]; !ok {
			return errors.New("credential type in allowed method configuration is not supported")
		}
	}
	// TODO - validate time to live
	return nil
}
