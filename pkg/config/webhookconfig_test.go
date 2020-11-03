package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebookConfig(t *testing.T) {
	cfg := NewWebhookConfig()
	assert.False(t, cfg.IsConfigured())

	err := cfg.ValidateConfig()
	assert.Nil(t, err)

	// this one should be all good
	cfg = &WebhookConfiguration{
		URL:     "https://foo.bar:4567",
		Headers: "Header=contentType,Value=application/json",
		Secret:  "1234",
	}

	assert.True(t, cfg.IsConfigured())

	err = cfg.ValidateConfig()
	assert.Nil(t, err)
	assert.Equal(t, "https://foo.bar:4567", cfg.GetURL())
	m := map[string]string{"contentType": "application/json"}
	assert.Equal(t, m, cfg.GetWebhookHeaders())
	assert.Equal(t, "1234", cfg.GetSecret())

	// this one should be all good with no headers
	cfg = &WebhookConfiguration{
		URL:     "https://foo.bar:4567",
		Headers: "",
		Secret:  "1234",
	}

	err = cfg.ValidateConfig()
	assert.Nil(t, err)

	// this one should be all good with no secret
	cfg = &WebhookConfiguration{
		URL:     "https://foo.bar:4567",
		Headers: "Header=contentType,Value=application/json",
		Secret:  "",
	}

	err = cfg.ValidateConfig()
	assert.Nil(t, err)

	// this one should be bad url
	cfg = &WebhookConfiguration{
		URL:     "xxxf",
		Headers: "Header=contentType,Value=application/json",
		Secret:  "1234",
	}
	err = cfg.ValidateConfig()
	assert.NotNil(t, err)
	assert.Equal(t, "central.subscriptions.approvalWebhook.URL is not a valid URL", err.Error())

	// this one should be bad header
	cfg = &WebhookConfiguration{
		URL:     "https://foo.bar:4567",
		Headers: "Header=contentType,Vue=application/json",
		Secret:  "1234",
	}
	err = cfg.ValidateConfig()
	assert.NotNil(t, err)
	assert.Equal(t, "could not parse value of central.subscriptions.approvalWebhook.headers", err.Error())
}
