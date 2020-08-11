package config

import (
	"errors"
	"net/url"
	"strings"

	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
)

// WebhookConfig - Interface for webhook config
type WebhookConfig interface {
	GetURL() string
	GetWebhookHeaders() map[string]string
	GetSecret() string
	IsConfigured() bool
	ValidateConfig() error
}

// WebhookConfiguration -
type WebhookConfiguration struct {
	WebhookConfig
	URL            string `config:"url"`
	Headers        string `config:"headers"`
	Secret         string `config:"secret"`
	webhookHeaders map[string]string
}

// NewWebhookConfig -
func NewWebhookConfig() WebhookConfig {
	return &WebhookConfiguration{}
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

// Validate the config
func (c *WebhookConfiguration) ValidateConfig() error {
	if webhookURL := c.GetURL(); webhookURL != "" {
		if _, err := url.ParseRequestURI(webhookURL); err != nil {
			return errors.New("Error central.subscriptions.approvalWebhook.URL is not a valid URL")
		}
		log.Debug("Subscription approval webhook notification set")
	}

	// (example header) Header=contentType,Value=application/json, Header=Elements-Formula-Instance-Id,Value=440874, Header=Authorization,Value=User F+rYQSfu0w5yIa5q7uNs2MKYcIok8pYpgAUwJtXFnzc=, Organization a1713018bbde8f54f4f55ff8c3bd8bfe
	c.webhookHeaders = map[string]string{}
	c.Headers = strings.Replace(c.Headers, ", ", ",", -1)
	headersValues := strings.Split(c.Headers, ",Header=")
	for _, headerValue := range headersValues {
		hvArray := strings.Split(headerValue, ",Value=")
		if len(hvArray) != 2 {
			return errors.New("Could not parse value of subscriptions.approvalWebhook.headers")
		}
		hvArray[0] = strings.TrimLeft(hvArray[0], "Header=") // handle the first	header in the list
		c.webhookHeaders[hvArray[0]] = hvArray[1]
	}

	return nil
}
