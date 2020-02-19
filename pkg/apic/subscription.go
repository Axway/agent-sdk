package apic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
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
