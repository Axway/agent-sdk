package apic

import (
	"encoding/json"
	"net/http"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
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

// SubscriptionProperties -
type SubscriptionProperties struct {
	Key    string            `json:"key"`
	Values map[string]string `json:"value"`
}

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
	ID                      string                      `json:"id"`
	Properties              []SubscriptionProperties    `json:"properties"`
	State                   string                      `json:"state"`
	StateDescription        string                      `json:"stateDescription"`
	CatalogItemID           string                      `json:"catalogItemId"`
	OwningTeamID            string                      `json:"owningTeamId"`
	Deletable               bool                        `json:"deletable"`
	Name                    string                      `json:"name"`
	NextPossibleStates      []string                    `json:"nextPossibleStates"`
	AllowedTransitionStates []string                    `json:"allowedTransitionStates"`
	Metadata                centralSubscriptionMetadata `json:"metadata"`
	ApicID                  string                      `json:"-"`
	RemoteAPIID             string                      `json:"-"`
	apicClient              *ServiceClient
}

// CentralSubscriptionMetadata -
type centralSubscriptionMetadata struct {
	CreateTimestamp string `json:"createTimestamp"`
	CreateUserID    string `json:"createUserId"`
	ModifyTimestamp string `json:"modifyTimestamp"`
	ModifyUserID    string `json:"modifyUserId"`
}

// GetCreatedUserID - Returns ID of the user that created the subscription
func (s *CentralSubscription) GetCreatedUserID() string {
	return s.Metadata.CreateUserID
}

// GetID - Returns ID of the subscription
func (s *CentralSubscription) GetID() string {
	return s.ID
}

// GetName - Returns Name of the subscription
func (s *CentralSubscription) GetName() string {
	return s.Name
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
	return s.CatalogItemID
}

// GetState - Returns subscription state
func (s *CentralSubscription) GetState() SubscriptionState {
	return SubscriptionState(s.State)
}

// GetPropertyValue - Returns subscription Property value based on the key
func (s *CentralSubscription) GetPropertyValue(key string) string {
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
func (s *CentralSubscription) UpdateState(newState SubscriptionState) error {
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
	subscriptions := make([]CentralSubscription, 0)
	json.Unmarshal(response.Body, &subscriptions)
	return subscriptions, nil
}
