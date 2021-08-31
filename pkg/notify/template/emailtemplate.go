package template

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

//TODO
/*
	1. Search for comment "DEPRECATED to be removed on major release"
	2. Remove deprecated code left from APIGOV-19751
*/

const (
	authTemplateTag = "{{.AuthTemplate}}"
)

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
	"${authtemplate}":    authTemplateTag,
	"${message}":         "{{.Message}}",
}

// EmailNotificationTemplate - (go) template for email notification
type EmailNotificationTemplate struct {
	CatalogItemID   string `json:"catalogItemId"`
	CatalogItemURL  string `json:"catalogItemUrl"`
	CatalogItemName string `json:"catalogItemName"`
	Email           string `json:"email,omitempty"`
	Message         string `json:"message,omitempty"`
	Key             string `json:"key,omitempty"`
	KeyHeaderName   string `json:"keyHeaderName,omitempty"`
	ClientID        string `json:"clientID,omitempty"`
	ClientSecret    string `json:"clientSecret,omitempty"`
	AuthTemplate    string `json:"authtemplate,omitempty"`
	IsAPIKey        bool   `json:"isAPIKey,omitempty"`
}

// ValidateSubscriptionConfigOnStartup - validate body and auth template tags during startup (config)
func ValidateSubscriptionConfigOnStartup(body, authTemplate string, emailNotificationTemplate EmailNotificationTemplate) (string, error) {
	return validateEmailTags(body, authTemplate, emailNotificationTemplate, false)
}

// ValidateSubscriptionConfigOnNotification - validate body and auth template tags  during notification
func ValidateSubscriptionConfigOnNotification(body, authTemplate string, emailNotificationTemplate EmailNotificationTemplate) (string, error) {
	return validateEmailTags(body, authTemplate, emailNotificationTemplate, true)
}

// validateEmailTags - validate body and auth template tags
// duringProcessing bool indicates whether this func is called during startup (config) or duringProcessing (subscription notification)
func validateEmailTags(body, authTemplate string, emailNotificationTemplate EmailNotificationTemplate, duringProcessing bool) (string, error) {
	//DEPRECATED to be removed on major release - this check for '${"' will no longer be needed after "${tag} is invalid"

	// Verify if customer is still using "${tag}" teamplate.  Warn them that it is going to be deprecated
	// Transform the old "${tag}" to the go template {{.Tag}}
	if strings.Contains(body, "${") {
		log.DeprecationWarningDoc("Using '${tag}' as part of CENTRAL_SUBSCRIPTIONS_NOTIFICATIONS_SMTP", fmt.Sprintf("'{{.Tag}}. Please consider updating template for : %s", body))
		// update body using the old style Body concat with AuthTemplate
		body = updateTemplate(fmt.Sprintf("%s. </br>%s", body, authTemplate))
	} // else customer is using the {{.Tag}} and therefore the body should already contain the authTemplate in the case of SUBSCRIBE

	// Bypass this check if the validation is happening during startup (config).  We won't know the auth type during starup
	if duringProcessing && strings.Contains(body, authTemplateTag) {
		body = strings.Replace(body, authTemplateTag, authTemplate, -1)
	}

	return setEmailBodyTemplate(body, emailNotificationTemplate)

}

//updateTemplate - update ${tag} to {{.Tag}}.  ${tag} to be deprecated
func updateTemplate(template string) string {

	for k, v := range subNotifTemplateMap {
		template = strings.ReplaceAll(template, k, v)
	}

	return template
}

// setEmailBodyTemplate - set email body using Go template
func setEmailBodyTemplate(body string, emailNotificationTemplate EmailNotificationTemplate) (string, error) {

	c, err := template.New("catalogTemplate").Parse(body)
	if err != nil {
		return "", errors.New(err.Error())
	}

	var catalogItem bytes.Buffer

	err = c.Execute(&catalogItem, emailNotificationTemplate)

	// Errors are returned in the following format
	// "template: catalogTemplate:1:63: executing "catalogTemplate" at <.CatffalogItemURL>: can't evaluate field CatffalogItemURL in type notify.SubscriptionNotification"
	// "template: catalogTemplate:1:207: executing "catalogTemplate" at <.KeyHeafederName>: can't evaluate field KeyHeafederName in type notify.SubscriptionNotification"
	if err != nil {
		// attempt to grab error returned from .Execute() beginning, "can't evaluate"
		errString := err.Error()
		indexCantEvaluate := strings.Index(errString, "can't evaluate")
		indexInType := strings.Index(errString, "in type")
		if indexCantEvaluate > 0 {
			errString = string(err.Error()[indexCantEvaluate:indexInType]) + "for SMTP template : " + body
		}

		return "", errors.New(errString)
	}

	return catalogItem.String(), nil
}
