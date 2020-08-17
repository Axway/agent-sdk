package notify

import (
	"strings"
	"testing"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func buildConfig() (config.SubscriptionConfig, error) {
	rootCmd := &cobra.Command{
		Use: "test",
	}
	props := properties.NewProperties(rootCmd)
	props.AddStringProperty("subscriptions.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("subscriptions.webhook.headers", "Header=contentType,Value=application/json", "")

	// the SMTP host/port are set to this to force the SMTP send to fail.
	props.AddStringProperty("subscriptions.smtp.host", "x", "")
	props.AddIntProperty("subscriptions.smtp.port", 0, "")

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

	return config.ParseSubscriptionConfig(props)
}

func TestSubscriptionNotification(t *testing.T) {
	cfg, err := buildConfig()
	assert.Nil(t, err)
	assert.NotNil(t, cfg)
	SetSubscriptionConfig(cfg)

	catalogID := "12345"
	catalogName := "MyAPI"
	catalogItemURL := "http://foo.bar/12345"
	recipient := "joe@axway.com"
	authID := "1111"
	apiKeyFieldName := "passthru"
	authSecret := "abcde"
	message := "new subscription received"

	subNotif := NewSubscriptionNotification(catalogID, catalogName, catalogItemURL, recipient,
		authID, apiKeyFieldName, authSecret, apic.SubscriptionApproved, message) // this is a bad action
	subNotif.SetAuthorizationTemplate(apikeys)

	subNotif = NewSubscriptionNotification(catalogID, catalogName, catalogItemURL, recipient,
		authID, apiKeyFieldName, authSecret, apic.SubscriptionActive, message)
	subNotif.apiClient = &coreapi.MockClient{}

	// Set the authtemplate based on the authtype
	subNotif.SetAuthorizationTemplate("")           // try a bad value
	subNotif.SetAuthorizationTemplate("apikeysfff") // try a bad value
	subNotif.SetAuthorizationTemplate(oauth)
	subNotif.SetAuthorizationTemplate(apikeys)

	err = subNotif.NotifySubscriber(recipient) // logon

	cfg1 := cfg.(*config.SubscriptionConfiguration)
	cfg1.SMTP.AuthType = config.AnonymousAuth
	err = subNotif.NotifySubscriber(recipient) // plainauth

	cfg1.SMTP.AuthType = config.PlainAuth
	err = subNotif.NotifySubscriber(recipient) // anonymous

	// this can never succeed because the SMTP will fail
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "could not send notification via smtp"))
}
