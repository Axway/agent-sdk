package notify

import (
	"encoding/json"
	"fmt"
	"strings"

	sasl "github.com/emersion/go-sasl"
	smtp "github.com/emersion/go-smtp"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
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
	apiClient       coreapi.Client
}

//NewSubscriptionNotification - creates a new subscription notification object
func NewSubscriptionNotification(catalogID, catalogName, catalogItemURL, recipient, key string, state apic.SubscriptionState, message string) *SubscriptionNotification {
	return &SubscriptionNotification{
		CatalogItemID:   catalogID,
		CatalogItemName: catalogName,
		CatalogItemURL:  catalogItemURL,
		Email:           recipient,
		Action:          state,
		Key:             key,
		Message:         message,
		apiClient:       coreapi.NewClient(corecfg.NewTLSConfig(), ""),
	}
}

// NotifySubscriber - send a notification to any configured notification type
func (s *SubscriptionNotification) NotifySubscriber(recipient string) error {
	for _, notificationType := range globalCfg.GetNotificationTypes() {
		switch notificationType {
		case config.NotifyWebhook:
			err := s.notifyViaWebhook()
			if err != nil {
				log.Errorf("Could not send notification via webook: %s", err.Error())
				return err
			}
			log.Debugf("Webhook notification sent to %s.", recipient)

		case config.NotifySMTP:
			err := s.notifyViaSMTP()
			if err != nil {
				log.Errorf("Could not send notification via smtp server: %s", err.Error())
				return err
			}
			log.Debugf("Email notification sent to %s.", recipient)
		}
	}

	return nil
}

func (s *SubscriptionNotification) notifyViaWebhook() error {
	buffer, err := json.Marshal(&s)
	if err != nil {
		log.Errorf("Error creating notification request: %s", err.Error())
		return err
	}

	fmt.Printf("%v\n", s)
	fmt.Println(string(buffer))
	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     globalCfg.GetNotificationWebhook(),
		Headers: globalCfg.GetNotificationHeaders(),
		Body:    buffer,
	}

	_, err = s.apiClient.Send(request)
	if err != nil {
		log.Errorf("Error sending notification webhook: %s", err.Error())
		return err
	}

	return nil
}

func (s *SubscriptionNotification) notifyViaSMTP() error {
	template := templateActionMap[s.Action]

	if template.Subject == "" && template.Body == "" {
		return nil
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
		log.Error(err)
		return err
	}
	return nil
}

// BuildSMTPMessage -
func (s *SubscriptionNotification) BuildSMTPMessage(template *config.EmailTemplate) *strings.Reader {
	mime := mimeMap{
		"MIME-version": "1.0",
		"Content-Type": "text/html",
		"charset":      "UTF-8",
	}

	fromAddress := fmt.Sprintf("From: %s", globalCfg.GetSMTPFromAddress())
	toAddress := fmt.Sprintf("To: %s", s.Email)
	subject := fmt.Sprintf("Subject: %s", s.UpdateTemplate(template.Subject))

	log.Debugf("Sending email %s, %s, %s", fromAddress, toAddress, subject)

	msgArray := []string{
		fromAddress,
		toAddress,
		subject,
		mime.String(),
		s.UpdateTemplate(template.Body),
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
	}

	return template
}
