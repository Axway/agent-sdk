package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	unifiedcatalog "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/unifiedcatalog/models"
	log "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/util/log"
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
	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
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

// getConsumerInstanceForCatalogItem -
func (c *ServiceClient) getConsumerInstanceForCatalogItem(itemID string) (*APIServer, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"query": "type==API_SERVER_CONSUMER_INSTANCE_NAME",
	}
	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetCatalogItemRelationshipsURL(itemID),
		Headers:     headers,
		QueryParams: params,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}

	relationships := make([]unifiedcatalog.EntityRelationship, 0)
	err = json.Unmarshal(response.Body, &relationships)
	if err != nil {
		return nil, err
	}
	if len(relationships) == 0 {
		return nil, errors.New("No relationships found")
	}

	return c.getAPIServerConsumerInstance(relationships[0].Value, nil)
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
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
	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return "", errors.New(strconv.Itoa(response.Code))
	}

	name := gjson.Get(string(response.Body), "name").String()
	return name, nil
}

// RemoveActiveSubscriptionsForCatalogItem - set all active subscriptions for the catalogItem to unsubscribed
func (c *ServiceClient) RemoveActiveSubscriptionsForCatalogItem(catalogItemID string) error {
	if c.cfg.IsPublishToEnvironmentOnlyMode() {
		return nil
	}

	// move any subscriptions directly through and delete the catalog item. By blacklisting the item,
	// the polling for subscriptions for this item will be circumvented
	_, err := c.initiateUnsubscribeCatalogItem(catalogItemID)
	if err != nil {
		log.Errorf("Error initiating unsubscribe of catalogItem with ID '%v': %v", catalogItemID, err.Error())
		return err
	}
	_, err = c.unsubscribeCatalogItem(catalogItemID)
	if err != nil {
		log.Errorf("Error unsubscribing of catalogItem with ID '%v': %v", catalogItemID, err.Error())
		return err
	}
	return nil
}

// initiateUnsubscribeCatalogItem - move the catalog item to unsubscribed initiated state
func (c *ServiceClient) initiateUnsubscribeCatalogItem(catalogItemID string) (int, error) {
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		subscriptions, err := c.getSubscriptionsForCatalogItem([]string{string(SubscriptionActive)}, catalogItemID)
		if err != nil {
			return 0, err
		}

		for _, subscription := range subscriptions {
			// just initiate the unsubscribe, and let the poller handle finishing it all up
			subscription.apicClient = c
			log.Debugf("Updating subscription '%s' for catalog item ID '%s' to state: %s", subscription.Name, catalogItemID, string(SubscriptionUnsubscribeInitiated))
			err = subscription.UpdateState(SubscriptionUnsubscribeInitiated)
			if err != nil {
				return len(subscriptions), err
			}
		}
		return len(subscriptions), nil
	}

	return 0, nil
}

// unsubscribeCatalogItem - move the catalog item to unsubscribed state
func (c *ServiceClient) unsubscribeCatalogItem(catalogItemID string) (int, error) {
	if c.cfg.IsPublishToEnvironmentAndCatalogMode() {
		subscriptions, err := c.getSubscriptionsForCatalogItem([]string{string(SubscriptionUnsubscribeInitiated)}, catalogItemID)
		if err != nil {
			return 0, err
		}

		for _, subscription := range subscriptions {
			subscription.apicClient = c
			log.Debugf("Updating subscription '%s' for catalog item ID '%s' to state: %s", subscription.Name, catalogItemID, string(SubscriptionUnsubscribed))
			err = subscription.UpdateState(SubscriptionUnsubscribed)
			if err != nil {
				return len(subscriptions), err
			}
		}
		return len(subscriptions), nil
	}

	return 0, nil
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
