package apic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
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

func createSubscription(ID, state, catalogID string, subscriptionProps map[string]interface{}) Subscription {
	return &CentralSubscription{
		ApicID:         "1111",
		RemoteAPIID:    "2222",
		RemoteAPIStage: "stage",
		CatalogItemSubscription: &uc.CatalogItemSubscription{
			Id:    ID,
			Name:  "testsubscription",
			State: state,
			Properties: []uc.CatalogItemProperty{
				{
					Key:   "profile",
					Value: subscriptionProps,
				},
			},
			CatalogItemId: catalogID,
			Metadata: uc.AuditMetadata{
				CreateUserId: "bbunny",
			},
		},
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
	subscriptionList = append(subscriptionList, createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"}))
	subscriptionList = append(subscriptionList, createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]interface{}{"orgId": "22222"}))
	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")
		if strings.Contains(req.RequestURI, "/subscriptions") {
			if sendList {
				ucSubscriptionList := make([]uc.CatalogItemSubscription, 0)
				for _, sub := range subscriptionList {
					ucSubscriptionList = append(ucSubscriptionList, *(sub.(*CentralSubscription)).CatalogItemSubscription)
				}
				b, _ = json.Marshal(ucSubscriptionList)
				sendList = false
			} else {
				b = []byte("[]")
			}
		}
		if strings.Contains(req.RequestURI, "/11111/subscriptions/11111/relationships") {
			subsRelations := []uc.EntityRelationship{
				{Type: "API_SERVER_CONSUMER_INSTANCE_ID", Value: "11111", Key: "apiServerInfo"},
				{Type: "API_SERVER_CONSUMER_INSTANCE_NAME", Value: "11111", Key: "apiServerInfo"},
				{Type: "API_SERVER_ENVIRONMENT_ID", Value: "00000", Key: "apiServerInfo"},
				{Type: "API_SERVER_ENVIRONMENT_NAME", Value: "test", Key: "apiServerInfo"},
			}

			b, _ = json.Marshal(subsRelations)
		}
		if strings.Contains(req.RequestURI, "/22222/subscriptions/22222/relationships") {
			subsRelations := []uc.EntityRelationship{
				{Type: "API_SERVER_CONSUMER_INSTANCE_ID", Value: "22222", Key: "apiServerInfo"},
				{Type: "API_SERVER_CONSUMER_INSTANCE_NAME", Value: "22222", Key: "apiServerInfo"},
				{Type: "API_SERVER_ENVIRONMENT_ID", Value: "00000", Key: "apiServerInfo"},
				{Type: "API_SERVER_ENVIRONMENT_NAME", Value: "test", Key: "apiServerInfo"},
			}

			b, _ = json.Marshal(subsRelations)
		}
		if strings.Contains(req.RequestURI, "/consumerinstances/11111") {
			apiserverRes := v1alpha1.ConsumerInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1alpha1.ConsumerInstanceGVK(),
					Name:             "11111",
					Title:            "ConsumerInstance_11111",
					Metadata: v1.Metadata{
						ID: "11111",
						References: []v1.Reference{
							{
								ID:   "11111",
								Kind: "APIServiceInstance",
							},
						},
					},
				},
			}
			b, _ = json.Marshal(apiserverRes)
		}
		if strings.Contains(req.RequestURI, "/consumerinstances/22222") {
			apiserverRes := v1alpha1.ConsumerInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1alpha1.ConsumerInstanceGVK(),
					Name:             "22222",
					Title:            "ConsumerInstance_22222",
					Metadata: v1.Metadata{
						ID: "22222",
						References: []v1.Reference{
							{
								ID:   "22222",
								Kind: "APIServiceInstance",
							},
						},
					},
				},
			}
			b, _ = json.Marshal(apiserverRes)
		}
		if strings.Contains(req.RequestURI, "/apiserviceinstances/11111") {
			apiserverRes := v1alpha1.APIServiceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1alpha1.APIServiceInstanceGVK(),
					Name:             "11111",
					Title:            "APIServiceInstance_11111",
					Metadata: v1.Metadata{
						ID: "11111",
					},
					Attributes: map[string]string{
						definitions.AttrExternalAPIID: "1111",
					},
				},
			}
			b, _ = json.Marshal(apiserverRes)
		}
		if strings.Contains(req.RequestURI, "/apiserviceinstances/22222") {
			apiserverRes := v1alpha1.APIServiceInstance{
				ResourceMeta: v1.ResourceMeta{
					GroupVersionKind: v1alpha1.APIServiceInstanceGVK(),
					Name:             "22222",
					Title:            "APIServiceInstance_2222",
					Metadata: v1.Metadata{
						ID: "22222",
					},
					Attributes: map[string]string{
						definitions.AttrExternalAPIID: "2222",
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
	subscriptionMap["11111"] = createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"})
	subscriptionMap["22222"] = createSubscription("22222", "UNSUBSCRIBE_INITIATED", "22222", map[string]interface{}{"orgId": "22222"})

	sendList := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		b := []byte("")
		if strings.Contains(req.RequestURI, "/subscriptions") {
			if sendList {
				ucSubscriptionList := make([]uc.CatalogItemSubscription, 0)
				for _, sub := range subscriptionMap {
					ucSubscriptionList = append(ucSubscriptionList, *(sub.(*CentralSubscription)).CatalogItemSubscription)
				}
				b, _ = json.Marshal(ucSubscriptionList)
				sendList = false
			} else {
				b = []byte("[]")
			}
		}
		if strings.Contains(req.RequestURI, "/11111/subscriptions/11111/state") {
			subState := make(map[string]string)
			json.NewDecoder(req.Body).Decode(&subState)
			subscription := subscriptionMap["11111"]
			(subscription.(*CentralSubscription)).CatalogItemSubscription.State = subState["state"]
		}
		if strings.Contains(req.RequestURI, "/22222/subscriptions/22222/state") {
			subState := make(map[string]string)
			json.NewDecoder(req.Body).Decode(&subState)
			subscription := subscriptionMap["22222"]
			(subscription.(*CentralSubscription)).CatalogItemSubscription.State = subState["state"]
		}
		if strings.Contains(req.RequestURI, "11111/subscriptions/11111/relationships") {
			subsRelations := []uc.EntityRelationship{
				{Type: "API_SERVER_CONSUMER_INSTANCE_NAME", Value: "foo", Key: "apiServerInfo"},
				{Type: "API_SERVER_ENVIRONMENT_NAME", Value: "testenvironment", Key: "apiServerInfo"},
			}

			b, _ = json.Marshal(subsRelations)
		}
		// Send response to be tested
		rw.Write(b)
	}))
	// Close the server when test finishes
	defer server.Close()

	client, _ := createServiceClientForSubscriptions(server)
	assert.NotNil(t, client)

	approvedProcessor := func(subscription Subscription) {
		subscription.UpdateState(SubscriptionActive, "approved")
	}
	unsubscribedProcessor := func(subscription Subscription) {
		subscription.UpdateState(SubscriptionUnsubscribed, "unsubscribed")
	}

	client.GetSubscriptionManager().RegisterProcessor(SubscriptionApproved, approvedProcessor)
	client.GetSubscriptionManager().RegisterProcessor(SubscriptionUnsubscribeInitiated, unsubscribedProcessor)
	client.GetSubscriptionManager().Start()

	time.Sleep(5 * time.Second)
	client.GetSubscriptionManager().Stop()

	// assert.Equal(t, SubscriptionActive, (subscriptionMap["11111"]).GetState())
	// assert.Equal(t, SubscriptionUnsubscribed, (subscriptionMap["22222"]).GetState())
	assert.Equal(t, SubscriptionApproved, (subscriptionMap["11111"]).GetState())
	assert.Equal(t, SubscriptionUnsubscribeInitiated, (subscriptionMap["22222"]).GetState())
}

func TestLocklist(t *testing.T) {
	client, _ := GetTestServiceClient()
	mgr := client.GetSubscriptionManager().(*subscriptionManager)
	mgr.addLocklistItem("123")
	assert.Equal(t, 1, len(mgr.locklist))
	mgr.addLocklistItem("456")
	assert.Equal(t, 2, len(mgr.locklist))
	assert.True(t, mgr.isItemOnLocklist("123"))
	mgr.removeLocklistItem("123")
	assert.Equal(t, 1, len(mgr.locklist))
	assert.False(t, mgr.isItemOnLocklist("123"))
	mgr.removeLocklistItem("456")
	assert.Equal(t, 0, len(mgr.locklist))
}
