package apic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	coreapi "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestProcessorRegistration(t *testing.T) {
	client, _ := GetTestServiceClient()
	assert.NotNil(t, client)

	approvedProcessor := func(subscription Subscription) {}
	unsubscribeProcessor := func(subscription Subscription) {}

	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribeProcessor)

	processorMap := client.subscriptionMgr.getProcessorMap()

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
	return &CentralSubscription{
		ID:    ID,
		State: state,
		Properties: []SubscriptionProperties{
			{
				Key:    "profile",
				Values: subscriptionProps,
			},
		},
		CatalogItemID: catalogID,
	}
}

func createServiceClientForSubscriptions(server *httptest.Server) (*ServiceClient, *corecfg.CentralConfiguration) {
	client, _ := GetTestServiceClient()
	cfg := GetTestServiceClientCentralConfiguration(client)
	cfg.URL = server.URL
	client.apiClient = coreapi.NewClient(nil, "")
	return client, cfg
}

func TestSubscriptionManagerPollPublishToEnvironmentMode(t *testing.T) {
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
						{
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
						{
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

	client, cfg := createServiceClientForSubscriptions(server)
	assert.NotNil(t, client)
	cfg.Mode = corecfg.PublishToEnvironment
	cfg.Environment = "test"

	approvedSubscriptions := make(map[string]Subscription)
	unsubscribedSubscriptions := make(map[string]Subscription)
	approvedProcessor := func(subscription Subscription) {
		approvedSubscriptions[subscription.GetID()] = subscription
	}
	unsubscribedProcessor := func(subscription Subscription) {
		unsubscribedSubscriptions[subscription.GetID()] = subscription
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
	subscriptionMap := make(map[string]Subscription)
	subscriptionMap["11111"] = createSubscription("11111", "APPROVED", "11111", map[string]string{"orgId": "11111"})
	subscriptionMap["22222"] = createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]string{"orgId": "22222"})

	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")
		if strings.Contains(req.RequestURI, "/subscriptions") {
			if sendList {
				subscriptionList := make([]Subscription, 0)
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
			(subscription.(*CentralSubscription)).State = subState["state"]
		}
		if strings.Contains(req.RequestURI, "/22222/subscriptions/22222/state") {
			subState := make(map[string]string)
			json.NewDecoder(req.Body).Decode(&subState)
			subscription := subscriptionMap["22222"]
			(subscription.(*CentralSubscription)).State = subState["state"]
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()

	client, _ := createServiceClientForSubscriptions(server)
	assert.NotNil(t, client)

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

	assert.Equal(t, SubscriptionActive, subscriptionMap["11111"].GetState())
	assert.Equal(t, SubscriptionUnsubscribed, subscriptionMap["22222"].GetState())
}

func TestBlacklist(t *testing.T) {
	client, _ := GetTestServiceClient()
	mgr := client.GetSubscriptionManager().(*subscriptionManager)
	mgr.AddBlacklistItem("123")
	assert.Equal(t, 1, len(mgr.blacklist))
	mgr.AddBlacklistItem("456")
	assert.Equal(t, 2, len(mgr.blacklist))

	mgr.RemoveBlacklistItem("123")
	assert.Equal(t, 1, len(mgr.blacklist))
	mgr.RemoveBlacklistItem("456")
	assert.Equal(t, 0, len(mgr.blacklist))
}
