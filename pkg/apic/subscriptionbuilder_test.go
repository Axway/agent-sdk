package apic

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

type subscriptionMock struct {
	Subscription
	catalogID     string
	serviceClient *ServiceClient
	updateErr     error
}

func (s *subscriptionMock) GetID() string                              { return "" }
func (s *subscriptionMock) GetName() string                            { return "" }
func (s *subscriptionMock) GetApicID() string                          { return "" }
func (s *subscriptionMock) GetRemoteAPIID() string                     { return "" }
func (s *subscriptionMock) GetRemoteAPIStage() string                  { return "" }
func (s *subscriptionMock) GetCatalogItemID() string                   { return s.catalogID }
func (s *subscriptionMock) GetCreatedUserID() string                   { return "" }
func (s *subscriptionMock) GetState() SubscriptionState                { return SubscriptionApproved }
func (s *subscriptionMock) GetServiceClient() *ServiceClient           { return s.serviceClient }
func (s *subscriptionMock) GetPropertyValue(propertyKey string) string { return "" }
func (s *subscriptionMock) UpdateState(newState SubscriptionState, description string) error {
	return nil
}
func (s *subscriptionMock) UpdateProperties(appName string) error { return nil }
func (s *subscriptionMock) UpdatePropertyValues(values map[string]interface{}) error {
	return s.updateErr
}

func TestNewSubscriptionBuilder(t *testing.T) {
	subscriptionMock := &subscriptionMock{}
	builder := NewSubscriptionBuilder(subscriptionMock)
	err := builder.Process()
	assert.Nil(t, err)

	// test all the default values
	assert.Nil(t, builder.(*subscriptionBuilder).err)
	assert.Len(t, builder.(*subscriptionBuilder).propertyValues, 0)
	assert.Equal(t, subscriptionMock, builder.(*subscriptionBuilder).subscription)
}

func TestSubscriptionBuilderFuncs(t *testing.T) {
	wd, _ := os.Getwd()
	subscription := &subscriptionMock{}
	builder := NewSubscriptionBuilder(subscription).
		SetStringPropertyValue("key1", "value1").
		SetStringPropertyValue("key1", "value2")

	err := builder.Process()
	assert.NotNil(t, err)

	subscription = &subscriptionMock{
		updateErr: fmt.Errorf("error"),
	}
	builder = NewSubscriptionBuilder(subscription).
		SetStringPropertyValue("key1", "value1").
		SetStringPropertyValue("key2", "value2")

	err = builder.Process()
	assert.NotNil(t, err)

	testServiceClient, mockHTTPClient := GetTestServiceClient()
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile2.json",
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusOK,
		},
	})
	subscription = &subscriptionMock{
		catalogID:     "1234",
		serviceClient: testServiceClient,
	}
	builder = NewSubscriptionBuilder(subscription).
		UpdateEnumProperty("appName", "value1", "string").
		SetStringPropertyValue("appName", "value1").
		SetStringPropertyValue("appID", "value2")

	err = builder.Process()
	// check builder properties
	assert.Len(t, builder.(*subscriptionBuilder).propertyValues, 2)
	assert.Equal(t, "value1", builder.(*subscriptionBuilder).propertyValues["appName"])
	assert.Equal(t, "value2", builder.(*subscriptionBuilder).propertyValues["appID"])
	assert.Nil(t, err)

	testServiceClient, mockHTTPClient = GetTestServiceClient()
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile2.json",
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusNotFound,
		},
	})
	subscription = &subscriptionMock{
		catalogID:     "1234",
		serviceClient: testServiceClient,
	}
	err = NewSubscriptionBuilder(subscription).
		UpdateEnumProperty("appName", "value1", "string").
		SetStringPropertyValue("appName", "value1").
		SetStringPropertyValue("appID", "value2").
		Process()
	assert.NotNil(t, err)

	testServiceClient, mockHTTPClient = GetTestServiceClient()
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile2.json",
			RespCode: http.StatusNotFound,
		},
	})
	subscription = &subscriptionMock{
		catalogID:     "1234",
		serviceClient: testServiceClient,
	}
	err = NewSubscriptionBuilder(subscription).
		UpdateEnumProperty("appName", "value1", "string").
		SetStringPropertyValue("appName", "value1").
		SetStringPropertyValue("appID", "value2").
		Process()
	assert.NotNil(t, err)
}
