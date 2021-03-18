package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// NotificationType - Type definition for subscription state
type NotificationType string

// NotificationTypes
const (
	NotifySMTP    = NotificationType("SMTP")
	NotifyWebhook = NotificationType("WEBHOOK")
)

// SMTPAuthType - the type of authentication methods the SMTP client supports
type SMTPAuthType string

// SMTPAuthTypes -
const (
	AnonymousAuth = SMTPAuthType("ANONYMOUS")
	LoginAuth     = SMTPAuthType("LOGIN")
	PlainAuth     = SMTPAuthType("PLAIN")
	NoAuth        = SMTPAuthType("NONE")
)

// SubscriptionConfig - Interface to get subscription config
type SubscriptionConfig interface {
	PollingEnabled() bool
	GetNotificationTypes() []NotificationType
	GetWebhookURL() string
	GetWebhookHeaders() map[string]string
	GetSMTPURL() string
	GetSMTPHost() string
	GetSMTPFromAddress() string
	GetSMTPAuthType() SMTPAuthType
	GetSMTPIdentity() string
	GetSMTPUsername() string
	GetSMTPPassword() string
	GetSubscribeTemplate() *EmailTemplate
	GetUnsubscribeTemplate() *EmailTemplate
	GetSubscribeFailedTemplate() *EmailTemplate
	GetUnsubscribeFailedTemplate() *EmailTemplate
	GetSubscriptionApprovalMode() string
	GetSubscriptionApprovalWebhookConfig() WebhookConfig
}

// NotificationConfig -
type NotificationConfig struct {
	SMTP    *smtp         `config:"smtp"`
	Webhook WebhookConfig `config:"webhook"`
}

type smtp struct {
	Host              string         `config:"host"`
	Port              int            `config:"port"`
	From              string         `config:"fromAddress"`
	AuthType          SMTPAuthType   `config:"authType"`
	Identity          string         `config:"identity"`
	Username          string         `config:"username"`
	Password          string         `config:"password"`
	Subscribe         *EmailTemplate `config:"subscribe"`
	Unsubscribe       *EmailTemplate `config:"unsubscribe"`
	SubscribeFailed   *EmailTemplate `config:"subscribeFailed"`
	UnsubscribeFailed *EmailTemplate `config:"unsubscribeFailed"`
}

// ApprovalConfig -
type ApprovalConfig struct {
	SubscriptionApprovalMode    string        `config:"mode"`
	SubscriptionApprovalWebhook WebhookConfig `config:"webhook"`
}

// SubscriptionConfiguration - Structure to hold the subscription config
type SubscriptionConfiguration struct {
	SubscriptionConfig
	IConfigValidator
	DisablePolling bool                `config:"disablePolling"`
	Approval       *ApprovalConfig     `config:"approval"`
	Notifications  *NotificationConfig `config:"notifications"`
	Types          []NotificationType
}

// These constants are the paths that the settings are at in a config file
const (
	pathSubscriptionsDisablePolling                            = "central.subscriptions.disablePolling"
	pathSubscriptionsApprovalMode                              = "central.subscriptions.approval.mode"
	pathSubscriptionsApprovalWebhookURL                        = "central.subscriptions.approval.webhook.url"
	pathSubscriptionsApprovalWebhookHeaders                    = "central.subscriptions.approval.webhook.headers"
	pathSubscriptionsApprovalWebhookSecret                     = "central.subscriptions.approval.webhook.authSecret"
	pathSubscriptionsNotificationsWebhookURL                   = "central.subscriptions.notifications.webhook.url"
	pathSubscriptionsNotificationsWebhookHeaders               = "central.subscriptions.notifications.webhook.headers"
	pathSubscriptionsNotificationsSMTPHost                     = "central.subscriptions.notifications.smtp.host"
	pathSubscriptionsNotificationsSMTPPort                     = "central.subscriptions.notifications.smtp.port"
	pathSubscriptionsNotificationsSMTPFrom                     = "central.subscriptions.notifications.smtp.fromAddress"
	pathSubscriptionsNotificationsSMTPIdentity                 = "central.subscriptions.notifications.smtp.identity"
	pathSubscriptionsNotificationsSMTPAuth                     = "central.subscriptions.notifications.smtp.authType"
	pathSubscriptionsNotificationsSMTPUserName                 = "central.subscriptions.notifications.smtp.username"
	pathSubscriptionsNotificationsSMTPUserPassword             = "central.subscriptions.notifications.smtp.password"
	pathSubscriptionsNotificationsSMTPSubscribeSubject         = "central.subscriptions.notifications.smtp.subscribe.subject"
	pathSubscriptionsNotificationsSMTPSubscribeBody            = "central.subscriptions.notifications.smtp.subscribe.body"
	pathSubscriptionsNotificationsSMTPSubscribeOauth           = "central.subscriptions.notifications.smtp.subscribe.oauth"
	pathSubscriptionsNotificationsSMTPSubscribeAPIKeys         = "central.subscriptions.notifications.smtp.subscribe.apikeys"
	pathSubscriptionsNotificationsSMTPUnsubscribeSubject       = "central.subscriptions.notifications.smtp.unsubscribe.subject"
	pathSubscriptionsNotificationsSMTPUnsubscribeBody          = "central.subscriptions.notifications.smtp.unsubscribe.body"
	pathSubscriptionsNotificationsSMTPSubscribeFailedSubject   = "central.subscriptions.notifications.smtp.subscribeFailed.subject"
	pathSubscriptionsNotificationsSMTPSubscribeFailedBody      = "central.subscriptions.notifications.smtp.subscribeFailed.body"
	pathSubscriptionsNotificationsSMTPUnsubscribeFailedSubject = "central.subscriptions.notifications.smtp.unsubscribeFailed.subject"
	pathSubscriptionsNotificationsSMTPUnsubscribeFailedBody    = "central.subscriptions.notifications.smtp.unsubscribeFailed.body"
)

