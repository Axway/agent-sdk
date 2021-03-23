package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	utilerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/tidwall/gjson"
)

// getCatalogItemIDForConsumerInstance -
func (c *ServiceClient) getCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	params := map[string]string{
		"query": fmt.Sprintf("relationships.type==API_SERVER_CONSUMER_INSTANCE_ID;relationships.value==%s", instanceID),
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetCatalogItemsURL(),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return "", utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	// the response is an array of IDs
	ids := gjson.Get(string(response.Body), "#.id")
	if !ids.Exists() {
		return "", nil
	}

	// the array should only contain 1 item,
	// since we have asked for a specific one
	catalogIDs := make([]string, 0)
	json.Unmarshal([]byte(ids.Raw), &catalogIDs)
	if len(catalogIDs) == 0 {
		return "", errors.New("Unable to find catalogID for consumerInstance " + instanceID)
	}

	return catalogIDs[0], nil
}

// GetCatalogItemName -
func (c *ServiceClient) GetCatalogItemName(ID string) (string, error) {
	headers, err := c.createHeader()
	if err != nil {
		return "", err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     c.cfg.GetCatalogItemByIDURL(ID),
		Headers: headers,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return "", err
	}
	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return "", utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	name := gjson.Get(string(response.Body), "name").String()
	return name, nil
}

func (c *ServiceClient) getSubscriptionsForCatalogItem(states []string, catalogItemID string) ([]CentralSubscription, error) {
	queryParams := make(map[string]string)

	searchQuery := ""
	for _, state := range states {
		if searchQuery != "" {
			searchQuery += ","
		}
		searchQuery += "state==" + state
	}

	queryParams["query"] = searchQuery
	subscriptions, err := c.sendSubscriptionsRequest(c.cfg.GetCatalogItemSubscriptionsURL(catalogItemID), queryParams)
	if err != nil {
		if err.Error() != strconv.Itoa(http.StatusNotFound) {
			return nil, err
		}
		return make([]CentralSubscription, 0), nil
	}
	return subscriptions, nil
}

func (c *ServiceClient) getSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (SubscriptionSchema, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:  coreapi.GET,
		URL:     fmt.Sprintf("%s/%s", c.cfg.GetCatalogItemSubscriptionDefinitionPropertiesURL(catalogItemID), propertyKey),
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

	ss := NewSubscriptionSchema("")
	err = json.Unmarshal(response.Body, &ss)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

func (c *ServiceClient) updateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema SubscriptionSchema) error {
	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	body, err := json.Marshal(subscriptionSchema)
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     fmt.Sprintf("%s/%s", c.cfg.GetCatalogItemSubscriptionDefinitionPropertiesURL(catalogItemID), propertyKey),
		Headers: headers,
		Body:    body,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return err
	}
	if response.Code != http.StatusOK {
		responseErr := readResponseErrors(response.Code, response.Body)
		return utilerrors.Wrap(ErrRequestQuery, responseErr)
	}

	return nil
}
