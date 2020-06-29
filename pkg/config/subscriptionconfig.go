package config

import (
	"fmt"
	"strings"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cmd/properties"
)

// NotificationType - Type definition for subscription state
type NotificationType string

// NotificationTypes
const (
	NotifySMTP = NotificationType("SMTP")
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
	SMTP *smtp `config:"smtp"`
	Type NotificationType
}

// These constants are the paths that the settings is at in a config file
const (
	smtpFrom     = "subscriptions.smtp.fromAddress"
	smtpAuthType = "subscriptions.smtp.authType"
	smtpIdentity = "subscriptions.smtp.identity"
)

// AddSubscriptionsConfigProperties -
func AddSubscriptionsConfigProperties(cmdProps properties.Properties) {
	cmdProps.AddStringProperty("subscription.smtp.host", "subscriptionSMTPHost", "", "desc")
	cmdProps.AddIntProperty("subscription.smtp.port", "subscriptionSMTPPort", 0, "desc")
	cmdProps.AddStringProperty(smtpFrom, "subscriptionSMTPFromAddress", "", "desc")
	cmdProps.AddStringProperty(smtpAuthType, "subscriptionSMTPAuthType", string(NoAuth), "desc")
	cmdProps.AddStringProperty(smtpIdentity, "subscriptionSMTPIdentity", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribe.subject", "subscriptionSubscribeSubject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribe.body", "subscriptionSubscribeBody", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribe.subject", "subscriptionUnsubscribeSubject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribe.body", "subscriptionUnsubscribeBody", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribeFailed.subject", "subscriptionSubscribeFailedSubject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.subscribeFailed.body", "subscriptionSubscribeFailedBody", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribeFailed.subject", "subscriptionUnsubscribeFailedSubject", "", "desc")
	cmdProps.AddStringProperty("subscription.smtp.unsubscribeFailed.body", "subscriptionUnsubscribeFailedBody", "", "desc")
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
	if s.SMTP.Host != "" {
		s.SetNotificationType(NotifySMTP)
		// Check values by auth type
		//check host set
		//check port set

		//check all subjects/templates have only variables expected
	}

	return nil
}
