package config

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestProvisioningWebhookConfigNotConfigured(t *testing.T) {
	cfg := newProvisioningWebhookConfig()
	assert.False(t, cfg.GetManagedApplicationWebhook().IsConfigured())
	assert.False(t, cfg.GetManagedApplicationProfileWebhook().IsConfigured())
	assert.False(t, cfg.GetAccessRequestWebhook().IsConfigured())
	assert.False(t, cfg.GetCredentialWebhook().IsConfigured())
	assert.Nil(t, cfg.ValidateConfig())
}

func TestProvisioningWebhookConfigAuthTypes(t *testing.T) {
	tests := []struct {
		name    string
		webhook *ProvisioningWebhookEndpointConfiguration
		wantErr string
	}{
		{
			name: "none",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthNone),
			},
		},
		{
			name: "basic ok",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthBasic),
				Username:             "user",
				Password:             "pass",
			},
		},
		{
			name: "basic missing password",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthBasic),
				Username:             "user",
			},
			wantErr: "central.provisioningWebhook.credential.username and .password are required when authType is basic",
		},
		{
			name: "apiKey ok",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthAPIKey),
				APIKeyHeader:         "X-Api-Key",
				APIKeyValue:          "abc123",
			},
		},
		{
			name: "apiKey missing header",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthAPIKey),
				APIKeyValue:          "abc123",
			},
			wantErr: "central.provisioningWebhook.credential.apiKeyHeader and .apiKeyValue are required when authType is apiKey",
		},
		{
			name: "bearer ok",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar", Secret: "token"},
				AuthType:             string(ProvisioningWebhookAuthBearer),
			},
		},
		{
			name: "bearer missing secret",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             string(ProvisioningWebhookAuthBearer),
			},
			wantErr: "central.provisioningWebhook.credential.secret is required when authType is bearer",
		},
		{
			name: "invalid auth type",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar"},
				AuthType:             "digest",
			},
			wantErr: "central.provisioningWebhook.credential.authType must be one of none, basic, apiKey, bearer",
		},
		{
			name: "bad url",
			webhook: &ProvisioningWebhookEndpointConfiguration{
				WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "xxxf"},
			},
			wantErr: "central.provisioningWebhook.credential.url is not a valid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.webhook.ValidateConfig()
			if tt.wantErr == "" {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			}
		})
	}
}

func TestProvisioningWebhookConfigRetryCountLimits(t *testing.T) {
	// retryCount is registered with WithLowerLimitInt(0)/WithUpperLimitInt(5); values outside that
	// range fall back to the default (0) rather than clamping to the nearest limit.
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{name: "below lower limit falls back to default", value: "-1", expected: 0},
		{name: "at lower limit", value: "0", expected: 0},
		{name: "within range", value: "3", expected: 3},
		{name: "at upper limit", value: "5", expected: 5},
		{name: "above upper limit falls back to default", value: "6", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := &cobra.Command{Use: "test"}
			props := properties.NewProperties(rootCmd)
			addProvisioningWebhookConfigProperties(props)

			err := rootCmd.Flags().Set("centralProvisioningWebhookCredentialRetryCount", tt.value)
			assert.Nil(t, err)

			cfg := parseProvisioningWebhookConfig(props)
			assert.Equal(t, tt.expected, cfg.GetCredentialWebhook().GetRetryCount())
		})
	}
}

func TestProvisioningWebhookConfigOnlyOneConfigured(t *testing.T) {
	cfg := &ProvisioningWebhookConfiguration{
		ManagedApplication:        &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.managedApplication"}},
		ManagedApplicationProfile: &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.managedApplicationProfile"}},
		AccessRequest:             &ProvisioningWebhookEndpointConfiguration{WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.accessRequest"}},
		Credential: &ProvisioningWebhookEndpointConfiguration{
			WebhookConfiguration: &WebhookConfiguration{Type: "provisioningWebhook.credential", URL: "https://foo.bar", Secret: "token"},
			AuthType:             string(ProvisioningWebhookAuthBearer),
		},
	}

	assert.False(t, cfg.GetManagedApplicationWebhook().IsConfigured())
	assert.False(t, cfg.GetManagedApplicationProfileWebhook().IsConfigured())
	assert.False(t, cfg.GetAccessRequestWebhook().IsConfigured())
	assert.True(t, cfg.GetCredentialWebhook().IsConfigured())
	assert.Nil(t, cfg.ValidateConfig())
}