//EmailTemplate -
type EmailTemplate struct {
	Subject string `config:"subject"`
	Body    string `config:"body"`
	Oauth   string `config:"oauth"`
	APIKey  string `config:"apikeys"`
}

// AddSubscriptionConfigProperties -
func AddSubscriptionConfigProperties(props properties.Properties) {
	// subscriptions enabled
	props.AddBoolProperty(pathSubscriptionsDisablePolling, false, "Set to true to disable polling for subscriptions on the agent")

	// subscription approvals
	props.AddStringProperty(pathSubscriptionsApprovalMode, ManualApproval, "The mode to use for approving subscriptions for AMPLIFY Central (manual, webhook, auto)")
	props.AddStringProperty(pathSubscriptionsApprovalWebhookURL, "", "The subscription webhook URL to use for approving subscriptions for AMPLIFY Central")
	props.AddStringProperty(pathSubscriptionsApprovalWebhookHeaders, "", "The subscription webhook headers to pass to the subscription approval webhook")
	props.AddStringProperty(pathSubscriptionsApprovalWebhookSecret, "", "The authentication secret to use for the subscription approval webhook")

	// subscription notifications
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPHost, "", "SMTP server where the email notifications will originate from")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPPort, "", "Port of the SMTP server")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPFrom, "", "Email address which will represent the sender")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPIdentity, "", "foobar")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPAuth, "", "The authentication type based on the email server")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUserName, "", "Login user for the SMTP server")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUserPassword, "", "Login password for the SMTP server")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeSubject, "Subscription Notification", "Subject of the email notification for action subscribe")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeBody, "Subscription created for Catalog Item: <a href= ${catalogItemUrl}> ${catalogItemName}</a><br/>${authtemplate}<br/>", "Body of the email notification for action subscribe")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeOauth, "Your API is secured using OAuth token. You can obtain your token using grant_type=client_credentials with the following client_id=<b>${clientID}</b> and client_secret=<b>${clientSecret}</b>", "Body of the email notification for action subscribe on OAuth authorization if your API is secured using OAuth token")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeAPIKeys, "Your API is secured using an APIKey credential: header: <b>${keyHeaderName}</b> / value: <b>${key}</b>", "Body of the email notification for action subscribe on APIKey authorization if your API is secured using an APIKey")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUnsubscribeSubject, "Subscription Removal Notification", "Subject of the email notification for action unsubscribe")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUnsubscribeBody, "Subscription for Catalog Item: <a href= ${catalogItemUrl}> ${catalogItemName}</a> has been unsubscribed", "Body of the email notification for action unsubscribe")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeFailedSubject, "Subscription Failed Notification", "Subject of the email notification for action subscribe failed")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPSubscribeFailedBody, "Could not subscribe to CatalogItem: <a href= ${catalogItemUrl}> ${catalogItemName}</a>", "Body of the email notification for action subscribe failed")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUnsubscribeFailedSubject, "Subscription Removal Failed Notification", "Subject of the email notification for action unsubscribe failed")
	props.AddStringProperty(pathSubscriptionsNotificationsSMTPUnsubscribeFailedBody, "Could not unsubscribe to Catalog Item: <a href= ${catalogItemUrl}> ${catalogItemName}</a>", "Body of the email notification for action unsubscribe failed")
}

