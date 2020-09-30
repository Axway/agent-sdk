package apic

import (
	"net/http"
	"strconv"
	"testing"

	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/api"
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

func TestIsValidAuthPolicy(t *testing.T) {
	assert.False(t, isValidAuthPolicy("foobar"))
	assert.True(t, isValidAuthPolicy(Apikey))
	assert.True(t, isValidAuthPolicy(Passthrough))
	assert.True(t, isValidAuthPolicy(Oauth))
}

func TestCreateService(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// this should be a full go right path
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
	})

	svcID, err := client.createService(serviceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svcID)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusRequestTimeout,
		},
	})

	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/instancenotfound.json", // this for call to create the serviceInstance
			RespCode: http.StatusNoContent,
		},
		{
			RespCode: http.StatusOK, // this for call to rollback
		},
	})

	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusRequestTimeout,
		},
	})

	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusRequestTimeout,
		},
		{
			RespCode: http.StatusOK, // this for call to rollback
		},
	})

	svcID, err = client.createService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)
}

func TestUpdateService(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// this should be a full go right path
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to check existance of the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
	})

	svcID, err := client.updateService(serviceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svcID)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)

	// this is a failure test
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json",
			RespCode: http.StatusRequestTimeout,
		},
	})

	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is a failure test
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/instancenotfound.json", // for call to update the serviceInstance
			RespCode: http.StatusNoContent,
		},
	})

	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is a failure test
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to test if consumerInstanceExists
			RespCode: http.StatusNotFound,
		},
	})

	svcID, err = client.updateService(serviceBody)
	assert.NotNil(t, err)
	assert.Equal(t, "", svcID)

	// this is another success test
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
	})

	svcID, err = client.updateService(serviceBody)
	assert.Nil(t, err)
	assert.Equal(t, "e4ecaab773dbc4850173e45f35b8026f", svcID)
}

func TestDeleteConsumerInstance(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	httpClient.ResponseCode = http.StatusRequestTimeout
	err := client.deleteConsumerInstance("12345")
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), strconv.Itoa(http.StatusRequestTimeout))

	httpClient.ResponseCode = http.StatusNoContent
	err = client.deleteConsumerInstance("12345")
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), strconv.Itoa(http.StatusNoContent))

	httpClient.ResponseCode = http.StatusOK
	err = client.deleteConsumerInstance("12345")
	assert.Nil(t, err)
}

func TestGetConsumerInstanceByID(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// bad
	httpClient.SetResponse("./testdata/instancenotfound.json", http.StatusBadRequest)
	instance, err := client.GetConsumerInstanceByID("")
	assert.NotNil(t, err)
	assert.Nil(t, instance)

	// not found
	httpClient.SetResponse("./testdata/instancenotfound.json", http.StatusOK)
	instance, err = client.GetConsumerInstanceByID("e4ecaab773dbc4850173e45f35b8026g")
	assert.NotNil(t, err)
	assert.Nil(t, instance)

	// good
	httpClient.SetResponse("./testdata/consumerinstancelist.json", http.StatusOK)
	instance, err = client.GetConsumerInstanceByID("e4ecaab773dbc4850173e45f35b8026f")
	assert.Nil(t, err)
	assert.Equal(t, "daleapi", instance.Name)
}
func TestRegisterSubscriptionWebhook(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// go right
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusCreated, // for call to createSecret
		},
		{
			RespCode: http.StatusCreated, // for call to createWebhook
		},
	})

	err := client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)

	// go wrong
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusConflict, // for call to createSecret
		},
		{
			RespCode: http.StatusOK, // for call to update the secret
		},
		{
			RespCode: http.StatusRequestTimeout, // for call to createWebhook
		},
	})

	err = client.RegisterSubscriptionWebhook()
	assert.NotNil(t, err)

	// go right
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusConflict, // for call to createSecret
		},
		{
			RespCode: http.StatusOK, // for call to update the secret
		},
		{
			RespCode: http.StatusCreated, // for call to createWebhook
		},
	})

	err = client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)

	// go right
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusCreated, // for call to createSecret
		},
		{
			RespCode: http.StatusConflict, // for call to createWebhook
		},
		{
			RespCode: http.StatusOK, // for call to update the webhook
		},
	})

	err = client.RegisterSubscriptionWebhook()
	assert.Nil(t, err)
}
