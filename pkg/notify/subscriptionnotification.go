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
	AuthType        string                 `json:"authType,omitempty"`
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

	switch authType {
	case Apikeys:
		s.AuthTemplate = s.UpdateTemplate(template.APIKey)
		s.AuthType = Apikeys
	case Oauth:
		s.AuthTemplate = s.UpdateTemplate(template.Oauth)
		s.AuthType = Oauth
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
	subject := fmt.Sprintf("Subject: %s", s.UpdateTemplate(template.Subject))

	log.Debugf("Sending email %s, %s, %s", fromAddress, toAddress, subject)

	emailBody := s.UpdateTemplate(template.Body)
	if strings.Contains(template.Body, "$") {
		log.Warnf("Using '${tag}' as part of CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP is deprecated. Please start using '{{.Tag}}")
	} else {
		emailBody = s.setEmailBodyTemplate(template.Body)
	}

	msgArray := []string{
		fromAddress,
		toAddress,
		subject,
		mime.String(),
		emailBody,
	}

	return strings.NewReader(strings.Join(msgArray, "\n"))
}

//UpdateTemplate -
func (s *SubscriptionNotification) UpdateTemplate(template string) string {
	var jsonMap map[string]string
	data, _ := json.Marshal(s)

	json.Unmarshal(data, &jsonMap)

	for k, v := range jsonMap {
		template = strings.Replace(template, fmt.Sprintf("${%s}", k), v, -1)
		template = strings.Replace(template, fmt.Sprintf("{{%s}}", k), v, -1)
	}

	return template
}

// setEmailBodyTemplate - set email body using Go template
func (s *SubscriptionNotification) setEmailBodyTemplate(body string) string {
	isAPIKey := s.AuthType == Apikeys

	// create emailBodyTemplate
	emailBodyTemplate := EmailBody{
		s.CatalogItemURL,
		s.CatalogItemName,
		s.CatalogItemID,
		s.KeyHeaderName,
		s.Key,
		isAPIKey,
		s.ClientID,
		s.ClientSecret,
	}

	c, err := template.New("catalogTemplate").Parse(body)
	if err != nil {
		log.Error("Could not render catalog item info : ", err)
	}

	var catalogItem bytes.Buffer

	err = c.Execute(&catalogItem, emailBodyTemplate)
	if err != nil {
		log.Error("Could not render catalog item info : ", err)
	}

	return catalogItem.String()
}

// CatalogItem - for template use
type EmailBody struct {
	CatalogItemURL  string
	CatalogItemName string
	CatalogItemID   string
	KeyHeaderName   string
	Key             string
	IsAPIKey        bool
	ClientID        string
	ClientSecret    string
}

/*
Subscription Body templates

CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP_SUBSCRIBE_BODY=
Subscription created for Catalog Item:  <a href= {{.CatalogItemURL}}> {{.CatalogItemName}} {{.CatalogItemID}}</a>
{{if .IsAPIKey}} Your API is secured using an APIKey credential:header:<b>{{.KeyHeaderName}}</b>/value:<b>{{.Key}}</b>
{{else}} Your API is secured using OAuth token. You can obtain your token using grant_type=client_credentials with the following client_id=<b>{{.ClientID}}</b> and client_secret=<b>{{.ClientSecret}}</b>{{end}}

CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP_UNSUBSCRIBE_BODY=
Subscription for Catalog Item: <a href= {{.CatalogItemURL}}> {{CatalogItemName}} </a> has been unsubscribed

CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP_SUBSCRIBEFAILED_BODY=
Could not subscribe to Catalog Item: <a href= {{CatalogItemURL}}> {{CatalogItemName}}</a> {{.Message}}

CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP_UNSUBSCRIBEFAILED_BODY=
Could not unsubscribe to Catalog Item: <a href= {{.CatalogItemUrl}}> {{.CatalogItemName}}  </a>*{{.Message}} /

*/
