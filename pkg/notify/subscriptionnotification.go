package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/apic"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

//TODO
/*
	1. Search for comment "DEPRECATED to be removed on major release"
	2. Remove deprecated code left from APIGOV-19751
*/

//DEPRECATED to be removed on major release - this map will no longer be needed after "${tag} is invalid"
// subNotifTemplateMap - map of date formats for apiservicerevision title
var subNotifTemplateMap = map[string]string{
	"${catalogItemUrl}":  "{{.CatalogItemURL}}",
	"${catalogItemName}": "{{.CatalogItemName}}",
	"${catalogItemId}":   "{{.CatalogItemID}}",
	"${keyHeaderName}":   "{{.KeyHeaderName}}",
	"${key}":             "{{.Key}}",
	"${clientID}":        "{{.ClientID}}",
	"${clientSecret}":    "{{.ClientSecret}}",
	"${action}":          "{{.Action}}",
	"${email}":           "{{.Email}}",
	"${authtemplate}":    "{{.AuthTemplate}}",
}

//SubscriptionNotification - the struct that is sent to the notification and used to fill in email templates
type SubscriptionNotification struct {
	CatalogItemID   string                 `json:"catalogItemId"`
	CatalogItemURL  string                 `json:"catalogItemUrl"`
	CatalogItemName string                 `json:"catalogItemName"`
	Action          apic.SubscriptionState `json:"action"`
	Email           string                 `json:"email,omitempty"`
	Message         string                 `json:"message,omitempty"`
	Key             string                 `json:"key,omitempty"`
	KeyHeaderName   string                 `json:"keyHeaderName,omitempty"`
	ClientID        string                 `json:"clientID,omitempty"`
	ClientSecret    string                 `json:"clientSecret,omitempty"`
	AuthTemplate    string                 `json:"authtemplate,omitempty"`
	IsAPIKey        bool                   `json:"isAPIKey,omitempty"`
	apiClient       coreapi.Client
}

//NewSubscriptionNotification - creates a new subscription notification object
func NewSubscriptionNotification(recipient, message string, state apic.SubscriptionState) *SubscriptionNotification {
	subscriptionNotification := &SubscriptionNotification{
		Email:     recipient,
		Action:    state,
		Message:   message,
		apiClient: coreapi.NewClient(corecfg.NewTLSConfig(), ""),
	}

	return subscriptionNotification
}

// consts
const (
	Apikeys = "apikeys"
	Oauth   = "oauth"
)

// SetCatalogItemInfo - Set the catalogitem info
func (s *SubscriptionNotification) SetCatalogItemInfo(catalogID, catalogName, catalogItemURL string) {
	s.CatalogItemID = catalogID
	s.CatalogItemName = catalogName
	s.CatalogItemURL = catalogItemURL
}

// SetAPIKeyInfo - Set the key and header
func (s *SubscriptionNotification) SetAPIKeyInfo(key, keyHeaderName string) {
	s.Key = key
	s.KeyHeaderName = keyHeaderName
}

// SetOauthInfo - Set the id and secret info
func (s *SubscriptionNotification) SetOauthInfo(clientID, clientSecret string) {
	s.ClientID = clientID
	s.ClientSecret = clientSecret
}

// SetAuthorizationTemplate - Set the authtemplate in the config central.subscriptions.notifications.smtp.subscribe.body {authtemplate}
func (s *SubscriptionNotification) SetAuthorizationTemplate(authType string) {
	if authType == "" {
		log.Debug("Subscription notification configuration for authorization type is not set")
		return
	}

	template := templateActionMap[s.Action]
	if template == nil {
		log.Error(ErrSubscriptionNoTemplateForAction.FormatError(s.Action))
		return
	}

	//DEPRECATED to be removed on major release - setting s.AuthTemplate will no longer be needed after "${tag} is invalid"
	switch authType {
	case Apikeys:
		s.AuthTemplate = template.APIKey
		s.IsAPIKey = true
	case Oauth:
		s.AuthTemplate = template.Oauth
		s.IsAPIKey = false
	default:
		log.Error(ErrSubscriptionBadAuthtype.FormatError(authType))
		return
	}

	log.Debugf("Subscription notification configuration for '{authtemplate}' is set to %s", authType)
}

// NotifySubscriber - send a notification to any configured notification type
func (s *SubscriptionNotification) NotifySubscriber(recipient string) error {
	var notificationSent bool
	for _, notificationType := range globalCfg.GetNotificationTypes() {
		log.Debugf("Attempt to notify using %s", notificationType)
		switch notificationType {
		case corecfg.NotifyWebhook:
			err := s.notifyViaWebhook()
			if err != nil {
				return utilerrors.Wrap(ErrSubscriptionNotification, err.Error()).FormatError("webhook")
			}
			notificationSent = true
			log.Debugf("Webhook notification sent to %s.", recipient)

		case corecfg.NotifySMTP:
			log.Info("Sending subscription email to subscriber.")
			err := s.notifyViaSMTP()
			if err != nil {
				return utilerrors.Wrap(ErrSubscriptionNotification, err.Error()).FormatError("smtp")
			}
			notificationSent = true
			log.Debugf("Email notification sent to %s.", recipient)
		}
	}

	if !notificationSent {
		return ErrSubscriptionNoNotifications
	}

	return nil

}

