package apic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
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

// SubscriptionProperties -
type SubscriptionProperties struct {
	Key    string            `json:"key"`
	Values map[string]string `json:"value"`
}

// Subscription -
type Subscription struct {
	ID                      string                   `json:"id"`
	Properties              []SubscriptionProperties `json:"properties"`
	State                   string                   `json:"state"`
	StateDescription        string                   `json:"stateDescription"`
	CatalogItemID           string                   `json:"catalogItemId"`
	OwningTeamID            string                   `json:"owningTeamId"`
	Deletable               bool                     `json:"deletable"`
	Name                    string                   `json:"name"`
	NextPossibleStates      []string                 `json:"nextPossibleStates"`
	AllowedTransitionStates []string                 `json:"allowedTransitionStates"`
	ApicID                  string                   `json:"-"`
	apicClient              *ServiceClient
}

// GetPropertyValue - Returns subscription Property value based on the key
func (s *Subscription) GetPropertyValue(key string) string {
	if len(s.Properties) > 0 {
		subscriptionProperties := s.Properties[0]
		value, ok := subscriptionProperties.Values[key]
		if ok {
			return value
		}
	}
	return ""
}

// UpdateState - Updates the state of subscription
func (s *Subscription) UpdateState(newState SubscriptionState) error {
	headers, err := s.getServiceClient().createHeader()
	if err != nil {
		return err
	}

	subStateURL := s.getServiceClient().cfg.GetCatalogItemsURL() + "/" + s.CatalogItemID + "/subscriptions/" + s.ID + "/states"
	subState := make(map[string]string)
	subState["state"] = string(newState)

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
		return err
	}
	if !(response.Code == http.StatusOK || response.Code == http.StatusCreated) {
		logResponseErrors(response.Body)
		return errors.New(strconv.Itoa(response.Code))
	}
	return nil
}

func (s *Subscription) getServiceClient() *ServiceClient {
	return s.apicClient
}

// getSubscriptions -
func (c *ServiceClient) getSubscriptions(states []string) ([]Subscription, error) {
	headers, err := c.createHeader()
	if err != nil {
		return nil, err
	}

	queryParams := make(map[string]string)

	searchQuery := ""
	for _, state := range states {
		if searchQuery != "" {
			searchQuery += ","
		}
		searchQuery += "state==" + state
	}

	queryParams["query"] = searchQuery

	request := coreapi.Request{
		Method:      coreapi.GET,
		URL:         c.cfg.GetSubscriptionURL(),
		QueryParams: queryParams,
		Headers:     headers,
		Body:        nil,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return nil, err
	}
	if response.Code != http.StatusOK {
		logResponseErrors(response.Body)
		return nil, errors.New(strconv.Itoa(response.Code))
	}
	subscriptions := make([]Subscription, 0)
	json.Unmarshal(response.Body, &subscriptions)
	return subscriptions, nil
}
