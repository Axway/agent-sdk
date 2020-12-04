package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic"
	v1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

type mockSvcClient struct {
	apiSvc *v1alpha1.APIService
}

func (m *mockSvcClient) PublishService(serviceBody apic.ServiceBody) (*v1alpha1.APIService, error) {
	return m.apiSvc, nil
}
func (m *mockSvcClient) RegisterSubscriptionWebhook() error { return nil }
func (m *mockSvcClient) RegisterSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}
func (m *mockSvcClient) UpdateSubscriptionSchema(subscriptionSchema apic.SubscriptionSchema) error {
	return nil
}
func (m *mockSvcClient) GetSubscriptionManager() apic.SubscriptionManager { return nil }
func (m *mockSvcClient) GetCatalogItemIDForConsumerInstance(instanceID string) (string, error) {
	return "", nil
}
func (m *mockSvcClient) DeleteConsumerInstance(instanceName string) error { return nil }
func (m *mockSvcClient) GetConsumerInstanceByID(consumerInstanceID string) (*v1alpha1.ConsumerInstance, error) {
	return nil, nil
}
func (m *mockSvcClient) GetUserEmailAddress(ID string) (string, error) { return "", nil }
func (m *mockSvcClient) GetSubscriptionsForCatalogItem(states []string, instanceID string) ([]apic.CentralSubscription, error) {
	return nil, nil
}
func (m *mockSvcClient) GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, propertyKey string) (apic.SubscriptionSchema, error) {
	return nil, nil
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
func TestDiscoveryCache(t *testing.T) {
	emptyAPISvc := []v1.ResourceInstance{}
	apiSvc1 := v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.APIServiceGVK(),
			Name:             "testAPIService1",
			Attributes: map[string]string{
				apic.AttrExternalAPIID: "1111",
				"Attr1":                "testValue",
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
	assert.Equal(t, "1111", GetAttributeOnPublishedAPI("1111", apic.AttrExternalAPIID))
	assert.Equal(t, "", GetAttributeOnPublishedAPI("2222", apic.AttrExternalAPIID))

	apicClient := agent.apicClient
	var apiSvc v1alpha1.APIService
	apiSvc.FromInstance(&apiSvc2)
	agent.apicClient = &mockSvcClient{apiSvc: &apiSvc}
	PublishAPI(apic.ServiceBody{})
	agent.apicClient = apicClient
	assert.Equal(t, 2, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublished("1111"))
	assert.True(t, IsAPIPublished("2222"))

	serverAPISvcResponse = []v1.ResourceInstance{apiSvc1}
	updateAPICache()
	assert.Equal(t, 1, len(agent.apiMap.GetKeys()))
	assert.True(t, IsAPIPublished("1111"))
	assert.False(t, IsAPIPublished("2222"))
}
