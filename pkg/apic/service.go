package apic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	unifiedcatalog "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

type actionType int

const (
	addAPI    = iota
	updateAPI = iota
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

	// TODO - consumer instance not needed after deprecation of unified catalog
	if !c.cfg.IsMarketplaceSubsEnabled() {
		// ConsumerInstanceProcessor
		err = c.processConsumerInstance(serviceBody)
		if err != nil {
			return nil, err
		}
	}
	return apiSvc, nil
}

// DeleteServiceByName -
func (c *ServiceClient) DeleteServiceByName(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetServicesURL()+"/"+name, nil)
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
func (c *ServiceClient) GetCatalogItemIDForConsumerInstance(id string) (string, error) {
	return c.getCatalogItemIDForConsumerInstance(id)
}

// DeleteConsumerInstance -
func (c *ServiceClient) DeleteConsumerInstance(name string) error {
	return c.deleteConsumerInstance(name)
}

// DeleteAPIServiceInstance deletes an api service instance in central by name
func (c *ServiceClient) DeleteAPIServiceInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}

func (c *ServiceClient) checkReferencesToAccessRequestDefinition(ard string) int {
	count := 0
	for _, instanceKey := range c.caches.GetAPIServiceInstanceKeys() {
		serviceInstance, err := c.caches.GetAPIServiceInstanceByID(instanceKey)
		if err != nil || serviceInstance == nil {
			// skip this key as it did not return a service instance
			continue
		}
		// check the references
		for _, ref := range serviceInstance.Metadata.References {
			if ref.Kind == v1alpha1.AccessRequestDefinitionGVK().Kind {
				if ref.Name == ard {
					count++
				}
				// only 1 ard per service instance
				continue
			}
		}
	}
	return count
}

// DeleteAPIServiceInstanceWithFinalizers deletes an api service instance in central, handling finalizers
func (c *ServiceClient) DeleteAPIServiceInstanceWithFinalizers(ri *v1.ResourceInstance) error {
	url := c.cfg.GetInstancesURL() + "/" + ri.Name
	finalizers := ri.Finalizers
	ri.Finalizers = make([]v1.Finalizer, 0)

	// handle finalizers
	for _, f := range finalizers {
		if f.Name == AccessRequestDefinitionFinalizer {
			// check if we should remove the accessrequestdefinition
			if c.checkReferencesToAccessRequestDefinition(f.Description) > 1 {
				continue // do not add the finalizer back
			}
			// 1 or fewer references to the ARD, clean it up
			tempARD := v1alpha1.NewAccessRequestDefinition(f.Description, c.cfg.GetEnvironmentName())
			_, err := c.apiServiceDeployAPI(http.MethodDelete, c.createAPIServerURL(tempARD.GetSelfLink()), nil)
			if err == nil {
				continue // do not add the finalizer back
			}
		}
		ri.Finalizers = append(ri.Finalizers, f)
	}

	// get the full instance
	currentRI, err := c.executeAPIServiceAPI(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// update the finalizers in the instance from central
	currentRI.Finalizers = ri.Finalizers

	// update instance
	updatedInstance, err := json.Marshal(currentRI)
	if err != nil {
		return err
	}
	_, err = c.apiServiceDeployAPI(http.MethodPut, url, updatedInstance)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}

	return c.DeleteAPIServiceInstance(ri.Name)
}

// GetConsumerInstanceByID -
func (c *ServiceClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return c.getConsumerInstanceByID(consumerInstanceID)
}

// GetConsumerInstancesByExternalAPIID - DEPRECATED
func (c *ServiceClient) GetConsumerInstancesByExternalAPIID(externalAPIID string) ([]*v1alpha1.ConsumerInstance, error) {
	log.DeprecationWarningReplace("GetConsumerInstancesByExternalAPIID", "")
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

func buildAgentDetailsSubResource(
	serviceBody *ServiceBody, isAPIService bool, additional map[string]interface{},
) map[string]interface{} {
	details := make(map[string]interface{})

	externalAPIID := serviceBody.RestAPIID
	// check to see if is an APIService
	if !isAPIService && serviceBody.Stage != "" {
		details[defs.AttrExternalAPIStage] = serviceBody.Stage
	}
	if serviceBody.PrimaryKey != "" {
		details[defs.AttrExternalAPIPrimaryKey] = serviceBody.PrimaryKey
	}

	details[defs.AttrExternalAPIID] = externalAPIID
	details[defs.AttrExternalAPIName] = serviceBody.APIName
	details[defs.AttrCreatedBy] = serviceBody.CreatedBy

	return util.MergeMapStringInterface(details, additional)
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
	return util.ConvertToDomainNameCompliant(name)
}

// apiServiceDeployAPI -
func (c *ServiceClient) apiServiceDeployAPI(method, url string, buffer []byte) (string, error) {
	ri, err := c.executeAPIServiceAPI(method, url, buffer)
	if err != nil {
		return "", err
	}
	resourceName := ""
	if ri != nil {
		resourceName = ri.Name
	}
	return resourceName, nil
}

// executeAPIServiceAPI -
func (c *ServiceClient) executeAPIServiceAPI(method, url string, buffer []byte) (*v1.ResourceInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
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
		return nil, err
	}
	//  Check to see if rollback was processed
	if method == http.MethodDelete && response.Code == http.StatusNoContent {
		return nil, nil
	}

	if response.Code >= http.StatusBadRequest {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}
	ri := &v1.ResourceInstance{}
	json.Unmarshal(response.Body, ri)
	return ri, nil
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
