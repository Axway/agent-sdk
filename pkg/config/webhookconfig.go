package config

import (
	"errors"
	"net/url"
	"strings"
)

// WebhookConfig - Interface for webhook config
type WebhookConfig interface {
	GetURL() string
	// GetRealm() string
	// GetAudience() string
	// GetClientID() string
	// GetPrivateKey() string
	// GetPublicKey() string
	// GetKeyPassword() string
	// GetTimeout() time.Duration
	Validate() error
}

// WebhookConfiguration -
type WebhookConfiguration struct {
	WebhookConfig
	url                 string `config:"url"`
	headers             string `config:"headers"`
	notificationHeaders map[string]string
}

func newWebhookConfig() WebhookConfig {
	return &WebhookConfiguration{}
}

// GetURL - Returns the URL
func (c *WebhookConfiguration) GetURL() string {
	return c.url
}

// GetNotificationHeaders - Returns the notification headers
func (c *WebhookConfiguration) GetNotificationHeaders() map[string]string {
	return c.notificationHeaders
}

// Validate the config
func (c *WebhookConfiguration) Validate() error {
	if webhookURL := c.GetURL(); webhookURL != "" {
		if _, err := url.ParseRequestURI(webhookURL); err != nil {
			return errors.New("Error central.subscriptions.webhook.URL not a valid URL")
		}
	}
	// Header=contentType,Value=application/json, Header=Elements-Formula-Instance-Id,Value=440874, Header=Authorization,Value=User F+rYQSfu0w5yIa5q7uNs2MKYcIok8pYpgAUwJtXFnzc=, Organization a1713018bbde8f54f4f55ff8c3bd8bfe
	c.notificationHeaders = map[string]string{}
	c.headers = strings.Replace(c.headers, ", ", ",", -1)
	headersValues := strings.Split(c.headers, ",Header=")
	for _, headerValue := range headersValues {
		hvArray := strings.Split(headerValue, ",Value=")
		if len(hvArray) != 2 {
			return errors.New("Could not parse value of subscriptions.approvalWebhook.headers")
		}
		hvArray[0] = strings.TrimLeft(hvArray[0], "Header=") // handle the first	header in the list
		c.notificationHeaders[hvArray[0]] = hvArray[1]
	}

	return nil
}
