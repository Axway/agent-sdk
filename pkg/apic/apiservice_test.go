package apic

import (
	"net/http"
	"strconv"
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
	corecfg "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

var serviceBody = ServiceBody{
	// NameToPush:       nameToPush,
	APIName: "daleapi",
	// RestAPIID:        proxy.ID,
	// URL:              url,
	// TeamID:           teamID,
	// Description:      description,
	// Version:          version,
	// AuthPolicy:       authType,
	// Swagger:       []byte(swagger),
	Documentation: []byte("\"docs\""),
	// Tags:             tags,
	// AgentMode:        a.getAgentMode(),
	Image:            "abcde",
	ImageContentType: "image/jpeg",
	// CreatedBy:        corecmd.BuildAgentName,
	ResourceType: Oas2,
	// SubscriptionName: proxy.OrganizationID,
}

func newServiceClient() *ServiceClient {
	webhook := &corecfg.WebhookConfiguration{
		URL:     "http://foo.bar",
		Headers: "Header=contentType,Value=application/json",
		Secret:  "",
	}
	cfg := &corecfg.CentralConfiguration{
		Mode: corecfg.PublishToEnvironmentAndCatalog,
		Auth: &corecfg.AuthConfiguration{
			URL: "http://localhost:8888",
		},
		SubscriptionApprovalMode:    "webhook",
		SubscriptionApprovalWebhook: webhook,
	}
	return &ServiceClient{
		cfg:                                cfg,
		tokenRequester:                     MockTokenGetter,
		apiClient:                          &api.MockClient{ResponseCode: http.StatusOK},
		DefaultSubscriptionApprovalWebhook: webhook,
	}
}

func TestIsValidAuthPolicy(t *testing.T) {
	assert.False(t, isValidAuthPolicy("foobar"))
	assert.True(t, isValidAuthPolicy(Apikey))
	assert.True(t, isValidAuthPolicy(Passthrough))
	assert.True(t, isValidAuthPolicy(Oauth))
}

func TestCreateService(t *testing.T) {
	client := newServiceClient()
	mockClient := setupMocks(client)

	// this should be a full go right path
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // this for call to create the service
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			respCode: http.StatusOK,
		},
	}

	svcID, err := client.createService(serviceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svcID)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)

	// this should fail
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // this for call to create the service
			respCode: http.StatusRequestTimeout,
		},
	}
	mockClient.respCount = 0

	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // this for call to create the service
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/instancenotfound.json", // this for call to create the serviceInstance
			respCode: http.StatusNoContent,
		},
		{
			respCode: http.StatusOK, // this for call to rollback
		},
	}

	mockClient.respCount = 0
	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // this for call to create the service
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			respCode: http.StatusRequestTimeout,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // this for call to create the service
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			respCode: http.StatusCreated,
		},
		{
			fileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			respCode: http.StatusRequestTimeout,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)
}

func TestUpdateService(t *testing.T) {
	client := newServiceClient()
	mockClient := setupMocks(client)

	// this should be a full go right path
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // for call to update the service
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/consumerinstance.json", // for call to check existance of the consumerInstance
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			respCode: http.StatusOK,
		},
	}

	svcID, err := client.updateService(serviceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svcID)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)

	// this is a failure test
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json",
			respCode: http.StatusRequestTimeout,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is a failure test
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // for call to update the service
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			respCode: http.StatusRequestTimeout,
		},
		{
			fileName: "./testdata/instancenotfound.json", // for call to update the serviceInstance
			respCode: http.StatusNoContent,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is a failure test
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // for call to update the service
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			respCode: http.StatusRequestTimeout,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is another success test
	mockClient.responses = []mockResponse{
		{
			fileName: "./testdata/apiservice.json", // for call to update the service
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			respCode: http.StatusOK,
		},
		{
			fileName: "./testdata/consumerinstance.json", // for call to test if consumerInstanceExists
			respCode: http.StatusNotFound,
		},
		{
			fileName: "./testdata/consumerinstance.json", // for call to create the consumerInstance
			respCode: http.StatusOK,
		},
	}

	mockClient.respCount = 0
	svcID, err = client.updateService(serviceBody)
	assert.Nil(t, err)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)
}

func TestDeleteConsumerInstance(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)
	mock.ResponseCode = http.StatusRequestTimeout
	err := client.deleteConsumerInstance("12345")
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), strconv.Itoa(http.StatusRequestTimeout))

	mock.ResponseCode = http.StatusNoContent
	err = client.deleteConsumerInstance("12345")
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), strconv.Itoa(http.StatusNoContent))

	mock.ResponseCode = http.StatusOK
	err = client.deleteConsumerInstance("12345")
	assert.Nil(t, err)
}

func TestGetConsumerInstanceByID(t *testing.T) {
	client := newServiceClient()
	mock := client.apiClient.(*api.MockClient)

	// bad
	mock.SetResponse("./testdata/instancenotfound.json", http.StatusBadRequest)
	instance, err := client.GetConsumerInstanceByID("")
	assert.NotNil(t, err)
	assert.Nil(t, instance)

	// not found
	mock.SetResponse("./testdata/instancenotfound.json", http.StatusOK)
	instance, err = client.GetConsumerInstanceByID("e4ecaab773dbc4850173e45f35b8026g")
	assert.NotNil(t, err)
	assert.Nil(t, instance)

	// good
	mock.SetResponse("./testdata/consumerinstancelist.json", http.StatusOK)
	instance, err = client.GetConsumerInstanceByID("e4ecaab773dbc4850173e45f35b8026f")
	assert.Nil(t, err)
	assert.Equal(t, "daleapi", instance.Name)
}
func TestRegisterSubscriptionWebhook(t *testing.T) {
	client := newServiceClient()
	mockClient := setupMocks(client)

	// go right
	mockClient.responses = []mockResponse{
		{
			respCode: http.StatusCreated, // for call to createSecret
		},
		{
			respCode: http.StatusCreated, // for call to createWebhook
		},
	}

	err := client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)

	// go wrong
	mockClient.responses = []mockResponse{
		{
			respCode: http.StatusConflict, // for call to createSecret
		},
		{
			respCode: http.StatusOK, // for call to update the secret
		},
		{
			respCode: http.StatusRequestTimeout, // for call to createWebhook
		},
	}

	mockClient.respCount = 0
	err = client.RegisterSubscriptionWebhook()
	assert.NotNil(t, err)

	// go right
	mockClient.responses = []mockResponse{
		{
			respCode: http.StatusConflict, // for call to createSecret
		},
		{
			respCode: http.StatusOK, // for call to update the secret
		},
		{
			respCode: http.StatusCreated, // for call to createWebhook
		},
	}

	mockClient.respCount = 0
	err = client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)

	// go right
	mockClient.responses = []mockResponse{
		{
			respCode: http.StatusCreated, // for call to createSecret
		},
		{
			respCode: http.StatusConflict, // for call to createWebhook
		},
		{
			respCode: http.StatusOK, // for call to update the webhook
		},
	}

	mockClient.respCount = 0
	err = client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)
}
