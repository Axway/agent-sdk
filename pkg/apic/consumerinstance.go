package apic

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/util"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/gabriel-vasile/mimetype"
)

func (c *ServiceClient) buildConsumerInstanceSpec(serviceBody *ServiceBody, doc string, categories []string) mv1a.ConsumerInstanceSpec {
	subscriptionDefinitionName := serviceBody.SubscriptionName

	autoSubscribe := false
	if c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalMode() == corecfg.AutoApproval {
		autoSubscribe = true
	}

	// Set default state to published
	if serviceBody.State == "" {
		serviceBody.State = PublishedState
	}

	enableSubscription := c.enableSubscription(serviceBody)

	owningTeam := c.cfg.GetTeamName()

	// If there is an organizationName in the serviceBody, try to find a match in the map of Central teams.
	// If found, use that as the owningTeam for the service. Otherwise, use the configured default team.
	if serviceBody.TeamName != "" {
		if _, found := c.getTeamFromCache(serviceBody.TeamName); found {
			owningTeam = serviceBody.TeamName
		} else {
			teamForMsg := "the default team"
			if owningTeam != "" {
				teamForMsg = fmt.Sprintf("team %s", owningTeam)
			}
			log.Infof("Amplify Central does not contain a team named %s for API %s. The Catalog Item will be assigned to %s.",
				serviceBody.TeamName, serviceBody.APIName, teamForMsg)
		}
	}

	return mv1a.ConsumerInstanceSpec{
		Name:               serviceBody.NameToPush,
		ApiServiceInstance: serviceBody.serviceContext.instanceName,
		Description:        serviceBody.Description,
		Visibility:         "RESTRICTED",
		Version:            serviceBody.Version,
		State:              serviceBody.State,
		Status:             serviceBody.Status,
		Tags:               mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
		Documentation:      doc,
		OwningTeam:         owningTeam,
		Subscription: mv1a.ConsumerInstanceSpecSubscription{
			Enabled:                enableSubscription,
			AutoSubscribe:          autoSubscribe,
			SubscriptionDefinition: subscriptionDefinitionName,
		},
		UnstructuredDataProperties: c.buildUnstructuredDataProperties(serviceBody),
		Categories:                 categories,
	}
}

// buildUnstructuredDataProperties - creates the unstructured data properties portion of the consumer instance
func (c *ServiceClient) buildUnstructuredDataProperties(serviceBody *ServiceBody) mv1a.ConsumerInstanceSpecUnstructuredDataProperties {
	if serviceBody.ResourceType != Unstructured {
		return mv1a.ConsumerInstanceSpecUnstructuredDataProperties{}
	}

	const defType = "Asset"
	unstructuredDataProperties := mv1a.ConsumerInstanceSpecUnstructuredDataProperties{
		Type:        defType,
		ContentType: mimetype.Detect(serviceBody.SpecDefinition).String(),
		Label:       defType,
		FileName:    serviceBody.APIName,
		Data:        base64.StdEncoding.EncodeToString(serviceBody.SpecDefinition),
	}

	if serviceBody.UnstructuredProps.AssetType != "" {
		unstructuredDataProperties.Type = serviceBody.UnstructuredProps.AssetType
		// Set the label to the same as the asset type
		unstructuredDataProperties.Label = serviceBody.UnstructuredProps.AssetType
	}

	if serviceBody.UnstructuredProps.ContentType != "" {
		unstructuredDataProperties.ContentType = serviceBody.UnstructuredProps.ContentType
	}

	if serviceBody.UnstructuredProps.Label != "" {
		unstructuredDataProperties.Label = serviceBody.UnstructuredProps.Label
		if serviceBody.UnstructuredProps.AssetType == "" {
			unstructuredDataProperties.Type = serviceBody.UnstructuredProps.Label
		}
	}

	if serviceBody.UnstructuredProps.Filename != "" {
		unstructuredDataProperties.FileName = serviceBody.UnstructuredProps.Filename
	}

	return unstructuredDataProperties
}

func (c *ServiceClient) enableSubscription(serviceBody *ServiceBody) bool {
	if len(serviceBody.authPolicies) > 0 {
		serviceBody.AuthPolicy = serviceBody.authPolicies[0] // use the first auth policy
	}
	enableSubscription := serviceBody.AuthPolicy != Passthrough
	// if there isn't a registered subscription schema, do not enable subscriptions,
	// or if the status is not PUBLISHED, do not enable subscriptions
	if serviceBody.Status != PublishedStatus || serviceBody.SubscriptionName == "" {
		enableSubscription = false
	}

	if enableSubscription {
		log.Debugf("Subscriptions will be enabled for '%s'", serviceBody.APIName)
	} else {
		log.Debugf("Subscriptions will be disabled for '%s', either because the authPolicy is pass-through or there is not a registered subscription schema", serviceBody.APIName)
	}
	return enableSubscription
}