func (s *SubscriptionNotification) notifyViaWebhook() error {
	buffer, err := json.Marshal(&s)
	if err != nil {
		return utilerrors.Wrap(ErrSubscriptionData, err.Error())
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     globalCfg.GetWebhookURL(),
		Headers: globalCfg.GetWebhookHeaders(),
		Body:    buffer,
	}

	_, err = s.apiClient.Send(request)
	if err != nil {
		return err
	}

	return nil
}

func (s *SubscriptionNotification) notifyViaSMTP() error {
	template := templateActionMap[s.Action]
	if template == nil {
		return fmt.Errorf("no template found for action %s", s.Action)
	}

	if template.Subject == "" && template.Body == "" {
		return fmt.Errorf("template subject and body not found for action %s", s.Action)
	}

	// determine the auth type to use
	var auth sasl.Client
	log.Debugf("SMTP authorization type %s", globalCfg.GetSMTPAuthType())

	switch globalCfg.GetSMTPAuthType() {
	case (corecfg.LoginAuth):
		auth = sasl.NewLoginClient(globalCfg.GetSMTPUsername(), globalCfg.GetSMTPPassword())
	case (corecfg.PlainAuth):
		auth = sasl.NewPlainClient(globalCfg.GetSMTPIdentity(), globalCfg.GetSMTPUsername(), globalCfg.GetSMTPPassword())
	case (corecfg.AnonymousAuth):
		auth = sasl.NewAnonymousClient(globalCfg.GetSMTPFromAddress())
	}

	msg := s.BuildSMTPMessage(template)
	err := smtp.SendMail(globalCfg.GetSMTPURL(), auth, globalCfg.GetSMTPFromAddress(), []string{s.Email}, msg)
	if err != nil {
		log.Error(utilerrors.Wrap(ErrSubscriptionSendEmail, err.Error()))
		return err
	}
	return nil
}

// BuildSMTPMessage -
func (s *SubscriptionNotification) BuildSMTPMessage(template *corecfg.EmailTemplate) *strings.Reader {
	mime := mimeMap{
		"MIME-version": "1.0",
		"Content-Type": "text/html",
		"charset":      "UTF-8",
	}

	fromAddress := fmt.Sprintf("From: %s", globalCfg.GetSMTPFromAddress())
	toAddress := fmt.Sprintf("To: %s", s.Email)
	subject := fmt.Sprintf("Subject: %s", template.Subject)

	log.Debugf("Sending email %s, %s, %s", fromAddress, toAddress, subject)

	//DEPRECATED to be removed on major release - this check for '${"' will no longer be needed after "${tag} is invalid"

	// Verify if customer is still using "${tag}" teamplate.  Warn them that it is going to be deprecated
	// Transform the old "${tag}" to the go template {{.Tag}}
	if strings.Contains(template.Body, "${") {
		log.Warnf("Using '${tag}' as part of CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP is being deprecated soon. Please start using '{{.Tag}}")
		// Create template.Body using the old style Body and AuthTemplate
		deprecatedEmailTemplate := s.UpdateTemplate(fmt.Sprintf("%s. </br>%s", template.Body, s.AuthTemplate))
		log.Warnf("Transforming deprecated email template to : %s", deprecatedEmailTemplate)
		template.Body = deprecatedEmailTemplate
	}

	emailBody := s.setEmailBodyTemplate(template.Body)

	msgArray := []string{
		fromAddress,
		toAddress,
		subject,
		mime.String(),
		emailBody,
	}

	return strings.NewReader(strings.Join(msgArray, "\n"))
}

//UpdateTemplate - update ${tag} to {{.Tag}}.  ${tag} to be deprecated
func (s *SubscriptionNotification) UpdateTemplate(template string) string {

	for k, v := range subNotifTemplateMap {
		template = strings.ReplaceAll(template, k, v)
	}

	return template
}

// setEmailBodyTemplate - set email body using Go template
func (s *SubscriptionNotification) setEmailBodyTemplate(body string) string {
	subNotifTemplate := SubscriptionNotification{
		s.CatalogItemID,
		s.CatalogItemURL,
		s.CatalogItemName,
		s.Action,
		s.Email,
		s.Message,
		s.Key,
		s.KeyHeaderName,
		s.ClientID,
		s.ClientSecret,
		s.AuthTemplate,
		s.IsAPIKey,
		s.apiClient,
	}

	c, err := template.New("catalogTemplate").Parse(body)
	if err != nil {
		log.Error("Could not render catalog item info : ", err)
	}

	var catalogItem bytes.Buffer

	err = c.Execute(&catalogItem, subNotifTemplate)
	if err != nil {
		log.Error("Could not render catalog item info : ", err)
	}

	return catalogItem.String()
}
