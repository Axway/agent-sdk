package apic

import (
	"encoding/json"
	"fmt"
	"net/http"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	uc "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/unifiedcatalog/models"
	agenterrors "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/errors"
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

// Subscription -
type Subscription interface {
	GetID() string
	GetName() string
	GetApicID() string
	GetRemoteAPIID() string
	GetCatalogItemID() string
	GetCreatedUserID() string
	GetState() SubscriptionState
	GetPropertyValue(key string) string
	UpdateState(newState SubscriptionState) error
}

// CentralSubscription -
type CentralSubscription struct {
	CatalogItemSubscription *uc.CatalogItemSubscription `json:"catalogItemSubscription"`
	ApicID                  string                      `json:"-"`
	RemoteAPIID             string                      `json:"-"`
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

// GetRemoteAPIID - Returns ID of the API on remote gatewat
func (s *CentralSubscription) GetRemoteAPIID() string {
	return s.RemoteAPIID
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
func (s *CentralSubscription) GetPropertyValue(key string) string {
	if len(s.CatalogItemSubscription.Properties) > 0 {
		subscriptionProperty := s.CatalogItemSubscription.Properties[0]
		value, ok := subscriptionProperty.Value[key]
		if ok {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

// UpdateState - Updates the state of subscription
func (s *CentralSubscription) UpdateState(newState SubscriptionState) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	subStateURL := s.getServiceClient().cfg.GetCatalogItemSubscriptionStatesURL(s.GetCatalogItemID(), s.GetID())
	subState := uc.CatalogItemSubscriptionState{
		State: string(newState),
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