// ParseSubscriptionConfig -
func ParseSubscriptionConfig(props properties.Properties) SubscriptionConfig {
	// Determine the auth type
	authTypeString := props.StringPropertyValue(pathSubscriptionsNotificationsSMTPAuth)
	authType := NoAuth
	switch strings.ToUpper(authTypeString) {
	case (string(LoginAuth)):
		authType = LoginAuth
	case (string(PlainAuth)):
		authType = PlainAuth
	case (string(AnonymousAuth)):
		authType = AnonymousAuth
	}

	cfg := &SubscriptionConfiguration{
		DisablePolling: props.BoolPropertyValue(pathSubscriptionsDisablePolling),
		Approval: &ApprovalConfig{
			SubscriptionApprovalMode: props.StringPropertyValue(pathSubscriptionsApprovalMode),
			SubscriptionApprovalWebhook: &WebhookConfiguration{
				URL:     props.StringPropertyValue(pathSubscriptionsApprovalWebhookURL),
				Headers: props.StringPropertyValue(pathSubscriptionsApprovalWebhookHeaders),
				Secret:  props.StringPropertyValue(pathSubscriptionsApprovalWebhookSecret),
			},
		},
		Notifications: &NotificationConfig{
			Webhook: &WebhookConfiguration{
				URL:     props.StringPropertyValue(pathSubscriptionsNotificationsWebhookURL),
				Headers: props.StringPropertyValue(pathSubscriptionsNotificationsWebhookHeaders),
			},
			SMTP: &smtp{
				Host:     props.StringPropertyValue(pathSubscriptionsNotificationsSMTPHost),
				Port:     props.IntPropertyValue(pathSubscriptionsNotificationsSMTPPort),
				From:     props.StringPropertyValue(pathSubscriptionsNotificationsSMTPFrom),
				AuthType: authType,
				Identity: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPIdentity),
				Username: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUserName),
				Password: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUserPassword),
				Subscribe: &EmailTemplate{
					Subject: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeSubject),
					Body:    props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeBody),
					Oauth:   props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeOauth),
					APIKey:  props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeAPIKeys),
				},
				Unsubscribe: &EmailTemplate{
					Subject: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUnsubscribeSubject),
					Body:    props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUnsubscribeBody),
				},
				SubscribeFailed: &EmailTemplate{
					Subject: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeFailedSubject),
					Body:    props.StringPropertyValue(pathSubscriptionsNotificationsSMTPSubscribeFailedBody),
				},
				UnsubscribeFailed: &EmailTemplate{
					Subject: props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUnsubscribeFailedSubject),
					Body:    props.StringPropertyValue(pathSubscriptionsNotificationsSMTPUnsubscribeFailedBody),
				},
			},
		},
	}

	return cfg
}

// NewSubscriptionConfig - Creates the default subscription config
func NewSubscriptionConfig() SubscriptionConfig {
	return &SubscriptionConfiguration{
		Approval: &ApprovalConfig{
			SubscriptionApprovalMode:    ManualApproval,
			SubscriptionApprovalWebhook: NewWebhookConfig(),
		},
		Notifications: &NotificationConfig{
			Webhook: NewWebhookConfig(),
			SMTP:    &smtp{},
		},
	}
}

// PollingEnabled - return true when the polling of subscriptsion should be ran on this agent
func (s *SubscriptionConfiguration) PollingEnabled() bool {
	return !s.DisablePolling
}

// SetNotificationType -
func (s *SubscriptionConfiguration) SetNotificationType(notificationType NotificationType) {
	s.Types = append(s.Types, notificationType)
}

// GetNotificationTypes -
func (s *SubscriptionConfiguration) GetNotificationTypes() []NotificationType {
	return s.Types
}

// GetWebhookURL - Returns the webhook url for notifications
func (s *SubscriptionConfiguration) GetWebhookURL() string {
	if s.Notifications.Webhook != nil {
		return s.Notifications.Webhook.GetURL()
	}
	return ""
}

// GetWebhookHeaders - Returns the notification headers
func (s *SubscriptionConfiguration) GetWebhookHeaders() map[string]string {
	if s.Notifications.Webhook != nil {
		return s.Notifications.Webhook.GetWebhookHeaders()
	}
	return make(map[string]string)
}

// GetSMTPURL - Returns the URL for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPURL() string {
	if s.Notifications.SMTP != nil {
		return fmt.Sprintf("%s:%d", s.Notifications.SMTP.Host, s.Notifications.SMTP.Port)
	}
	return ""
}

