package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/auth"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/stretchr/testify/assert"
)

type mockSvcClient struct {
	apiSvc *v1alpha1.APIService
}

func (m *mockSvcClient) GetAPIRevisions(queryParams map[string]string, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceInstances(queryParams map[string]string, URL string) ([]*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceRevisions(queryParams map[string]string, URL, stage string) ([]*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIV1ResourceInstancesWithPageSize(queryParams map[string]string, URL string, pageSize int) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIV1ResourceInstances(queryParams map[string]string, URL string) ([]*v1.ResourceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) CreateCategory(categoryName string) (*catalog.Category, error) {
	return nil, nil
}

func (m *mockSvcClient) AddCategoryCache(categoryCache cache.Cache) {
	return
}

func (m *mockSvcClient) GetOrCreateCategory(category string) string {
	return ""
}

func (m *mockSvcClient) GetAPIServiceByName(serviceName string) (*v1alpha1.APIService, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIRevisionByName(revisionName string) (*v1alpha1.APIServiceRevision, error) {
	return nil, nil
}

func (m *mockSvcClient) GetAPIServiceInstanceByName(instanceName string) (*v1alpha1.APIServiceInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) SetTokenGetter(tokenGetter auth.PlatformTokenGetter) {}
func (m *mockSvcClient) PublishService(serviceBody *apic.ServiceBody) (*v1alpha1.APIService, error) {
	return m.apiSvc, nil
}
func (m *mockSvcClient) RegisterSubscriptionWebhook() error { return nil }
func (m *mockSvcClient) RegisterSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema, update bool) error {
	return nil
}
func (m *mockSvcClient) UpdateSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}
func (m *mockSvcClient) GetSubscriptionManager() apic.SubscriptionManager { return nil }
func (m *mockSvcClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return "", nil
}
func (m *mockSvcClient) DeleteServiceByAPIID(externalAPIID string) error    { return nil }
func (m *mockSvcClient) DeleteConsumerInstance(instanceName string) error   { return nil }
func (m *mockSvcClient) DeleteAPIServiceInstance(instanceName string) error { return nil }
func (m *mockSvcClient) UpdateConsumerInstanceSubscriptionDefinition(externalAPIID, subscriptionDefinitionName string) error {
	return nil
}
func (m *mockSvcClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
func (m *mockSvcClient) GetConsumerInstancesByExternalAPIID(consumerInstanceID string) ([]*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}

func (m *mockSvcClient) GetUserName(ID string) (string, error)         { return "", nil }
func (m *mockSvcClient) GetUserEmailAddress(ID string) (string, error) { return "", nil }
func (m *mockSvcClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]apic.CentralSubscription, error) {
	return nil, nil
}
func (m *mockSvcClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	return nil, nil
}
func (m *mockSvcClient) Healthcheck(name string) *hc.Status {
	return &hc.Status{Result: hc.OK}
}

// UpdateSubscriptionDefinitionPropertiesForCatalogItem -
func (m *mockSvcClient) UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string, subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}

func (m *mockSvcClient) GetCatalogItemName(ID string) (string, error) { return "", nil }
func (m *mockSvcClient) ExecuteAPI(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
	return nil, nil
}
func (m *mockSvcClient) OnConfigChange(cfg config.CentralConfig) {}

func (m *mockSvcClient) SetConfig(cfg corecfg.CentralConfig) {
}

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
	dcj := newDiscoveryCache(true)
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
	dcj.updateAPICache()
	assert.Equal(t, 0, len(agent.apiMap.GetKeys()))
	assert.False(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.False(t, IsAPIPublishedByID("2222"))
	assert.Equal(t, "1111", GetAttributeOnPublishedAPIByID("1111", apic.AttrExternalAPIID))
	assert.Equal(t, "", GetAttributeOnPublishedAPIByID("2222", apic.AttrExternalAPIID))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByPrimaryKey("1234", attributeKey))
	assert.Equal(t, attributeValue, GetAttributeOnPublishedAPIByName("NAME", attributeKey))

	apicClient := agent.apicClient
	var apiSvc v1alpha1.APIService
	apiSvc.FromInstance(&apiSvc2)
	agent.apicClient = &mockSvcClient{apiSvc: &apiSvc}
	StartAgentStatusUpdate()
	PublishAPI(apic.ServiceBody{})
	agent.apicClient = apicClient
	assert.Equal(t, 2, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByID("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	dcj.updateAPICache()
	assert.Equal(t, 1, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublishedByID("1111"))
	assert.True(t, IsAPIPublishedByPrimaryKey("1234"))
	assert.False(t, IsAPIPublishedByID("2222"))

	restoreCacheUpdateCalls()
}