func (c *ServiceClient) buildConsumerInstance(serviceBody *ServiceBody, name string, doc string) *mv1a.ConsumerInstance {
	owner, _ := c.getOwnerObject(serviceBody, false) // owner, _ := at this point, we don't need to validate error on getOwnerObject.  This is used for subresource status update
	ci := &mv1a.ConsumerInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: mv1a.ConsumerInstanceGVK(),
			Name:             name,
			Title:            serviceBody.NameToPush,
			Attributes:       util.MergeMapStringString(map[string]string{}, serviceBody.InstanceAttributes),
			Tags:             mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish()),
		},
		Spec:  c.buildConsumerInstanceSpec(serviceBody, doc, serviceBody.categoryNames),
		Owner: owner,
	}

	ciDetails := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	agentDetails := buildAgentDetailsSubResource(serviceBody, false, ciDetails)
	util.SetAgentDetails(ci, agentDetails)

	// add all agent details keys to the ci attributes
	for k, v := range agentDetails {
		ci.Attributes[k] = v.(string)
	}

	return ci
}

func (c *ServiceClient) updateConsumerInstance(serviceBody *ServiceBody, ci *mv1a.ConsumerInstance, doc string) {
	owner, _ := c.getOwnerObject(serviceBody, false) // owner, _ := at this point, we don't need to validate error on getOwnerObject.  This is used for subresource status update
	ci.GroupVersionKind = mv1a.ConsumerInstanceGVK()
	ci.Metadata.ResourceVersion = ""
	ci.Title = serviceBody.NameToPush
	ci.Tags = mapToTagsArray(serviceBody.Tags, c.cfg.GetTagsToPublish())
	ci.Owner = owner
	ci.Attributes = util.MergeMapStringString(map[string]string{}, serviceBody.InstanceAttributes)

	ciDetails := util.MergeMapStringInterface(serviceBody.ServiceAgentDetails, serviceBody.InstanceAgentDetails)
	agentDetails := buildAgentDetailsSubResource(serviceBody, false, ciDetails)
	util.SetAgentDetails(ci, agentDetails)

	// add all agent details keys to the ci attributes
	for k, v := range agentDetails {
		ci.Attributes[k] = v.(string)
	}

	// use existing categories only if mappings have not been configured
	categories := ci.Spec.Categories
	if corecfg.IsMappingConfigured() {
		// use only mapping categories if mapping was configured
		categories = serviceBody.categoryNames
	}
	ci.Spec = c.buildConsumerInstanceSpec(serviceBody, doc, categories)
}

// processConsumerInstance - deal with either a create or update of a consumerInstance
func (c *ServiceClient) processConsumerInstance(serviceBody *ServiceBody) error {
	// Before attempting to create the consumer instance ensure all categories exist
	for _, categoryTitle := range serviceBody.categoryTitles {
		categoryName := c.GetOrCreateCategory(categoryTitle)
		// only add categories that exist on central
		if categoryName != "" {
			serviceBody.categoryNames = append(serviceBody.categoryNames, categoryName)
		}
	}

	// Allow catalog asset to be created.  However, set to pass-through so subscriptions aren't enabled
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		log.Warnf("'%s' has an inbound policy of (%s) and is not supported. Catalog asset will be created with a pass-through inbound policy. ", serviceBody.APIName, serviceBody.AuthPolicy)
		serviceBody.AuthPolicy = Passthrough
		serviceBody.Status = UnidentifiedInboundPolicy
	}

	var doc = ""
	if serviceBody.Documentation != nil {
		var err error
		doc, err = strconv.Unquote(string(serviceBody.Documentation))
		if err != nil {
			return err
		}
	}

	consumerInstanceName := serviceBody.serviceContext.serviceName
	if serviceBody.Stage != "" {
		consumerInstanceName = sanitizeAPIName(fmt.Sprintf("%s-%s", serviceBody.serviceContext.serviceName, serviceBody.Stage))
	}

	httpMethod := http.MethodPost
	consumerInstanceURL := c.cfg.GetConsumerInstancesURL()

	var instance *mv1a.ConsumerInstance
	var err error
	if serviceBody.serviceContext.serviceAction == updateAPI {
		instance, err = c.getConsumerInstanceByName(consumerInstanceName)
		if err != nil {
			return err
		}
	}

	if instance != nil {
		httpMethod = http.MethodPut
		consumerInstanceURL += "/" + consumerInstanceName
		c.updateConsumerInstance(serviceBody, instance, doc)
	} else {
		instance = c.buildConsumerInstance(serviceBody, consumerInstanceName, doc)
	}

	buffer, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, consumerInstanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
		}
		return err
	}

	if err == nil {
		if len(instance.SubResources) > 0 {
			err = c.CreateSubResourceScoped(
				mv1a.EnvironmentResourceName,
				c.cfg.GetEnvironmentName(),
				instance.PluralName(),
				instance.Name,
				instance.Group,
				instance.APIVersion,
				instance.SubResources,
			)
			if err != nil {
				_, rollbackErr := c.rollbackAPIService(serviceBody.serviceContext.serviceName)
				if rollbackErr != nil {
					return errors.New(err.Error() + rollbackErr.Error())
				}
			}
		}
	}

	serviceBody.serviceContext.consumerInstanceName = consumerInstanceName

	return err
}

