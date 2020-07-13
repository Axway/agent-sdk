package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/properties"
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
	GetNotificationType() NotificationType
	GetNotificationWebhook() string
	GetNotificationHeaders() map[string]string
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
}

// SubscriptionConfiguration - Structure to hold the subscription config
type SubscriptionConfiguration struct {
	SubscriptionConfig
	SMTP    *smtp    `config:"smtp"`
	Webhook *webhook `config:"webhook"`
	Type    NotificationType
}

// These constants are the paths that the settings is at in a config file
const (
	webhookURL     = "subscriptions.webhook.url"
	webhookHeaders = "subscriptions.webhook.headers"
	smtpFrom       = "subscriptions.smtp.fromAddress"
	smtpAuthType   = "subscriptions.smtp.authType"
	smtpIdentity   = "subscriptions.smtp.identity"
)

// AddSubscriptionsConfigProperties -
func AddSubscriptionsConfigProperties(cmdProps properties.Properties) {
	cmdProps.AddStringProperty("subscription.webhook.url", "", "desc")
	cmdProps.AddStringProperty("subscription.webhook.headers", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.host", "", "desc")
	cmdProps.AddIntProperty("subscription.smtp.port", 0, "desc")
	cmdProps.AddStringProperty(smtpFrom, "", "desc")
	cmdProps.AddStringProperty(smtpAuthType, string(NoAuth), "desc")
	cmdProps.AddStringProperty(smtpIdentity, "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribe.subject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribe.body", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribe.subject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribe.body", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribeFailed.subject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribeFailed.body", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribeFailed.subject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribeFailed.body", "", "desc")
}

type webhook struct {
	URL     string `config:"webhook.url"`
	Headers string `config:"webhook.headers"`
	headers map[string]string
}

type smtp struct {
	Host              string         `config:"smtp.host"`
	Port              int            `config:"smtp.port"`
	From              string         `config:"smtp.fromAddress"`
	AuthType          SMTPAuthType   `config:"smtp.authType"`
	Identity          string         `config:"smtp.identity"`
	Username          string         `config:"smtp.username"`
	Password          string         `config:"smtp.password"`
	Subscribe         *EmailTemplate `config:"smtp.subscribe"`
	Unsubscribe       *EmailTemplate `config:"smtp.unsubscribe"`
	SubscribeFailed   *EmailTemplate `config:"smtp.subscribeFailed"`
	UnsubscribeFailed *EmailTemplate `config:"smtp.unsubscribeFailed"`
}

//EmailTemplate -
type EmailTemplate struct {
	Subject string `config:"subject"`
	Body    string `config:"body"`
}

// ParseSubscriptionConfig -
func ParseSubscriptionConfig(cmdProps properties.Properties) (SubscriptionConfig, error) {
	// Determine the auth type
	authTypeString := cmdProps.StringPropertyValue(smtpAuthType)
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
		Webhook: &webhook{
			URL:     cmdProps.StringPropertyValue("subscriptions.webhook.url"),
			Headers: cmdProps.StringPropertyValue("subscriptions.webhook.headers"),
		},
		SMTP: &smtp{
			Host:     cmdProps.StringPropertyValue("subscriptions.smtp.host"),
			Port:     cmdProps.IntPropertyValue("subscriptions.smtp.port"),
			From:     cmdProps.StringPropertyValue(smtpFrom),
			AuthType: authType,
			Identity: cmdProps.StringPropertyValue(smtpIdentity),
			Username: cmdProps.StringPropertyValue("subscriptions.smtp.username"),
			Password: cmdProps.StringPropertyValue("subscriptions.smtp.password"),
			Subscribe: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("subscriptions.smtp.subscribe.subject"),
				Body:    cmdProps.StringPropertyValue("subscriptions.smtp.subscribe.body"),
			},
			Unsubscribe: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("subscriptions.smtp.unsubscribe.subject"),
				Body:    cmdProps.StringPropertyValue("subscriptions.smtp.unsubscribe.body"),
			},
			SubscribeFailed: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("subscriptions.smtp.subscribeFailed.subject"),
				Body:    cmdProps.StringPropertyValue("subscriptions.smtp.subscribeFailed.body"),
			},
			UnsubscribeFailed: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("subscriptions.smtp.unsubscribeFailed.subject"),
				Body:    cmdProps.StringPropertyValue("subscriptions.smtp.unsubscribeFailed.body"),
			},
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// NewSubscriptionConfig - Creates the default subscription config
func NewSubscriptionConfig() SubscriptionConfig {
	return &SubscriptionConfiguration{}
}

