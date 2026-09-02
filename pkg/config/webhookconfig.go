package config

import (
	"fmt"
	"net/url"
	"strings"

	log "github.com/Axway/agent-sdk/pkg/util/log"
)

// WebhookConfig - Interface for webhook config
type WebhookConfig interface {
	GetURL() string
	GetWebhookHeaders() map[string]string
	GetSecret() string
	IsConfigured() bool
	ValidateConfig() error
}

// WebhookConfiguration - do NOT make this an IConfigValidator, as it is validated as part of subscriptionConfig
type WebhookConfiguration struct {
	WebhookConfig
	// Type identifies which webhook config this is (e.g. "subscriptions.approvalWebhook", "provisioningWebhook"),
	// used only to make ValidateConfig error messages point at the right config section.
	Type           string
	URL            string `config:"url"`
	Headers        string `config:"headers"`
	Secret         string `config:"secret"`
	webhookHeaders map[string]string
}

// NewWebhookConfig - Creates a webhook config identified by name (used to disambiguate error messages
// when more than one webhook config section exists, e.g. "subscriptions.approvalWebhook", "provisioningWebhook")
func NewWebhookConfig(name string) WebhookConfig {
	return &WebhookConfiguration{Type: name}
}

// GetURL - Returns the URL
func (c *WebhookConfiguration) GetURL() string {
	return c.URL
}

// IsConfigured - bool
func (c *WebhookConfiguration) IsConfigured() bool {
	return c.URL != ""
}

// GetWebhookHeaders - Returns the webhook headers
func (c *WebhookConfiguration) GetWebhookHeaders() map[string]string {
	return c.webhookHeaders
}

// GetSecret - Returns the secret
func (c *WebhookConfiguration) GetSecret() string {
	return c.Secret
}

// ValidateConfig - Validate the config. Do NOT make this ValidateCfg IConfigValidator. It is called directly from the subscriptionconfig
// validator. But, it is ONLY called if the approvalMode is "webhook"
func (c *WebhookConfiguration) ValidateConfig() error {
	if c.IsConfigured() {
		webhookURL := c.GetURL()
		if _, err := url.ParseRequestURI(webhookURL); err != nil {
			return fmt.Errorf("central.%s.url is not a valid URL", c.Type)
		}

		// headers are allowed to be empty, so only validate if there is a configured value
		if c.Headers != "" {
			// (example header) Header=contentType,Value=application/json, Header=Elements-Formula-Instance-Id,Value=440874, Header=Authorization,Value=User F+rYQSfu0w5yIa5q7uNs2MKYcIok8pYpgAUwJtXFnzc=, Organization a1713018bbde8f54f4f55ff8c3bd8bfe
			c.webhookHeaders = map[string]string{}
			c.Headers = strings.Replace(c.Headers, ", ", ",", -1)
			headersValues := strings.Split(c.Headers, ",Header=")
			for _, headerValue := range headersValues {
				hvArray := strings.Split(headerValue, ",Value=")
				if len(hvArray) != 2 {
					return fmt.Errorf("could not parse value of central.%s.headers", c.Type)
				}
				hvArray[0] = strings.TrimPrefix(hvArray[0], "Header=") // handle the first header in the list
				c.webhookHeaders[hvArray[0]] = hvArray[1]
			}
		}
		log.Tracef("%s webhook configuration set", c.Type)
	}

	return nil
}
