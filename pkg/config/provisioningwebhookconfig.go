package config

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

// ProvisioningWebhookAuthType - the authentication method used to call a provisioning webhook
type ProvisioningWebhookAuthType string

const (
	// ProvisioningWebhookAuthNone - no authentication
	ProvisioningWebhookAuthNone ProvisioningWebhookAuthType = "none"
	// ProvisioningWebhookAuthBasic - HTTP Basic authentication (username/password)
	ProvisioningWebhookAuthBasic ProvisioningWebhookAuthType = "basic"
	// ProvisioningWebhookAuthAPIKey - a static API key sent as a configurable header
	ProvisioningWebhookAuthAPIKey ProvisioningWebhookAuthType = "apiKey"
	// ProvisioningWebhookAuthBearer - a static token sent as an Authorization: Bearer header
	ProvisioningWebhookAuthBearer ProvisioningWebhookAuthType = "bearer"
)

// ProvisioningWebhookConfig - Interface for the on-prem provisioning webhook config, one webhook per resource type
type ProvisioningWebhookConfig interface {
	GetManagedApplicationWebhook() ProvisioningWebhookEndpointConfig
	GetAccessRequestWebhook() ProvisioningWebhookEndpointConfig
	GetCredentialWebhook() ProvisioningWebhookEndpointConfig
	ValidateConfig() error
}

// ProvisioningWebhookConfiguration - holds the provisioning webhook config for each resource type
type ProvisioningWebhookConfiguration struct {
	ProvisioningWebhookConfig
	ManagedApplication ProvisioningWebhookEndpointConfig `config:"managedApplication"`
	AccessRequest      ProvisioningWebhookEndpointConfig `config:"accessRequest"`
	Credential         ProvisioningWebhookEndpointConfig `config:"credential"`
}

func newProvisioningWebhookConfig() ProvisioningWebhookConfig {
	return &ProvisioningWebhookConfiguration{
		ManagedApplication: &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.managedApplication"}},
		AccessRequest:      &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.accessRequest"}},
		Credential:         &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential"}},
	}
}

// GetManagedApplicationWebhook - Returns the webhook config used for ManagedApplication provisioning
func (c *ProvisioningWebhookConfiguration) GetManagedApplicationWebhook() ProvisioningWebhookEndpointConfig {
	return c.ManagedApplication
}

// GetAccessRequestWebhook - Returns the webhook config used for AccessRequest provisioning
func (c *ProvisioningWebhookConfiguration) GetAccessRequestWebhook() ProvisioningWebhookEndpointConfig {
	return c.AccessRequest
}

// GetCredentialWebhook - Returns the webhook config used for Credential provisioning
func (c *ProvisioningWebhookConfiguration) GetCredentialWebhook() ProvisioningWebhookEndpointConfig {
	return c.Credential
}

// ValidateConfig - Validates each configured webhook
func (c *ProvisioningWebhookConfiguration) ValidateConfig() error {
	for _, webhook := range []ProvisioningWebhookEndpointConfig{c.ManagedApplication, c.AccessRequest, c.Credential} {
		if err := webhook.ValidateConfig(); err != nil {
			return err
		}
	}
	return nil
}

// ProvisioningWebhookEndpointConfig - Interface for a single provisioning webhook (one resource type).
// Extends WebhookConfig with a selectable auth method beyond a single secret.
type ProvisioningWebhookEndpointConfig interface {
	WebhookConfig
	GetAuthType() ProvisioningWebhookAuthType
	GetUsername() string
	GetPassword() string
	GetAPIKeyHeader() string
	GetAPIKeyValue() string
}

// ProvisioningWebhookEndpointConfiguration - config for a single provisioning webhook, built on WebhookConfiguration
type ProvisioningWebhookEndpointConfiguration struct {
	*WebhookConfiguration
	AuthType     string `config:"authType"`
	Username     string `config:"username"`
	Password     string `config:"password"`
	APIKeyHeader string `config:"apiKeyHeader"`
	APIKeyValue  string `config:"apiKeyValue"`
}

// GetAuthType - Returns the authentication method configured for this webhook
func (c *ProvisioningWebhookEndpointConfiguration) GetAuthType() ProvisioningWebhookAuthType {
	return ProvisioningWebhookAuthType(c.AuthType)
}

// GetUsername - Returns the username used for basic auth
func (c *ProvisioningWebhookEndpointConfiguration) GetUsername() string {
	return c.Username
}

// GetPassword - Returns the password used for basic auth
func (c *ProvisioningWebhookEndpointConfiguration) GetPassword() string {
	return c.Password
}

// GetAPIKeyHeader - Returns the header name the API key is sent on
func (c *ProvisioningWebhookEndpointConfiguration) GetAPIKeyHeader() string {
	return c.APIKeyHeader
}

