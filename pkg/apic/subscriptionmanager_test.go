package apic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/cache"
	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

type mockTokenGetter struct {
	token string
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return m.token, nil
}

func TestProcessorRegistration(t *testing.T) {
	cfg := &corecfg.CentralConfiguration{
		TeamID: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)

	approvedProcessor := func(subscription Subscription) {}
	unsubscribeProcessor := func(subscription Subscription) {}

	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribeProcessor)

	processorMap := serviceClient.subscriptionMgr.getProcessorMap()

	registeredApprovedProcessor := processorMap[SubscriptionApproved]
	assert.NotNil(t, registeredApprovedProcessor)
	assert.NotEqual(t, 0, len(registeredApprovedProcessor))
	sf1 := reflect.ValueOf(approvedProcessor)
	sf2 := reflect.ValueOf(registeredApprovedProcessor[0])
	assert.Equal(t, sf1.Pointer(), sf2.Pointer(), "Verify registered approved subscription processor")

	registeredUnsubscribeProcessor := processorMap[SubscriptionUnsubscribeInitiated]
	assert.NotNil(t, registeredUnsubscribeProcessor)
	assert.NotEqual(t, 0, len(registeredUnsubscribeProcessor))
	sf1 = reflect.ValueOf(unsubscribeProcessor)
	sf2 = reflect.ValueOf(registeredUnsubscribeProcessor[0])

	assert.Equal(t, sf1.Pointer(), sf2.Pointer(), "Verify registered unsubscribe initiated subscription processor")
}

func createSubscription(ID, state, catalogID string, subscriptionProps map[string]string) Subscription {
	return Subscription{
		ID:    ID,
		State: state,
		Properties: []SubscriptionProperties{
			SubscriptionProperties{
				Key:    "profile",
				Values: subscriptionProps,
			},
		},
		CatalogItemID: catalogID,
	}
}

func TestSubscriptionManagerPollDisconnectedMode(t *testing.T) {
	// Start a local HTTP server
	subscriptionList := make([]Subscription, 0)
	subscriptionList = append(subscriptionList, createSubscription("11111", "APPROVED", "11111", map[string]string{"orgId": "11111"}))
	subscriptionList = append(subscriptionList, createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]string{"orgId": "22222"}))
	subscriptionList = append(subscriptionList, createSubscription("33333", "APPROVED", "33333", map[string]string{"orgId": "33333"}))
	cache.GetCache().Set("1", "1_proxy")
	cache.GetCache().SetSecondaryKey("1", "11111")
	cache.GetCache().Set("2", "2_proxy")
	cache.GetCache().SetSecondaryKey("2", "22222")
	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("[]")
		if sendList {
			b, _ = json.Marshal(subscriptionList)
			sendList = false
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()

	cfg := &corecfg.CentralConfiguration{
		TeamID:       "test",
		URL:          server.URL,
		PollInterval: 1 * time.Second,
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)

	serviceClient.tokenRequester = &mockTokenGetter{
		token: "testToken",
	}
	approvedSubscriptions := make(map[string]*Subscription)
	unsubscribedSubscriptions := make(map[string]*Subscription)
	approvedProcessor := func(subscription Subscription) {
		approvedSubscriptions[subscription.ID] = &subscription
	}
	unsubscribedProcessor := func(subscription Subscription) {
		unsubscribedSubscriptions[subscription.ID] = &subscription
	}
	subscriptionValidator := func(subscription Subscription) bool {
		apiCache := cache.GetCache()
		api, _ := apiCache.GetBySecondaryKey(subscription.ApicID)
		return api != nil
	}

	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribedProcessor)
	client.GetSubscriptionManager().RegisterValidator(subscriptionValidator)
	client.GetSubscriptionManager().Start()

	time.Sleep(2 * time.Second)
	client.GetSubscriptionManager().Stop()

	assert.NotEqual(t, 0, len(approvedSubscriptions))
	// approved Subscription for API in cache
	assert.NotNil(t, approvedSubscriptions["11111"])
	assert.Equal(t, "11111", approvedSubscriptions["11111"].GetPropertyValue("orgId"))
	// approved Subscription for API in not cache, so not processed
	assert.Nil(t, approvedSubscriptions["33333"])

	// unsubscribe initiated Subscription for API in cache
	assert.NotNil(t, unsubscribedSubscriptions["22222"])
	assert.Equal(t, "22222", unsubscribedSubscriptions["22222"].GetPropertyValue("orgId"))
}

