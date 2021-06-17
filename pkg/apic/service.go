package apic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	unifiedcatalog "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/tidwall/gjson"
)

type actionType int

const (
	none      actionType = iota
	addAPI               = iota
	updateAPI            = iota
	deleteAPI            = iota
)

const (
	apiServerPageSize = 20
)

// PublishService - processes the API to create/update apiservice, revision, instance and consumer instance
func (c *ServiceClient) PublishService(serviceBody ServiceBody) (*v1alpha1.APIService, error) {
	apiSvc, err := c.processService(&serviceBody)
	if err != nil {
		return nil, err
	}
	// Update description title after creating APIService to inlcude the stage name if it exists
	c.postAPIServiceUpdate(&serviceBody)
	err = c.processRevision(&serviceBody)
	if err != nil {
		return nil, err
	}
	err = c.processInstance(&serviceBody)
	if err != nil {
		return nil, err
	}
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		err = c.processConsumerInstance(&serviceBody)
		if err != nil {
			return nil, err
		}
	}
	return apiSvc, nil
}

// DeleteServiceByAPIID -
func (c *ServiceClient) DeleteServiceByAPIID(externalAPIID string) error {
	return c.deleteServiceByAPIID(externalAPIID)
}

// RegisterSubscriptionWebhook - Adds a new Subscription webhook. There is a single webhook
// per environment
func (c *ServiceClient) RegisterSubscriptionWebhook() error {
	// if the default is already set up, do nothing
	webhookCfg := c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
	if webhookCfg == nil || !webhookCfg.IsConfigured() {
		return nil
	}

	// create the secret
	err := c.createSecret()
	if err != nil {
		return utilerrors.Wrap(ErrCreateSecret, err.Error())

	}

	err = c.createWebhook()
	if err != nil {
		return utilerrors.Wrap(ErrCreateWebhook, err.Error())
	}

	return nil
}

// GetCatalogItemIDForConsumerInstance -
func (c *ServiceClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return c.getCatalogItemIDForConsumerInstance(instanceID)
}

// DeleteConsumerInstance -
func (c *ServiceClient) DeleteConsumerInstance(instanceName string) error {
	return c.deleteConsumerInstance(instanceName)
}

// GetConsumerInstanceByID -
func (c *ServiceClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return c.getConsumerInstanceByID((consumerInstanceID))
}

// GetConsumerInstancesByExternalAPIID -
func (c *ServiceClient) GetConsumerInstancesByExternalAPIID(externalAPIID string) ([]*v1alpha1.ConsumerInstance, error) {
	return c.getConsumerInstancesByExternalAPIID(externalAPIID)
}

// GetSubscriptionsForCatalogItem -
func (c *ServiceClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]CentralSubscription, error) {
	return c.getSubscriptionsForCatalogItem(states, instanceID)
}

// GetSubscriptionDefinitionPropertiesForCatalogItem -
func (c *ServiceClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (SubscriptionSchema, error) {
	return c.getSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey)
}

// UpdateSubscriptionDefinitionPropertiesForCatalogItem -
func (c *ServiceClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema SubscriptionSchema) error {
	return c.updateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey, subscriptionSchema)
}

// postApiServiceUpdate - called after APIService was created or updated.
// Update description and title after updating or creating APIService to include the stage name if it exists
func (c *ServiceClient) postAPIServiceUpdate(serviceBody *ServiceBody) {
	if serviceBody.Stage != "" {
		addDescription := fmt.Sprintf("StageName: %s", serviceBody.Stage)
		if len(serviceBody.Description) > 0 {
			serviceBody.Description = fmt.Sprintf("%s, %s", serviceBody.Description, addDescription)
		} else {
			serviceBody.Description = addDescription
		}
		serviceBody.NameToPush = fmt.Sprintf("%v (Stage: %v)", serviceBody.NameToPush, serviceBody.Stage)
	} else if c.cfg.GetAppendEnvironmentToTitle() {
		// Append the environment name to the title, if set
		serviceBody.NameToPush = fmt.Sprintf("%v (%v)", serviceBody.NameToPush, c.cfg.GetEnvironmentName())
	}
}

