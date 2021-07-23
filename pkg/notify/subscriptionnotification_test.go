package notify

import (
	"strings"
	"testing"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func buildConfig() (config.SubscriptionConfig, error) {
	rootCmd := &cobra.Command{
		Use: "test",
	}
	props := properties.NewProperties(rootCmd)
	props.AddStringProperty("central.subscriptions.notifications.webhook.url", "https://foo.bar", "")
	props.AddStringProperty("central.subscriptions.notifications.webhook.headers", "Header=contentType,Value=application/json", "")

	// the SMTP host/port are set to this to force the SMTP send to fail.
	props.AddStringProperty("central.subscriptions.notifications.smtp.host", "x", "")
	props.AddIntProperty("central.subscriptions.notifications.smtp.port", 0, "")

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

	cfg := config.ParseSubscriptionConfig(props)
	err := config.ValidateConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestSubscriptionNotification(t *testing.T) {
	cfg, err := buildConfig()
	assert.Nil(t, err)
	assert.NotNil(t, cfg)

	assert.Nil(t, err)

	SetSubscriptionConfig(cfg)

	catalogID := "12345"
	catalogName := "MyAPI"
	catalogItemURL := "http://foo.bar/12345"
	recipient := "joe@axway.com"
	authID := "1111"
	apiKeyFieldName := "passthru"
	authSecret := "abcde"
	message := "new subscription received"

	subNotif := NewSubscriptionNotification(recipient, message, apic.SubscriptionApproved) // this is a bad action
	subNotif.SetCatalogItemInfo(catalogID, catalogName, catalogItemURL)
	subNotif.SetAPIKeyInfo(authID, apiKeyFieldName)
	subNotif.SetAuthorizationTemplate(Apikeys)

	subNotif = NewSubscriptionNotification(recipient, message, apic.SubscriptionActive)
	subNotif.SetCatalogItemInfo(catalogID, catalogName, catalogItemURL)
	subNotif.SetAPIKeyInfo(authID, apiKeyFieldName)

	// set up a mock HTTP client
	subNotif.apiClient = &coreapi.MockHTTPClient{}

	// Set the authtemplate based on the authtype
	subNotif.SetAuthorizationTemplate("")           // try a bad value
	subNotif.SetAuthorizationTemplate("apikeysfff") // try a bad value
	subNotif.SetOauthInfo(authID, authSecret)
	subNotif.SetAuthorizationTemplate(Oauth)
	subNotif.SetAPIKeyInfo(authID, apiKeyFieldName)
	subNotif.SetAuthorizationTemplate(Apikeys)

	err = subNotif.NotifySubscriber(recipient) // logon
	assert.Nil(t, err)

	cfg1 := cfg.(*config.SubscriptionConfiguration)
	cfg1.Notifications.SMTP.AuthType = config.AnonymousAuth
	err = subNotif.NotifySubscriber(recipient) // plainauth
	assert.Nil(t, err)

	cfg1.Notifications.SMTP.AuthType = config.PlainAuth
	err = subNotif.NotifySubscriber(recipient) // anonymous

	// this can never succeed because the SMTP will fail
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "could not send notification via smtp"))
}