func TestSubscriptionManagerPollConnectedMode(t *testing.T) {
	// Start a local HTTP server
	subscriptionList := make([]Subscription, 0)
	subscriptionList = append(subscriptionList, createSubscription("11111", "APPROVED", "11111", map[string]string{"orgId": "11111"}))
	subscriptionList = append(subscriptionList, createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]string{"orgId": "22222"}))
	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")
		if strings.Contains(req.RequestURI, "/subscriptions") {
			if sendList {
				b, _ = json.Marshal(subscriptionList)
				sendList = false
			} else {
				b = []byte("[]")
			}
		}
		if strings.Contains(req.RequestURI, "/11111/properties/apiServerInfo") {
			serverInfo := APIServerInfo{
				ConsumerInstance: APIServerInfoProperty{Name: "11111", ID: "11111"},
				Environment:      APIServerInfoProperty{Name: "test", ID: "00000"},
			}
			b, _ = json.Marshal(serverInfo)
		}
		if strings.Contains(req.RequestURI, "/22222/properties/apiServerInfo") {
			serverInfo := APIServerInfo{
				ConsumerInstance: APIServerInfoProperty{Name: "22222", ID: "22222"},
				Environment:      APIServerInfoProperty{Name: "test", ID: "00000"},
			}
			b, _ = json.Marshal(serverInfo)
		}
		if strings.Contains(req.RequestURI, "/consumerinstances/11111") {
			apiserverRes := APIServer{
				Name:  "11111",
				Title: "ConsumerInstance_11111",
				Metadata: &APIServerMetadata{
					ID: "11111",
					References: []APIServerReference{
						APIServerReference{
							ID:   "11111",
							Kind: "APIServiceInstance",
						},
					},
				},
			}
			b, _ = json.Marshal(apiserverRes)
		}
		if strings.Contains(req.RequestURI, "/consumerinstances/22222") {
			apiserverRes := APIServer{
				Name:  "22222",
				Title: "ConsumerInstance_22222",
				Metadata: &APIServerMetadata{
					ID: "22222",
					References: []APIServerReference{
						APIServerReference{
							ID:   "22222",
							Kind: "APIServiceInstance",
						},
					},
				},
			}
			b, _ = json.Marshal(apiserverRes)
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()

	cfg := &corecfg.CentralConfiguration{
		Mode:                 corecfg.Connected,
		TeamID:               "test",
		URL:                  server.URL,
		PollInterval:         1 * time.Second,
		APIServerEnvironment: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)
	serviceClient.tokenRequester = &mockTokenGetter{
		token: "testToken",
	}
	approvedSubscriptions := make(map[string]*Subscription)
	unsubscribedSubscriptions := make(map[string]*Subscription)
	approvedProcessor := func(subscription Subscription) {
		approvedSubscriptions[subscription.ID] = &subscription
	}
	unsubscribedProcessor := func(subscription Subscription) {
		unsubscribedSubscriptions[subscription.ID] = &subscription
	}
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribedProcessor)
	client.GetSubscriptionManager().Start()

	time.Sleep(2 * time.Second)
	client.GetSubscriptionManager().Stop()

	assert.NotEqual(t, 0, len(approvedSubscriptions))
	// approved Subscription for API in cache
	assert.NotNil(t, approvedSubscriptions["11111"])
	assert.Equal(t, "11111", approvedSubscriptions["11111"].GetPropertyValue("orgId"))

	// unsubscribe initiated Subscription for API in cache
	assert.NotNil(t, unsubscribedSubscriptions["22222"])
	assert.Equal(t, "22222", unsubscribedSubscriptions["22222"].GetPropertyValue("orgId"))
}

func TestSubscriptionUpdate(t *testing.T) {
	// Start a local HTTP server
	subscriptionMap := make(map[string]*Subscription)
	sub1 := createSubscription("11111", "APPROVED", "11111", map[string]string{"orgId": "11111"})
	sub2 := createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]string{"orgId": "22222"})
	subscriptionMap["11111"] = &sub1
	subscriptionMap["22222"] = &sub2

	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")
		if strings.Contains(req.RequestURI, "/subscriptions") {
			if sendList {
				subscriptionList := make([]*Subscription, 0)
				for _, subscription := range subscriptionMap {
					subscriptionList = append(subscriptionList, subscription)
				}
				b, _ = json.Marshal(subscriptionList)
				sendList = false
			} else {
				b = []byte("[]")
			}
		}
		if strings.Contains(req.RequestURI, "/11111/subscriptions/11111/state") {
			subState := make(map[string]string)
			json.NewDecoder(req.Body).Decode(&subState)
			subscription := subscriptionMap["11111"]
			subscription.State = subState["state"]
		}
		if strings.Contains(req.RequestURI, "/22222/subscriptions/22222/state") {
			subState := make(map[string]string)
			json.NewDecoder(req.Body).Decode(&subState)
			subscription := subscriptionMap["22222"]
			subscription.State = subState["state"]
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()

	cfg := &corecfg.CentralConfiguration{
		TeamID:       "test",
		URL:          server.URL,
		PollInterval: 1 * time.Second,
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	client := New(cfg)
	assert.NotNil(t, client)
	serviceClient := client.(*ServiceClient)
	assert.NotNil(t, serviceClient)

	serviceClient.tokenRequester = &mockTokenGetter{
		token: "testToken",
	}
	approvedProcessor := func(subscription Subscription) {
		subscription.UpdateState(SubscriptionActive)
	}
	unsubscribedProcessor := func(subscription Subscription) {
		subscription.UpdateState(SubscriptionUnsubscribed)
	}

	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribedProcessor)
	client.GetSubscriptionManager().Start()

	time.Sleep(2 * time.Second)
	client.GetSubscriptionManager().Stop()

	assert.Equal(t, string(SubscriptionActive), subscriptionMap["11111"].State)
	assert.Equal(t, string(SubscriptionUnsubscribed), subscriptionMap["22222"].State)
}