// updateAPIServiceRevisionTitle - update title after creating or updating APIService Revision according to the APIServiceRevision Pattern
func (c *ServiceClient) updateAPIServiceRevisionTitle(serviceBody *ServiceBody) string {
	title := c.cfg.GetAPIServiceRevisionPattern() // "{{APIServiceName}} - {{date:YYYY/MM/DD}} - r {{revision}}"
	revision := strconv.Itoa(serviceBody.serviceContext.revisionCount + 1)

	replaceVars := map[string]string{"APIServiceName": serviceBody.APIName, "revision": revision}
	// replace occurrences of `APIServiceName` and `revision` in title
	for k, v := range replaceVars {
		title = strings.Replace(title, fmt.Sprintf("{{%s}}", k), v, -1)
	}

	dateRegEx := regexp.MustCompile(`{{date:.*?}}`)
	if dateRegEx.MatchString(title) {
		var createdOn time.Time
		if serviceBody.serviceContext.previousRevision == nil {
			createdOn = time.Now()
		} else {
			createdOn = time.Time(serviceBody.serviceContext.previousRevision.ResourceMeta.Metadata.Audit.CreateTimestamp)
		}
		year := strconv.Itoa(createdOn.Year())
		month := createdOn.Format("01")
		day := strconv.Itoa(createdOn.Day())

		date := dateRegEx.FindString(title)
		replaceDate := map[string]string{"MM": month, "DD": day, "YYYY": year}
		for k, v := range replaceDate {
			date = strings.Replace(date, k, v, -1)
		}
		date = strings.TrimPrefix(date, "{{date:")
		date = strings.TrimSuffix(date, "}}")

		//swap the date pattern with the date variable in the title
		title = strings.Replace(title, dateRegEx.FindString(title), date, -1)
	}
	return title
}

