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
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "Header=contentType,Value=application/json", "")

	cfg := ParseSubscriptionConfig(props)
	assert.NotNil(t, cfg)

	err := ValidateConfig(cfg)
	assert.Nil(t, err)

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
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "x", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "Header=contentType,Value=application/json", "")

	cfg = ParseSubscriptionConfig(props)
	err = ValidateConfig(cfg)

	assert.NotNil(t, cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "central.subscriptions.notifications.webhook is not a valid URL", err.Error())

	// this one should be bad header
	rootCmd = &cobra.Command{
		Use: "test",
	}
	props = properties.NewProperties(rootCmd)
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "Header=contentType,Valaaaue=application/json", "")

	cfg = ParseSubscriptionConfig(props)
	err = ValidateConfig(cfg)

	assert.NotNil(t, err)
	assert.Equal(t, "could not parse value of central.subscriptions.notifications.headers", err.Error())

	// this one should be empty header
	rootCmd = &cobra.Command{
		Use: "test",
	}
	props = properties.NewProperties(rootCmd)
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "", "")

	cfg = ParseSubscriptionConfig(props)
	err = ValidateConfig(cfg)

	assert.NotNil(t, err)
	assert.Equal(t, "central.subscriptions.notifications.headers cannot be empty", err.Error())

	// this one should be ok, approval mode webhook
	rootCmd = &cobra.Command{
		Use: "test",
	}
	props = properties.NewProperties(rootCmd)
	props.AddStringProperty("central.subscriptions.approval.mode", "webhook", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "Header=contentType,Value=application/json", "")

	cfg = ParseSubscriptionConfig(props)
	err = ValidateConfig(cfg)

	assert.Nil(t, err)
	// assert.Equal(t, "central.subscriptions.notifications.headers cannot be empty", err.Error())
}

func TestSubscriptionSMTPConfig(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}
	props := properties.NewProperties(rootCmd)

	// this line is strange. Without it, it seems that rootCmd still has the values from
	// previous test, which cause validations to fail
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.host", "mail.axway.com", "")
	props.AddIntProperty("central.subscriptions.notifications.smtp.port", 111, "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.fromAddress", "foo@axway.com", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.authtype", "LOGIN", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.identity", "foo", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.username", "bill", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.password", "pwd", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.subject", "subscribe subject", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.body", "subscribe body", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.oath", "oath", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.apikeys", "apikeys", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribe.subject", "unsubscribe subject", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribe.body", "unsubscribe body", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribeFailed.subject", "subscribe failed subject", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribeFailed.body", "subscribe failed body", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribeFailed.subject", "unsubscribe failed subject", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribeFailed.body", "unsubscribe failed body", "")

	cfg := ParseSubscriptionConfig(props)
	assert.NotNil(t, cfg)

	err := ValidateConfig(cfg)
	assert.Nil(t, err)

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