// getAPIServerConsumerInstance -
func (c *ServiceClient) getAPIServerConsumerInstance(name string, query map[string]string) (*mv1a.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	consumerInstanceURL := c.cfg.GetConsumerInstancesURL() + "/" + name

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         consumerInstanceURL,
		Headers:     headers,
		QueryParams: query,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			responseErr := readResponseErrors(response.Code, response.Body)
			return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
		}
		return nil, nil
	}
	consumerInstance := new(mv1a.ConsumerInstance)
	err = json.Unmarshal(response.Body, consumerInstance)
	return consumerInstance, err
}

// UpdateConsumerInstanceSubscriptionDefinition -
func (c *ServiceClient) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	consumerInstance, err := c.getConsumerInstancesByExternalAPIID(externalAPIID)
	if err != nil {
		return err
	}

	// Update the subscription definition
	if consumerInstance[0].Spec.Subscription.SubscriptionDefinition == subscriptionDefinitionName {
		return nil // no updates to be made
	}

	consumerInstance[0].Metadata.ResourceVersion = ""
	consumerInstance[0].Spec.Subscription.SubscriptionDefinition = subscriptionDefinitionName

	consumerInstanceURL := c.cfg.GetConsumerInstancesURL() + "/" + consumerInstance[0].Name
	buffer, err := json.Marshal(consumerInstance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(http.MethodPut, consumerInstanceURL, buffer)

	return err
}

// getConsumerInstancesByExternalAPIID gets consumer instances
func (c *ServiceClient) getConsumerInstancesByExternalAPIID(externalAPIID string) ([]*mv1a.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Tracef("Get consumer instance by external api id: %s", externalAPIID)

	params := map[string]string{
		"query": fmt.Sprintf("attributes."+definitions.AttrExternalAPIID+"==\"%s\"", externalAPIID),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetConsumerInstancesURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)

	if err != nil {
		return nil, err
	}
	if !(response.Code == http.StatusOK) {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	consumerInstances := make([]*mv1a.ConsumerInstance, 0)
	err = json.Unmarshal(response.Body, &consumerInstances)
	if err != nil {
		return nil, err
	}
	if len(consumerInstances) == 0 {
		return nil, errors.New("Unable to find consumerInstance using external api id: " + externalAPIID)
	}

	return consumerInstances, nil
}

// getConsumerInstanceByID
func (c *ServiceClient) getConsumerInstanceByID(instanceID string) (*mv1a.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Tracef("Get consumer instance by id: %s", instanceID)

	params := map[string]string{
		"query": fmt.Sprintf("metadata.id==%s", instanceID),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetConsumerInstancesURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)

	if err != nil {
		return nil, err
	}
	if !(response.Code == http.StatusOK) {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	consumerInstances := make([]*mv1a.ConsumerInstance, 0)
	err = json.Unmarshal(response.Body, &consumerInstances)
	if len(consumerInstances) == 0 {
		return nil, errors.New("Unable to find consumerInstance using instanceID " + instanceID)
	}

	return consumerInstances[0], err
}

// getConsumerInstanceByName
func (c *ServiceClient) getConsumerInstanceByName(name string) (*mv1a.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Tracef("Get consumer instance by name: %s", name)

	params := map[string]string{
		"query": fmt.Sprintf("name==%s", name),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetConsumerInstancesURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)

	if err != nil {
		return nil, err
	}
	if !(response.Code == http.StatusOK) {
		responseErr := readResponseErrors(response.Code, response.Body)
		return nil, utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	consumerInstances := make([]*mv1a.ConsumerInstance, 0)
	err = json.Unmarshal(response.Body, &consumerInstances)
	if len(consumerInstances) == 0 {
		return nil, nil
	}

	return consumerInstances[0], err
}

// deleteConsumerInstance -
func (c *ServiceClient) deleteConsumerInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetConsumerInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}
