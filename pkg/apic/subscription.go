package apic

import (
	"encoding/json"
	"fmt"
	"net/http"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

// SubscriptionState - Type definition for subscription state
type SubscriptionState string

// SubscriptionState
const (
	SubscriptionApproved             = SubscriptionState("APPROVED")
	SubscriptionRequested            = SubscriptionState("REQUESTED")
	SubscriptionRejected             = SubscriptionState("REJECTED")
	SubscriptionActive               = SubscriptionState("ACTIVE")
	SubscriptionUnsubscribed         = SubscriptionState("UNSUBSCRIBED")
	SubscriptionUnsubscribeInitiated = SubscriptionState("UNSUBSCRIBE_INITIATED")
	SubscriptionFailedToSubscribe    = SubscriptionState("FAILED_TO_SUBSCRIBE")
	SubscriptionFailedToUnsubscribe  = SubscriptionState("FAILED_TO_UNSUBSCRIBE")
)

const (
	appNameKey              = "appName"
	subscriptionAppNameType = "string"
	profileKey              = "profile"
)

// Subscription -
type Subscription interface {
	GetID() string
	GetName() string
	GetApicID() string
	GetRemoteAPIID() string
	GetRemoteAPIStage() string
	GetCatalogItemID() string
	GetCreatedUserID() string
	GetState() SubscriptionState
	GetPropertyValue(propertyKey string) string
	UpdateState(newState SubscriptionState, description string) error
	UpdateStateWithProperties(newState SubscriptionState, description string, properties map[string]interface{}) error
	UpdateProperties(appName string) error
}

// CentralSubscription -
type CentralSubscription struct {
	CatalogItemSubscription *uc.CatalogItemSubscription `json:"catalogItemSubscription"`
	ApicID                  string                      `json:"-"`
	RemoteAPIID             string                      `json:"-"`
	RemoteAPIStage          string                      `json:"-"`
	apicClient              *ServiceClient
}

// GetCreatedUserID - Returns ID of the user that created the subscription
func (s *CentralSubscription) GetCreatedUserID() string {
	return s.CatalogItemSubscription.Metadata.CreateUserId
}

// GetID - Returns ID of the subscription
func (s *CentralSubscription) GetID() string {
	return s.CatalogItemSubscription.Id
}

// GetName - Returns Name of the subscription
func (s *CentralSubscription) GetName() string {
	return s.CatalogItemSubscription.Name
}

// GetApicID - Returns ID of the Catalog Item or API Service instance
func (s *CentralSubscription) GetApicID() string {
	return s.ApicID
}

// GetRemoteAPIID - Returns ID of the API on remote gateway
func (s *CentralSubscription) GetRemoteAPIID() string {
	return s.RemoteAPIID
}

// GetRemoteAPIStage - Returns the stage name of the API on remote gateway
func (s *CentralSubscription) GetRemoteAPIStage() string {
	return s.RemoteAPIStage
}

// GetCatalogItemID - Returns ID of the Catalog Item
func (s *CentralSubscription) GetCatalogItemID() string {
	return s.CatalogItemSubscription.CatalogItemId
}

// GetState - Returns subscription state
func (s *CentralSubscription) GetState() SubscriptionState {
	return SubscriptionState(s.CatalogItemSubscription.State)
}

// GetPropertyValue - Returns subscription Property value based on the key
func (s *CentralSubscription) GetPropertyValue(propertyKey string) string {
	if len(s.CatalogItemSubscription.Properties) > 0 {
		subscriptionProperty := s.CatalogItemSubscription.Properties[0]
		value, ok := subscriptionProperty.Value[propertyKey]
		if ok {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

func (s *CentralSubscription) updateProperties(properties map[string]interface{}) error {
	if len(properties) == 0 {
		return nil
	}

	// keep existing properties
	var profile map[string]interface{}
	for _, p := range s.CatalogItemSubscription.Properties {
		if p.Key == profileKey {
			profile = p.Value
		}
	}

	allProps := map[string]interface{}{}
	// keep existing properties
	for k, v := range profile {
		allProps[k] = v
	}

	// override with new values
	for k, v := range properties {
		allProps[k] = v
	}

	return s.updatePropertyValue(profileKey, allProps)
}

// UpdateStateWithProperties - Updates the state of subscription
func (s *CentralSubscription) UpdateStateWithProperties(newState SubscriptionState, description string, properties map[string]interface{}) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	subStateURL := s.getServiceClient().cfg.GetCatalogItemSubscriptionStatesURL(s.GetCatalogItemID(), s.GetID())
	subState := uc.CatalogItemSubscriptionState{
		Description: description,
		State:       string(newState),
	}

	statePostBody, err := json.Marshal(subState)
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:      coreapi.POST,
		URL:         subStateURL,
		QueryParams: nil,
		Headers:     headers,
		Body:        statePostBody,
	}

	if err = s.updateProperties(properties); err != nil {
		return err
	}

	response, err := s.getServiceClient().apiClient.Send(request)
	if err != nil {
		return agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
	}
	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		logResponseErrors(response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}

// UpdateState - Updates the state of subscription
func (s *CentralSubscription) UpdateState(newState SubscriptionState, description string) error {
	return s.UpdateStateWithProperties(newState, description, map[string]interface{}{})
}

func (s *CentralSubscription) getServiceClient() *ServiceClient {
	return s.apicClient
}

// getSubscriptions -
func (c *ServiceClient) getSubscriptions(states []string) ([]CentralSubscription, error) {
	queryParams := make(map[string]string)

	searchQuery := ""
	for _, state := range states {
		if searchQuery != "" {
			searchQuery += ","
		}
		searchQuery += "state==" + state
	}

	queryParams["query"] = searchQuery
	return c.sendSubscriptionsRequest(c.cfg.GetSubscriptionURL(), queryParams)
}

func (c *ServiceClient) sendSubscriptionsRequest(url string, queryParams map[string]string) ([]CentralSubscription, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         url,
		QueryParams: queryParams,
		Headers:     headers,
		Body:        nil,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, agenterrors.Wrap(ErrSubscriptionQuery, err.Error())
	}
	if response.Code != http.StatusOK && response.Code != http.StatusNotFound {
		logResponseErrors(response.Body)
		return nil, ErrSubscriptionResp.FormatError(response.Code)
	}

	subscriptions := make([]uc.CatalogItemSubscription, 0)
	json.Unmarshal(response.Body, &subscriptions)

	// build the CentralSubscriptions from the UC ones
	centralSubscriptions := make([]CentralSubscription, 0)
	for i := range subscriptions {
		sub := CentralSubscription{
			CatalogItemSubscription: &subscriptions[i],
			apicClient:              c,
		}
		centralSubscriptions = append(centralSubscriptions, sub)
	}
	return centralSubscriptions, nil
}

// UpdateProperties -
func (s *CentralSubscription) UpdateProperties(appName string) error {
	catalogItemID := s.GetCatalogItemID()

	// First need to get the subscriptionDefProperties for the catalog item
	ss, err := s.getServiceClient().GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey)
	if ss == nil || err != nil {
		return agenterrors.Wrap(ErrGetSubscriptionDefProperties, err.Error())
	}

	// update the appName in the enum
	prop := ss.GetProperty(appNameKey)
	apps := append(prop.Enum, appName)
	ss.AddProperty(appNameKey, subscriptionAppNameType, "", "", true, apps)

	// update the the subscriptionDefProperties for the catalog item. This MUST be done before updating the subscription
	err = s.getServiceClient().UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey, ss)
	if err != nil {
		return agenterrors.Wrap(ErrUpdateSubscriptionDefProperties, err.Error())
	}

	// Now we can update the appname in the subscription itself
	err = s.updatePropertyValue(profileKey, map[string]interface{}{appNameKey: appName})
	if err != nil {
		return agenterrors.Wrap(ErrUpdateSubscriptionDefProperties, err.Error())
	}

	return nil
}

// UpdatePropertyValue - Updates the property value of the subscription
func (s *CentralSubscription) updatePropertyValue(propertyKey string, value map[string]interface{}) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", s.getServiceClient().cfg.GetCatalogItemSubscriptionPropertiesURL(s.GetCatalogItemID(), s.GetID()), propertyKey)
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     url,
		Headers: headers,
		Body:    body,
	}

	response, err := s.getServiceClient().apiClient.Send(request)
	if err != nil {
		return err
	}

	if !(response.Code == http.StatusOK) {
		logResponseErrors(response.Body)
		return ErrSubscriptionResp.FormatError(response.Code)
	}
	return nil
}
