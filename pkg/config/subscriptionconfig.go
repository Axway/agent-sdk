package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
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
}

// SubscriptionConfiguration - Structure to hold the subscription config
type SubscriptionConfiguration struct {
	SubscriptionConfig
	SMTP    *smtp    `config:"smtp"`
	Webhook *webhook `config:"webhook"`
	Types   []NotificationType
}

// These constants are the paths that the settings is at in a config file
const (
	smtpFrom     = "central.subscriptions.notifications.smtp.fromAddress"
	smtpAuthType = "central.subscriptions.notifications.smtp.authType"
	smtpIdentity = "central.subscriptions.notifications.smtp.identity"
)

type webhook struct {
	URL                 string `config:"webhook.url"`
	Headers             string `config:"webhook.headers"`
	notificationHeaders map[string]string
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
	Oauth   string `config:"oauth"`
	APIKey  string `config:"apikeys"`
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
			URL:     cmdProps.StringPropertyValue("central.subscriptions.notifications.webhook.url"),
			Headers: cmdProps.StringPropertyValue("central.subscriptions.notifications.webhook.headers"),
		},
		SMTP: &smtp{
			Host:     cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.host"),
			Port:     cmdProps.IntPropertyValue("central.subscriptions.notifications.smtp.port"),
			From:     cmdProps.StringPropertyValue(smtpFrom),
			AuthType: authType,
			Identity: cmdProps.StringPropertyValue(smtpIdentity),
			Username: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.username"),
			Password: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.password"),
			Subscribe: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribe.subject"),
				Body:    cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribe.body"),
				Oauth:   cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribe.oauth"),
				APIKey:  cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribe.apikeys"),
			},
			Unsubscribe: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.unsubscribe.subject"),
				Body:    cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.unsubscribe.body"),
			},
			SubscribeFailed: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribeFailed.subject"),
				Body:    cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.subscribeFailed.body"),
			},
			UnsubscribeFailed: &EmailTemplate{
				Subject: cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.unsubscribeFailed.subject"),
				Body:    cmdProps.StringPropertyValue("central.subscriptions.notifications.smtp.unsubscribeFailed.body"),
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
	return &SubscriptionConfiguration{
		Webhook: &webhook{},
		SMTP:    &smtp{},
	}
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
	if s.Webhook != nil {
		return s.Webhook.URL
	}
	return ""
}

// GetWebhookHeaders - Returns the notification headers
func (s *SubscriptionConfiguration) GetWebhookHeaders() map[string]string {
	if s.Webhook != nil {
		return s.Webhook.notificationHeaders
	}
	return make(map[string]string)
}

// GetSMTPURL - Returns the URL for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPURL() string {
	if s.SMTP != nil {
		return fmt.Sprintf("%s:%d", s.SMTP.Host, s.SMTP.Port)
	}
	return ""
}

// GetSMTPHost - Returns the Host for the SMTP server
func (s *SubscriptionConfiguration) GetSMTPHost() string {
	if s.SMTP != nil {
		return s.SMTP.Host
	}
	return ""
}

// GetSMTPFromAddress -
func (s *SubscriptionConfiguration) GetSMTPFromAddress() string {
	if s.SMTP != nil {
		return s.SMTP.From
	}
	return ""
}

// GetSMTPAuthType -
func (s *SubscriptionConfiguration) GetSMTPAuthType() SMTPAuthType {
	if s.SMTP != nil {
		return s.SMTP.AuthType
	}
	return ""
}

// GetSMTPIdentity -
func (s *SubscriptionConfiguration) GetSMTPIdentity() string {
	if s.SMTP != nil {
		return s.SMTP.Identity
	}
	return ""
}

// GetSMTPUsername -
func (s *SubscriptionConfiguration) GetSMTPUsername() string {
	if s.SMTP != nil {
		return s.SMTP.Username
	}
	return ""
}

// GetSMTPPassword -
func (s *SubscriptionConfiguration) GetSMTPPassword() string {
	if s.SMTP != nil {
		return s.SMTP.Password
	}
	return ""
}

// GetSubscribeTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeTemplate() *EmailTemplate {
	if s.SMTP != nil {
		return s.SMTP.Subscribe
	}
	return nil
}

// GetUnsubscribeTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeTemplate() *EmailTemplate {
	if s.SMTP != nil {
		return s.SMTP.Unsubscribe
	}
	return nil
}

// GetSubscribeFailedTemplate - returns the email template info for a subscribe
func (s *SubscriptionConfiguration) GetSubscribeFailedTemplate() *EmailTemplate {
	if s.SMTP != nil {
		return s.SMTP.SubscribeFailed
	}
	return nil
}

// GetUnsubscribeFailedTemplate - returns the email template info for an unsubscribe
func (s *SubscriptionConfiguration) GetUnsubscribeFailedTemplate() *EmailTemplate {
	if s.SMTP != nil {
		return s.SMTP.UnsubscribeFailed
	}
	return nil
}

func (s *SubscriptionConfiguration) validate() error {
	if s.Webhook.URL != "" {
		s.SetNotificationType(NotifyWebhook)
		log.Debug("Webhook notification set")
		err := s.validateWebhook()
		if err != nil {
			return err
		}
	}
	if s.SMTP.Host != "" {
		s.SetNotificationType(NotifySMTP)
		log.Debug("SMTP notification set")
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
	s.Webhook.notificationHeaders = map[string]string{}
	s.Webhook.Headers = strings.Replace(s.Webhook.Headers, ", ", ",", -1)
	headersValues := strings.Split(s.Webhook.Headers, ",Header=")
	for _, headerValue := range headersValues {
		hvArray := strings.Split(headerValue, ",Value=")
		if len(hvArray) != 2 {
			return errors.New("could not parse value of central.subscriptions.notifications.headers")
		}
		hvArray[0] = strings.TrimLeft(hvArray[0], "Header=") // handle the first	header in the list
		s.Webhook.notificationHeaders[hvArray[0]] = hvArray[1]
	}

	return nil
}