func (c *ServiceClient) buildAPIResourceAttributes(serviceBody *ServiceBody, additionalAttr map[string]string, isAPIService bool) map[string]string {
	attributes := make(map[string]string)

	// Add attributes from resource if present
	if additionalAttr != nil {
		for key, val := range additionalAttr {
			attributes[key] = val
		}
	}

	// Add attributes from service body setup by agent
	if serviceBody.ServiceAttributes != nil {
		for key, val := range serviceBody.ServiceAttributes {
			attributes[key] = val
		}
	}

	externalAPIID := serviceBody.RestAPIID
	// check to see if its an APIService
	if !isAPIService && serviceBody.Stage != "" {
		attributes[AttrExternalAPIStage] = serviceBody.Stage
	}
	if serviceBody.PrimaryKey != "" {
		attributes[AttrExternalAPIPrimaryKey] = serviceBody.PrimaryKey
	}

	attributes[AttrExternalAPIID] = externalAPIID
	attributes[AttrExternalAPIName] = serviceBody.APIName
	attributes[AttrCreatedBy] = serviceBody.CreatedBy

	return attributes
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// Sanitize name to be path friendly and follow this regex: ^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*
func sanitizeAPIName(name string) string {
	// convert all letters to lower first
	newName := strings.ToLower(name)

	// parse name out. All valid parts must be '-', '.', a-z, or 0-9
	re := regexp.MustCompile(`[-\.a-z0-9]*`)
	matches := re.FindAllString(newName, -1)

	// join all of the parts, separated with '-'. This in effect is substituting all illegal chars with a '-'
	newName = strings.Join(matches, "-")

	// The regex rule says that the name must not begin or end with a '-' or '.', so trim them off
	newName = strings.TrimLeft(strings.TrimRight(newName, "-."), "-.")

	// The regex rule also says that the name must not have a sequence of ".-", "-.", or "..", so replace them
	r1 := strings.ReplaceAll(newName, "-.", "--")
	r2 := strings.ReplaceAll(r1, ".-", "--")
	r3 := strings.ReplaceAll(r2, "..", "--")

	return r3
}

// apiServiceDeployAPI -
func (c *ServiceClient) apiServiceDeployAPI(method, url string, buffer []byte) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	request := coreapi.Request{
		Method:      method,
		URL:         url,
		QueryParams: nil,
		Headers:     headers,
		Body:        buffer,
	}
	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	//  Check to see if rollback was processed
	if method == http.MethodDelete && response.Code == http.StatusNoContent {
		return "", nil
	}

	if response.Code >= http.StatusBadRequest {
		responseErr := readResponseErrors(response.Code, response.Body)
		return "", utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	itemID := ""
	metadata := gjson.Get(string(response.Body), "metadata").String()
	if metadata != "" {
		itemID = gjson.Get(string(metadata), "id").String()
	}

	return itemID, nil
}

// create the on-and-only secret for the environment
func (c *ServiceClient) createSecret() error {
	s := c.DefaultSubscriptionApprovalWebhook.GetSecret()
	spec := v1alpha1.SecretSpec{
		Data: map[string]string{DefaultSubscriptionWebhookAuthKey: base64.StdEncoding.EncodeToString([]byte(s))},
	}

	secret := v1alpha1.Secret{
		ResourceMeta: v1.ResourceMeta{Name: DefaultSubscriptionWebhookName},
		Spec:         spec,
	}

	buffer, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.cfg.GetAPIServerSecretsURL(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return err
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		responseErr := readResponseErrors(response.Code, response.Body)
		return utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
	if response.Code == http.StatusConflict {
		request = coreapi.Request{
			Method:  coreapi.PUT,
			URL:     c.cfg.GetAPIServerSecretsURL() + "/" + DefaultSubscriptionWebhookName,
			Headers: headers,
			Body:    buffer,
		}

		response, err := c.apiClient.Send(request)
		if err != nil {
			return err
		}
		if !(response.Code == http.StatusOK) {
			responseErr := readResponseErrors(response.Code, response.Body)
			return utilerrors.Wrap(ErrRequestQuery, responseErr)
		}
	}

	return nil
}

// create the on-and-only subscription approval webhook for the environment
func (c *ServiceClient) createWebhook() error {
	webhookCfg := c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalWebhookConfig()
	specSecret := v1alpha1.WebhookSpecAuthSecret{
		Name: DefaultSubscriptionWebhookName,
		Key:  DefaultSubscriptionWebhookAuthKey,
	}
	authSpec := v1alpha1.WebhookSpecAuth{
		Secret: specSecret,
	}
	webSpec := v1alpha1.WebhookSpec{
		Auth:    authSpec,
		Enabled: true,
		Url:     webhookCfg.GetURL(),
		Headers: webhookCfg.GetWebhookHeaders(),
	}

	webhook := v1alpha1.Webhook{
		ResourceMeta: v1.ResourceMeta{Name: DefaultSubscriptionWebhookName},
		Spec:         webSpec,
	}

	buffer, err := json.Marshal(webhook)
	if err != nil {
		return err
	}

	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.cfg.GetAPIServerWebhooksURL(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return err
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		responseErr := readResponseErrors(response.Code, response.Body)
		return utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
	if response.Code == http.StatusConflict {
		request = coreapi.Request{
			Method:  coreapi.PUT,
			URL:     c.cfg.GetAPIServerWebhooksURL() + "/" + DefaultSubscriptionWebhookName,
			Headers: headers,
			Body:    buffer,
		}

		response, err := c.apiClient.Send(request)
		if err != nil {
			return err
		}
		if !(response.Code == http.StatusOK) {
			responseErr := readResponseErrors(response.Code, response.Body)
			return utilerrors.Wrap(ErrRequestQuery, responseErr)
		}
	}

	return nil
}

// getCatalogItemAPIServerInfoProperty -
func (c *ServiceClient) getCatalogItemAPIServerInfoProperty(catalogID, subscriptionID string) (*APIServerInfo, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	subscriptionRelationshipsURL := c.cfg.GetCatalogItemSubscriptionRelationshipURL(catalogID, subscriptionID)

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     subscriptionRelationshipsURL,
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	relationships := make([]unifiedcatalog.EntityRelationship, 0)
	json.Unmarshal(response.Body, &relationships)
	apiserverInfo := new(APIServerInfo)
	for _, relationship := range relationships {
		if relationship.Key == "apiServerInfo" {
			switch relationship.Type {
			case "API_SERVER_CONSUMER_INSTANCE_ID":
				apiserverInfo.ConsumerInstance.ID = relationship.Value
			case "API_SERVER_CONSUMER_INSTANCE_NAME":
				apiserverInfo.ConsumerInstance.Name = relationship.Value
			case "API_SERVER_ENVIRONMENT_ID":
				apiserverInfo.Environment.ID = relationship.Value
			case "API_SERVER_ENVIRONMENT_NAME":
				apiserverInfo.Environment.Name = relationship.Value
			}
		}
	}

	return apiserverInfo, nil
}