// GetSMTPHost - Returns the Host for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPHost() string {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Host
	}
	return ""
}

// GetSMTPFromAddress -
func (s *SubscriptionConfiguration) GetSMTPFromAddress() string {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.From
	}
	return ""
}

// GetSMTPAuthType -
func (s *SubscriptionConfiguration) GetSMTPAuthType() SMTPAuthType {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.AuthType
	}
	return ""
}

// GetSMTPIdentity -
func (s *SubscriptionConfiguration) GetSMTPIdentity() string {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Identity
	}
	return ""
}

// GetSMTPUsername -
func (s *SubscriptionConfiguration) GetSMTPUsername() string {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Username
	}
	return ""
}

// GetSMTPPassword -
func (s *SubscriptionConfiguration) GetSMTPPassword() string {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Password
	}
	return ""
}

// GetSubscribeTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeTemplate() *EmailTemplate {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Subscribe
	}
	return nil
}

// GetUnsubscribeTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeTemplate() *EmailTemplate {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.Unsubscribe
	}
	return nil
}

// GetSubscribeFailedTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeFailedTemplate() *EmailTemplate {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.SubscribeFailed
	}
	return nil
}

// GetUnsubscribeFailedTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeFailedTemplate() *EmailTemplate {
	if s.Notifications.SMTP != nil {
		return s.Notifications.SMTP.UnsubscribeFailed
	}
	return nil
}

// GetSubscriptionApprovalMode - Returns the subscription approval mode
func (s *SubscriptionConfiguration) GetSubscriptionApprovalMode() string {
	return s.Approval.SubscriptionApprovalMode
}

// GetSubscriptionApprovalWebhookConfig - Returns the Config for the subscription webhook
func (s *SubscriptionConfiguration) GetSubscriptionApprovalWebhookConfig() WebhookConfig {
	return s.Approval.SubscriptionApprovalWebhook
}

// ValidateCfg - Validates the config, implementing IConfigInterface
func (s *SubscriptionConfiguration) ValidateCfg() error {
	if s.Notifications.Webhook.GetURL() != "" {
		s.SetNotificationType(NotifyWebhook)
		log.Debug("Webhook notification set")
		err := s.validateWebhook()
		if err != nil {
			return err
		}
	}
	if s.Notifications.SMTP.Host != "" {
		s.SetNotificationType(NotifySMTP)
		log.Debug("SMTP notification set")
	}

	switch s.GetSubscriptionApprovalMode() {
	case ManualApproval, AutoApproval, WebhookApproval:
		// these are all OK
	case "":
	default:
		return ErrBadConfig.FormatError(pathSubscriptionsApprovalMode)
	}

	log.Debugf("Approval mode set: %s", s.GetSubscriptionApprovalMode())

	// only validate the webhook approval config settings if the approval mode is for webhook
	if s.GetSubscriptionApprovalMode() == WebhookApproval {
		err := s.Approval.SubscriptionApprovalWebhook.ValidateConfig()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SubscriptionConfiguration) validateWebhook() error {
	if webhookURL := s.GetWebhookURL(); webhookURL != "" {
		if _, err := url.ParseRequestURI(webhookURL); err != nil {
			return errors.New("central.subscriptions.notifications.webhook is not a valid URL")
		}
	}

	// Header=contentType,Value=application/json, Header=Elements-Formula-Instance-Id,Value=440874, Header=Authorization,Value=User F+rYQSfu0w5yIa5q7uNs2MKYcIok8pYpgAUwJtXFnzc=, Organization a1713018bbde8f54f4f55ff8c3bd8bfe
	webhookConfig := s.Notifications.Webhook.(*WebhookConfiguration)
	webhookConfig.webhookHeaders = map[string]string{}

	// webhook headers for subscription notification cannot be empty
	if webhookConfig.Headers == "" {
		return errors.New("central.subscriptions.notifications.headers cannot be empty")
	}
	webhookConfig.Headers = strings.Replace(webhookConfig.Headers, ", ", ",", -1)
	headersValues := strings.Split(webhookConfig.Headers, ",Header=")
	for _, headerValue := range headersValues {
		hvArray := strings.Split(headerValue, ",Value=")
		if len(hvArray) != 2 {
			return errors.New("could not parse value of central.subscriptions.notifications.headers")
		}
		hvArray[0] = strings.TrimLeft(hvArray[0], "Header=") // handle the first	header in the list
		webhookConfig.webhookHeaders[hvArray[0]] = hvArray[1]
	}

	return nil
}