// SetNotificationType -
func (s *SubscriptionConfiguration) SetNotificationType(notificationType NotificationType) {
	s.Type = notificationType
}

// GetNotificationType -
func (s *SubscriptionConfiguration) GetNotificationType() NotificationType {
	return s.Type
}

// GetNotificationWebhook - Returns the webhook url for notifications
func (s *SubscriptionConfiguration) GetNotificationWebhook() string {
	return s.Webhook.URL
}

// GetNotificationHeaders - Returns the notification headers
func (s *SubscriptionConfiguration) GetNotificationHeaders() map[string]string {
	return s.Webhook.headers
}

// GetSMTPURL - Returns the URL for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPURL() string {
	return fmt.Sprintf("%s:%d", s.SMTP.Host, s.SMTP.Port)
}

// GetSMTPHost - Returns the Host for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPHost() string {
	return s.SMTP.Host
}

// GetSMTPFromAddress -
func (s *SubscriptionConfiguration) GetSMTPFromAddress() string {
	return s.SMTP.From
}

// GetSMTPAuthType -
func (s *SubscriptionConfiguration) GetSMTPAuthType() SMTPAuthType {
	return s.SMTP.AuthType
}

// GetSMTPIdentity -
func (s *SubscriptionConfiguration) GetSMTPIdentity() string {
	return s.SMTP.Identity
}

// GetSMTPUsername -
func (s *SubscriptionConfiguration) GetSMTPUsername() string {
	return s.SMTP.Username
}

// GetSMTPPassword -
func (s *SubscriptionConfiguration) GetSMTPPassword() string {
	return s.SMTP.Password
}

// GetSubscribeTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeTemplate() *EmailTemplate {
	return s.SMTP.Subscribe
}

// GetUnsubscribeTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeTemplate() *EmailTemplate {
	return s.SMTP.Unsubscribe
}

// GetSubscribeFailedTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeFailedTemplate() *EmailTemplate {
	return s.SMTP.SubscribeFailed
}

// GetUnsubscribeFailedTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeFailedTemplate() *EmailTemplate {
	return s.SMTP.UnsubscribeFailed
}

func (s *SubscriptionConfiguration) validate() error {
	if s.Webhook.URL != "" {
		s.SetNotificationType(NotifyWebhook)
		if webhookURL := s.GetNotificationWebhook(); webhookURL != "" {
			if _, err := url.ParseRequestURI(webhookURL); err != nil {
				return errors.New("Error central.subscriptions.notificationWebhook nota valid URL")
			}
		}

		// Header=contentType,Value=application/json, Header=Elements-Formula-Instance-Id,Value=440874, Header=Authorization,Value=User F+rYQSfu0w5yIa5q7uNs2MKYcIok8pYpgAUwJtXFnzc=, Organization a1713018bbde8f54f4f55ff8c3bd8bfe
		s.Webhook.headers = map[string]string{}
		s.Webhook.Headers = strings.Replace(s.Webhook.Headers, ", ", ",", -1)
		headersValues := strings.Split(s.Webhook.Headers, ",Header=")
		for _, headerValue := range headersValues {
			hvArray := strings.Split(headerValue, ",Value=")
			if len(hvArray) != 2 {
				return errors.New("Could not parse value of central.subscriptions.notificationHeaders")
			}
			hvArray[0] = strings.TrimLeft(hvArray[0], "Header=") // handle the first	header in the list
			s.Webhook.headers[hvArray[0]] = hvArray[1]
		}
		return nil
	}
	if s.SMTP.Host != "" {
		s.SetNotificationType(NotifySMTP)
		return nil
	}

	return nil
}
