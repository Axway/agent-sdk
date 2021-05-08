package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
)

type MockSvcClient struct {
	apiSvc *v1alpha1.APIService
}

func (m *MockSvcClient) SetTokenGetter(tokenGetter auth.PlatformTokenGetter) {}
func (m *MockSvcClient) PublishService(serviceBody apic.ServiceBody) (*v1alpha1.APIService, error) {
	return m.apiSvc, nil
}
func (m *MockSvcClient) RegisterSubscriptionWebhook() error { return nil }
func (m *MockSvcClient) RegisterSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema, update bool) error {
	return nil
}
func (m *MockSvcClient) UpdateSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}
func (m *MockSvcClient) GetSubscriptionManager() apic.SubscriptionManager { return nil }
func (m *MockSvcClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return "", nil
}
func (m *MockSvcClient) DeleteConsumerInstance(instanceName string) error { return nil }
func (m *MockSvcClient) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	return nil
}
func (m *MockSvcClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
func (m *MockSvcClient) GetUserEmailAddress(ID string) (string, error) { return "", nil }
func (m *MockSvcClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]apic.CentralSubscription, error) {
	return nil, nil
}
func (m *MockSvcClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	return nil, nil
}
func (m *MockSvcClient) Healthcheck(name string) *hc.Status {
	return &hc.Status{Result: hc.OK}
}

// UpdateSubscriptionDefinitionPropertiesForCatalogItem -
func (m *MockSvcClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}

func (m *MockSvcClient) GetCatalogItemName(ID string) (string, error) { return "", nil }
func (m *MockSvcClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}
func (m *MockSvcClient) OnConfigChange(cfg config.CentralConfig) {}

var oldUpdateCacheForExternalAPIID = updateCacheForExternalAPIID
var oldUpdateCacheForExternalAPIName = updateCacheForExternalAPIName
var oldUpdateCacheForExternalAPI = updateCacheForExternalAPI

func fakeCacheUpdateCalls() {
	updateCacheForExternalAPIID = func(string) (interface{}, error) { return nil, nil }
	updateCacheForExternalAPIName = func(string) (interface{}, error) { return nil, nil }
	updateCacheForExternalAPI = func(map[string]string) (interface{}, error) { return nil, nil }
}

func restoreCacheUpdateCalls() {
	updateCacheForExternalAPIID = oldUpdateCacheForExternalAPIID
	updateCacheForExternalAPIName = oldUpdateCacheForExternalAPIName
	updateCacheForExternalAPI = oldUpdateCacheForExternalAPI
}

func TestDiscoveryCache(t *testing.T) {
	fakeCacheUpdateCalls()
	attributeKey := "Attr1"
	attributeValue := "testValue"
	emptyAPISvc := []v1.ResourceInstance{}
	apiSvc1 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService1",
			Attributes: map[string]string{
				apic.AttrExternalAPIID:         "1111",
				apic.AttrExternalAPIPrimaryKey: "1234",
				apic.AttrExternalAPIName:       "NAME",
				attributeKey:                   attributeValue,
			},
		},
	}
	apiSvc2 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService2",
			Attributes: map[string]string{
				apic.AttrExternalAPIID: "2222",
			},
		},
	}
	var serverAPISvcResponse []v1.ResourceInstance
	// var apiSvc *v1.ResourceInstance
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.RequestURI, "/auth") {
			token := "{\"access_token\":\"somevalue\",\"expires_in\": 12235677}"
			resp.Write([]byte(token))
		}
		if strings.Contains(req.RequestURI, "/apis/management/v1alpha1/environments/test/apiservices") {
			buf, _ := json.Marshal(serverAPISvcResponse)
			resp.Write(buf)
		}
	}))
	defer s.Close()

	cfg := createCentralCfg(s.URL, "test")
	resetResources()
	err := Initialize(cfg)
	assert.Nil(t, err)

	serverAPISvcResponse = emptyAPISvc
	updateAPICache()
	assert.Equal(t, 0, len(agent.apiMap.GetKeys()))
	assert.False(t, IsAPIPublished("1111"))
	assert.False(t, IsAPIPublished("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	updateAPICache()
	assert.Equal(t, 1, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublished("1111"))
	assert.False(t, IsAPIPublished("2222"))
	assert.Equal(t, "1111", GetAttributeOnPublishedAPIByID("1111", apic.AttrExternalAPIID))
	assert.Equal(t, "", GetAttributeOnPublishedAPI("2222", apic.AttrExternalAPIID))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByPrimaryKey("1234", attributeKey))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByName("NAME", attributeKey))

	apicClient := agent.apicClient
	var apiSvc v1alpha1.APIService
	apiSvc.FromInstance(&apiSvc2)
	agent.apicClient = &MockSvcClient{apiSvc: &apiSvc}
	StartPeriodicStatusUpdate()
	PublishAPI(apic.ServiceBody{})
	agent.apicClient = apicClient
	assert.Equal(t, 2, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublished("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	updateAPICache()
	assert.Equal(t, 1, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublished("1111"))
	assert.True(t, IsAPIPublishedByPrimaryKey("1234"))
	assert.False(t, IsAPIPublished("2222"))

	restoreCacheUpdateCalls()
}