// GetAPIKeyValue - Returns the API key value
func (c *ProvisioningWebhookEndpointConfiguration) GetAPIKeyValue() string {
	return c.APIKeyValue
}

// ValidateConfig - Validates the base webhook config (URL/headers), then the fields required by the configured auth type
func (c *ProvisioningWebhookEndpointConfiguration) ValidateConfig() error {
	if err := c.WebhookConfiguration.ValidateConfig(); err != nil {
		return err
	}
	if !c.IsConfigured() {
		return nil
	}

	switch c.GetAuthType() {
	case ProvisioningWebhookAuthNone, "":
	case ProvisioningWebhookAuthBasic:
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("central.%s.username and .password are required when authType is basic", c.Type)
		}
	case ProvisioningWebhookAuthAPIKey:
		if c.APIKeyHeader == "" || c.APIKeyValue == "" {
			return fmt.Errorf("central.%s.apiKeyHeader and .apiKeyValue are required when authType is apiKey", c.Type)
		}
	case ProvisioningWebhookAuthBearer:
		if c.GetSecret() == "" {
			return fmt.Errorf("central.%s.secret is required when authType is bearer", c.Type)
		}
	default:
		return fmt.Errorf("central.%s.authType must be one of none, basic, apiKey, bearer", c.Type)
	}

	return nil
}

const (
	pathProvisioningWebhookManagedApplication = "central.provisioningWebhook.managedApplication"
	pathProvisioningWebhookAccessRequest      = "central.provisioningWebhook.accessRequest"
	pathProvisioningWebhookCredential         = "central.provisioningWebhook.credential"
)

func addProvisioningWebhookConfigProperties(props properties.Properties) {
	addSingleProvisioningWebhookProperties(props, pathProvisioningWebhookManagedApplication, "ManagedApplication")
	addSingleProvisioningWebhookProperties(props, pathProvisioningWebhookAccessRequest, "AccessRequest")
	addSingleProvisioningWebhookProperties(props, pathProvisioningWebhookCredential, "Credential")
}

func addSingleProvisioningWebhookProperties(props properties.Properties, path, resourceType string) {
	props.AddStringProperty(path+".url", "", "URL of the webhook the agent calls instead of its own provisioning for "+resourceType+" events")
	props.AddStringProperty(path+".headers", "", "Static headers to send with every "+resourceType+" provisioning webhook call")
	props.AddStringProperty(path+".secret", "", "Bearer token for the "+resourceType+" provisioning webhook, used when authType is bearer")
	props.AddStringProperty(path+".authType", string(ProvisioningWebhookAuthNone), "Authentication method for the "+resourceType+" provisioning webhook: none, basic, apiKey, bearer")
	props.AddStringProperty(path+".username", "", "Username for "+resourceType+" provisioning webhook basic auth")
	props.AddStringProperty(path+".password", "", "Password for "+resourceType+" provisioning webhook basic auth")
	props.AddStringProperty(path+".apiKeyHeader", "", "Header name used to send the "+resourceType+" provisioning webhook API key")
	props.AddStringProperty(path+".apiKeyValue", "", "API key value for the "+resourceType+" provisioning webhook")
}

func parseProvisioningWebhookConfig(props properties.Properties) ProvisioningWebhookConfig {
	return &ProvisioningWebhookConfiguration{
		ManagedApplication: parseSingleProvisioningWebhookConfig(props, pathProvisioningWebhookManagedApplication, "provisioningWebhook.managedApplication"),
		AccessRequest:      parseSingleProvisioningWebhookConfig(props, pathProvisioningWebhookAccessRequest, "provisioningWebhook.accessRequest"),
		Credential:         parseSingleProvisioningWebhookConfig(props, pathProvisioningWebhookCredential, "provisioningWebhook.credential"),
	}
}

func parseSingleProvisioningWebhookConfig(props properties.Properties, path, name string) ProvisioningWebhookEndpointConfig {
	return &ProvisioningWebhookEndpointConfiguration{
		WebhookConfiguration: &WebhookConfiguration{
			Type:    name,
			URL:     props.StringPropertyValue(path + ".url"),
			Headers: props.StringPropertyValue(path + ".headers"),
			Secret:  props.StringPropertyValue(path + ".secret"),
		},
		AuthType:     props.StringPropertyValue(path + ".authType"),
		Username:     props.StringPropertyValue(path + ".username"),
		Password:     props.StringPropertyValue(path + ".password"),
		APIKeyHeader: props.StringPropertyValue(path + ".apiKeyHeader"),
		APIKeyValue:  props.StringPropertyValue(path + ".apiKeyValue"),
	}
}
