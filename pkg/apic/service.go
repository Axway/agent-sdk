package apic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

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
func (c *ServiceClient) PublishService(serviceBody *ServiceBody) (*v1alpha1.APIService, error) {
	// if the team is set in the config, use that team name and id for all services
	if c.cfg.GetTeamName() != "" {
		if teamID, found := c.getTeamFromCache(c.cfg.GetTeamName()); found {
			serviceBody.TeamName = c.cfg.GetTeamName()
			serviceBody.teamID = teamID
		}
	}
	apiSvc, err := c.processService(serviceBody)
	if err != nil {
		return nil, err
	}
	// Update description title after creating APIService to include the stage name if it exists
	c.postAPIServiceUpdate(serviceBody)
	// RevisionProcessor
	err = c.processRevision(serviceBody)
	if err != nil {
		return nil, err
	}

	// InstanceProcessor
	err = c.processInstance(serviceBody)
	if err != nil {
		return nil, err
	}

	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		// ConsumerInstanceProcessor
		err = c.processConsumerInstance(serviceBody)
		if err != nil {
			return nil, err
		}
	}
	return apiSvc, nil
}

// DeleteServiceByName -
func (c *ServiceClient) DeleteServiceByName(apiName string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetServicesURL()+"/"+apiName, nil)
	if err != nil {
		return err
	}
	return nil
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

// DeleteAPIServiceInstance deletes an api service instance in central by name
func (c *ServiceClient) DeleteAPIServiceInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
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
		addDescription := fmt.Sprintf("%s: %s", serviceBody.StageDescriptor, serviceBody.Stage)
		if len(serviceBody.Description) > 0 {
			serviceBody.Description = fmt.Sprintf("%s, %s", serviceBody.Description, addDescription)
		} else {
			serviceBody.Description = addDescription
		}
		serviceBody.NameToPush = fmt.Sprintf("%v (%s: %v)", serviceBody.NameToPush, serviceBody.StageDescriptor, serviceBody.Stage)
	} else if c.cfg.GetAppendEnvironmentToTitle() {
		// Append the environment name to the title, if set
		serviceBody.NameToPush = fmt.Sprintf("%v (%v)", serviceBody.NameToPush, c.cfg.GetEnvironmentName())
	}
}

func (c *ServiceClient) buildAPIResourceAttributes(serviceBody *ServiceBody, additionalAttr map[string]string, isAPIService bool) map[string]string {
	attributes := make(map[string]string)

	// Add attributes from resource if present
	for key, val := range additionalAttr {
		attributes[key] = val
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
		attributes[definitions.AttrExternalAPIStage] = serviceBody.Stage
	}
	if serviceBody.PrimaryKey != "" {
		attributes[definitions.AttrExternalAPIPrimaryKey] = serviceBody.PrimaryKey
	}

	attributes[definitions.AttrExternalAPIID] = externalAPIID
	attributes[definitions.AttrExternalAPIName] = serviceBody.APIName
	attributes[definitions.AttrCreatedBy] = serviceBody.CreatedBy

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
