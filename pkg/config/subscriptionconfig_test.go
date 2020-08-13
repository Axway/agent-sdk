package config

import (
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSubscriptionWebhookConfig(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}
	props := properties.NewProperties(rootCmd)
	props.AddStringProperty("subscriptions.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("subscriptions.webhook.headers", "Header=contentType,Value=application/json", "")

	cfg, err := ParseSubscriptionConfig(props)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "https://foo.bar", cfg.GetWebhookURL())
	m := map[string]string{"contentType": "application/json"}
	assert.Equal(t, m, cfg.GetWebhookHeaders())

	types := cfg.GetNotificationTypes()
	assert.NotNil(t, types)
	assert.Equal(t, 1, len(types))
	assert.Equal(t, NotifyWebhook, types[0])

	// this one should be bad url
	rootCmd = &cobra.Command{
		Use: "test",
	}
	props = properties.NewProperties(rootCmd)
	props.AddStringProperty("subscriptions.webhook.url", "x", "")
	props.AddStringProperty("subscriptions.webhook.headers", "Header=contentType,Value=application/json", "")

	cfg, err = ParseSubscriptionConfig(props)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "central.subscriptions.webhook is not a valid URL", err.Error())

	// this one should be bad header
	rootCmd = &cobra.Command{
		Use: "test",
	}
	props = properties.NewProperties(rootCmd)
	props.AddStringProperty("subscriptions.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("subscriptions.webhook.headers", "Header=contentType,Valaaaue=application/json", "")

	cfg, err = ParseSubscriptionConfig(props)
	assert.Nil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "could not parse value of central.subscriptions.notificationHeaders", err.Error())
}

func TestSubscriptionSMTPConfig(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}
	props := properties.NewProperties(rootCmd)

	// this line is strange. Without it, it seems that rootCmd still has the values from
	// previous test, which cause validations to fail
	props.AddStringProperty("subscriptions.webhook.url", "", "")
	props.AddStringProperty("subscriptions.smtp.host", "mail.axway.com", "")
	props.AddIntProperty("subscriptions.smtp.port", 111, "")
	props.AddStringProperty("subscriptions.smtp.fromAddress", "foo@axway.com", "")
	props.AddStringProperty("subscriptions.smtp.authtype", "LOGIN", "")
	props.AddStringProperty("subscriptions.smtp.identity", "foo", "")
	props.AddStringProperty("subscriptions.smtp.username", "bill", "")
	props.AddStringProperty("subscriptions.smtp.password", "pwd", "")
	props.AddStringProperty("subscriptions.smtp.subscribe.subject", "subscribe subject", "")
	props.AddStringProperty("subscriptions.smtp.subscribe.body", "subscribe body", "")
	props.AddStringProperty("subscriptions.smtp.subscribe.oath", "oath", "")
	props.AddStringProperty("subscriptions.smtp.subscribe.apikeys", "apikeys", "")
	props.AddStringProperty("subscriptions.smtp.unsubscribe.subject", "unsubscribe subject", "")
	props.AddStringProperty("subscriptions.smtp.unsubscribe.body", "unsubscribe body", "")
	props.AddStringProperty("subscriptions.smtp.subscribeFailed.subject", "subscribe failed subject", "")
	props.AddStringProperty("subscriptions.smtp.subscribeFailed.body", "subscribe failed body", "")
	props.AddStringProperty("subscriptions.smtp.unsubscribeFailed.subject", "unsubscribe failed subject", "")
	props.AddStringProperty("subscriptions.smtp.unsubscribeFailed.body", "unsubscribe failed body", "")

	cfg, err := ParseSubscriptionConfig(props)
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	types := cfg.GetNotificationTypes()
	assert.NotNil(t, types)
	assert.Equal(t, 1, len(types))
	assert.Equal(t, NotifySMTP, types[0])

	assert.Equal(t, "mail.axway.com:111", cfg.GetSMTPURL())
	assert.Equal(t, "mail.axway.com", cfg.GetSMTPHost())
	assert.Equal(t, "foo@axway.com", cfg.GetSMTPFromAddress())
	assert.Equal(t, LoginAuth, cfg.GetSMTPAuthType())
	assert.Equal(t, "foo", cfg.GetSMTPIdentity())
	assert.Equal(t, "bill", cfg.GetSMTPUsername())
	assert.Equal(t, "pwd", cfg.GetSMTPPassword())

	template := cfg.GetSubscribeTemplate()
	assert.NotNil(t, template)
	assert.Equal(t, "subscribe subject", template.Subject)
	assert.Equal(t, "subscribe body", template.Body)

	template = cfg.GetUnsubscribeTemplate()
	assert.NotNil(t, template)
	assert.Equal(t, "unsubscribe subject", template.Subject)
	assert.Equal(t, "unsubscribe body", template.Body)

	template = cfg.GetSubscribeFailedTemplate()
	assert.NotNil(t, template)
	assert.Equal(t, "subscribe failed subject", template.Subject)
	assert.Equal(t, "subscribe failed body", template.Body)

	template = cfg.GetUnsubscribeFailedTemplate()
	assert.NotNil(t, template)
	assert.Equal(t, "unsubscribe failed subject", template.Subject)
	assert.Equal(t, "unsubscribe failed body", template.Body)
}

func TestNewSubscriptionConfig(t *testing.T) {
	cfg := NewSubscriptionConfig()
	assert.NotNil(t, cfg)
}
