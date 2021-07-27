package notify

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"${message}":         "{{.Messaage}}",
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

	msg, err := s.BuildSMTPMessage(template)
	if err != nil {
		return err
	}

	err = smtp.SendMail(globalCfg.GetSMTPURL(), auth, globalCfg.GetSMTPFromAddress(), []string{s.Email}, msg)
	if err != nil {
		log.Error(utilerrors.Wrap(ErrSubscriptionSendEmail, err.Error()))
		return err
	}
	return nil
}

// BuildSMTPMessage -
func (s *SubscriptionNotification) BuildSMTPMessage(template *corecfg.EmailTemplate) (*strings.Reader, error) {
	mime := mimeMap{
		"MIME-version": "1.0",
		"Content-Type": "text/html",
		"charset":      "UTF-8",
	}

	fromAddress := fmt.Sprintf("From: %s", globalCfg.GetSMTPFromAddress())
	toAddress := fmt.Sprintf("To: %s", s.Email)
	subject := fmt.Sprintf("Subject: %s", template.Subject)

	log.Debugf("Sending email %s, %s, %s", fromAddress, toAddress, subject)

	// Shouldn't have to check error from ValidateSubscriptionConfig since startup passed the subscription validation check
	emailBody, err := s.ValidateSubscriptionConfig(template.Body, s.AuthTemplate)
	if err != nil {
		return nil, err
	}

	msgArray := []string{
		fromAddress,
		toAddress,
		subject,
		mime.String(),
		emailBody,
	}

	return strings.NewReader(strings.Join(msgArray, "\n")), nil
}

// ValidateSubscriptionConfig - validate body and auth template tags
func (s *SubscriptionNotification) ValidateSubscriptionConfig(body, authTemplate string) (string, error) {
	//DEPRECATED to be removed on major release - this check for '${"' will no longer be needed after "${tag} is invalid"

	// Verify if customer is still using "${tag}" teamplate.  Warn them that it is going to be deprecated
	// Transform the old "${tag}" to the go template {{.Tag}}
	if strings.Contains(body, "${") {
		log.Warnf("Using '${tag}' as part of CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP is deprecated. Please refer to docs.axway to start using '{{.Tag}}")
		// update body using the old style Body concat with AuthTemplate
		body = s.updateTemplate(fmt.Sprintf("%s. </br>%s", body, authTemplate))
	} // else customer is using the {{.Tag}} and therefore the body should already contain the authTemplate in the case of SUBSCRIBE

	return s.setEmailBodyTemplate(body)

}

//updateTemplate - update ${tag} to {{.Tag}}.  ${tag} to be deprecated
func (s *SubscriptionNotification) updateTemplate(template string) string {

	for k, v := range subNotifTemplateMap {
		template = strings.ReplaceAll(template, k, v)
	}

	return template
}

// setEmailBodyTemplate - set email body using Go template
func (s *SubscriptionNotification) setEmailBodyTemplate(body string) (string, error) {
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
		return "", errors.New(err.Error())
	}

	var catalogItem bytes.Buffer

	err = c.Execute(&catalogItem, subNotifTemplate)

	// Errors are returned in the following format
	// "template: catalogTemplate:1:63: executing "catalogTemplate" at <.CatffalogItemURL>: can't evaluate field CatffalogItemURL in type notify.SubscriptionNotification"
	// "template: catalogTemplate:1:207: executing "catalogTemplate" at <.KeyHeafederName>: can't evaluate field KeyHeafederName in type notify.SubscriptionNotification"
	if err != nil {
		// attempt to grab error returned from .Execute() beginning, "can't evaluate"
		errString := err.Error()
		index := strings.Index(errString, "can't evaluate")
		if index > 0 {
			errString = string(err.Error()[index:len(err.Error())])
		}

		return "", errors.New(errString)
	}

	return catalogItem.String(), nil
}
