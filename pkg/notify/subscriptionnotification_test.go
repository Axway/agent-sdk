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

//TODO
/*
	1. Search for comment "DEPRECATED to be removed on major release"
	2. Remove deprecated code left from APIGOV-19751
*/

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
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.body", "Subscription created for Catalog Item:  <a href= {{.CatalogItemURL}}> {{.CatalogItemName}} {{.CatalogItemID}}.</br></a>{{if .IsAPIKey}} Your API is secured using an APIKey credential:header:<b>{{.KeyHeaderName}}</b>/value:<b>{{.Key}}</b>{{else}} Your API is secured using OAuth token. You can obtain your token using grant_type=client_credentials with the following client_id=<b>{{.ClientID}}</b> and client_secret=<b>{{.ClientSecret}}</b>{{end}}", "")

	//DEPRECATED to be removed on major release - this property will no longer be needed after "${tag} is invalid"
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.oath", "Your API is secured using OAuth token. You can obtain your token using grant_type=client_credentials with the following client_id=<b>{{.ClientID}}</b> and client_secret=<b>{{.ClientSecret}}</b>", "")
	//DEPRECATED to be removed on major release - this property will no longer be needed after "${tag} is invalid"
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribe.apikeys", "Your API is secured using an APIKey credential:header:<b>{{.KeyHeaderName}}</b>/value:<b>{{}.Key}}</b>", "")

	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribe.subject", "Subscription Removal Notification", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribe.body", "Subscription for Catalog Item: <a href= {{.CatalogItemURL}}> {{CatalogItemName}} </a> has been unsubscribed", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribeFailed.subject", "Subscription Failed Notification", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.subscribeFailed.body", "Could not subscribe to Catalog Item: <a href= {{CatalogItemURL}}> {{CatalogItemName}}</a> {{.Message}}", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribeFailed.subject", "Subscription Removal Failed Notification", "")
	props.AddStringProperty("central.subscriptions.notifications.smtp.unsubscribeFailed.body", "Could not unsubscribe to Catalog Item: <a href= {{.CatalogItemUrl}}> {{.CatalogItemName}}  </a>*{{.Message}}", "")

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

	_ = subNotif.NotifySubscriber(recipient) // logon

	cfg1 := cfg.(*config.SubscriptionConfiguration)
	cfg1.Notifications.SMTP.AuthType = config.AnonymousAuth
	_ = subNotif.NotifySubscriber(recipient) // plainauth

	cfg1.Notifications.SMTP.AuthType = config.PlainAuth
	err = subNotif.NotifySubscriber(recipient) // anonymous

	// this can never succeed because the SMTP will fail
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "could not send notification via smtp"))
}
