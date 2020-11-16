package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	log "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
)

func (c *ServiceClient) buildConsumerInstanceSpec(serviceBody *ServiceBody, doc string) v1alpha1.ConsumerInstanceSpec {
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

	return v1alpha1.ConsumerInstanceSpec{
		Name:               serviceBody.NameToPush,
		ApiServiceInstance: serviceBody.serviceContext.currentInstance,
		Description:        serviceBody.Description,
		Visibility:         "RESTRICTED",
		Version:            serviceBody.Version,
		State:              string(serviceBody.State),
		Status:             serviceBody.Status,
		Tags:               c.mapToTagsArray(serviceBody.Tags),
		Documentation:      doc,
		OwningTeam:         c.cfg.GetTeamName(),
		Subscription: v1alpha1.ConsumerInstanceSpecSubscription{
			Enabled:                enableSubscription,
			AutoSubscribe:          autoSubscribe,
			SubscriptionDefinition: subscriptionDefinitionName,
		},
	}
}

func (c *ServiceClient) enableSubscription(serviceBody *ServiceBody) bool {
	enableSubscription := serviceBody.AuthPolicy != Passthrough
	// if there isn't a registered subscription schema, do not enable subscriptions,
	// or if the status is not PUBLISHED, do not enable subscriptions
	if enableSubscription && c.RegisteredSubscriptionSchema == nil || serviceBody.Status != PublishedStatus {
		enableSubscription = false
	}

	if enableSubscription {
		log.Debug("Subscriptions will be enabled for consumer instances")
	} else {
		log.Debug("Subscriptions will be disabled for consumer instances, either because the authPolicy is pass-through or there is not a registered subscription schema")
	}
	return enableSubscription
}

func (c *ServiceClient) buildConsumerInstance(serviceBody *ServiceBody, consumerInstanceName, doc string) *v1alpha1.ConsumerInstance {
	return &v1alpha1.ConsumerInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.ConsumerInstanceGVK(),
			Name:             consumerInstanceName,
			Title:            serviceBody.NameToPush,
			Attributes:       c.buildAPIResourceAttributes(serviceBody, nil, false),
			Tags:             c.mapToTagsArray(serviceBody.Tags),
		},
		Spec: c.buildConsumerInstanceSpec(serviceBody, doc),
	}
}

func (c *ServiceClient) updateConsumerInstanceResource(revision *v1alpha1.ConsumerInstance, serviceBody *ServiceBody, doc string) {
	revision.Title = serviceBody.NameToPush
	revision.ResourceMeta.Attributes = c.buildAPIResourceAttributes(serviceBody, revision.ResourceMeta.Attributes, true)
	revision.ResourceMeta.Tags = c.mapToTagsArray(serviceBody.Tags)
	revision.Spec = c.buildConsumerInstanceSpec(serviceBody, doc)
}

//processConsumerInstance - deal with either a create or update of a consumerInstance
func (c *ServiceClient) processConsumerInstance(serviceBody *ServiceBody) error {
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

	var consumerInstance *v1alpha1.ConsumerInstance
	var err error
	if serviceBody.serviceContext.serviceAction == updateAPI {
		consumerInstance, err = c.getConsumerInstanceByName(consumerInstanceName)
		if err != nil {
			return err
		}
	}

	if consumerInstance != nil {
		httpMethod = http.MethodPut
		consumerInstanceURL += "/" + consumerInstanceName
		c.updateConsumerInstanceResource(consumerInstance, serviceBody, doc)
	} else {
		consumerInstance = c.buildConsumerInstance(serviceBody, consumerInstanceName, doc)
	}

	buffer, err := json.Marshal(consumerInstance)
	if err != nil {
		return err
	}

	_, err = c.apiServiceDeployAPI(httpMethod, consumerInstanceURL, buffer)
	if err != nil {
		if serviceBody.serviceContext.serviceAction == addAPI {
			_, rollbackErr := c.rollbackAPIService(*serviceBody, serviceBody.serviceContext.serviceName)
			if rollbackErr != nil {
				err = rollbackErr
			}
			return err
		}
	} else {
		serviceBody.serviceContext.consumerInstance = consumerInstanceName
	}
	return err
}

// getAPIServerConsumerInstance -
func (c *ServiceClient) getAPIServerConsumerInstance(consumerInstanceName string, queryParams map[string]string) (*v1alpha1.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	consumerInstanceURL := c.cfg.GetConsumerInstancesURL() + "/" + consumerInstanceName

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         consumerInstanceURL,
		Headers:     headers,
		QueryParams: queryParams,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		if response.Code != http.StatusNotFound {
			logResponseErrors(response.Body)
			return nil, errors.New(strconv.Itoa(response.Code))
		}
		return nil, nil
	}
	consumerInstance := new(v1alpha1.ConsumerInstance)
	json.Unmarshal(response.Body, consumerInstance)
	return consumerInstance, nil
}

// getConsumerInstanceByID
func (c *ServiceClient) getConsumerInstanceByID(instanceID string) (*v1alpha1.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Debugf("Get consumer instance by id: %s", instanceID)

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
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	consumerInstances := make([]*v1alpha1.ConsumerInstance, 0)
	json.Unmarshal(response.Body, &consumerInstances)
	if len(consumerInstances) == 0 {
		return nil, errors.New("Unable to find consumerInstance using instanceID " + instanceID)
	}

	return consumerInstances[0], nil
}

// getConsumerInstanceByName
func (c *ServiceClient) getConsumerInstanceByName(consumerInstanceName string) (*v1alpha1.ConsumerInstance, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	log.Debugf("Get consumer instance by name: %s", consumerInstanceName)

	params := map[string]string{
		"query": fmt.Sprintf("name==%s", consumerInstanceName),
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
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	consumerInstances := make([]*v1alpha1.ConsumerInstance, 0)
	json.Unmarshal(response.Body, &consumerInstances)
	if len(consumerInstances) == 0 {
		return nil, nil
	}

	return consumerInstances[0], nil
}

// deleteConsumerInstance -
func (c *ServiceClient) deleteConsumerInstance(name string) error {
	_, err := c.apiServiceDeployAPI(http.MethodDelete, c.cfg.GetConsumerInstancesURL()+"/"+name, nil)
	if err != nil && err.Error() != strconv.Itoa(http.StatusNotFound) {
		return err
	}
	return nil
}
