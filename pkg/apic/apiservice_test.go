package apic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var testCategories = []string{"CategoryA", "CategoryB", "CategoryC"}

var serviceBody = ServiceBody{
	APIName:          "daleapi",
	Documentation:    []byte("\"docs\""),
	Image:            "abcde",
	ImageContentType: "image/jpeg",
	ResourceType:     Oas2,
	categoryTitles:   testCategories,
}

func TestIsValidAuthPolicy(t *testing.T) {
	assert.False(t, isValidAuthPolicy("foobar"))
	assert.True(t, isValidAuthPolicy(Apikey))
	assert.True(t, isValidAuthPolicy(Passthrough))
	assert.True(t, isValidAuthPolicy(Oauth))
}

func TestCreateService(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	serviceBody.AuthPolicy = "pass-through"

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

	// Setup category cache
	categoryCache := cache.New()
	teamCache := cache.New()
	for _, category := range testCategories {
		newID := uuid.New().String()
		categoryInstance := &v1.ResourceInstance{
			ResourceMeta: v1.ResourceMeta{
				Name:  newID,
				Title: category,
			},
			Spec: map[string]interface{}{},
		}
		categoryCache.SetWithSecondaryKey(newID, category, categoryInstance)
	}
	client.AddCache(categoryCache, teamCache)

	// Test oas2 object
	oas2Json, _ := os.Open("./testdata/petstore-swagger2.json") // OAS2
	defer oas2Json.Close()
	oas2Bytes, _ := ioutil.ReadAll(oas2Json)
	cloneServiceBody := serviceBody
	cloneServiceBody.SpecDefinition = oas2Bytes

	apiSvc, err := client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)
	assert.Equal(t, &cloneServiceBody.serviceContext.revisionName, &cloneServiceBody.serviceContext.instanceName)
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

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)

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
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to rollback apiservice
			RespCode: http.StatusOK,
		},
	})

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: "./testdata/empty-list.json", // this for call to rollback apiservice
			RespCode: http.StatusOK,
		},
	})

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)

	// this should fail
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
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
			RespCode: http.StatusRequestTimeout,
		},
		{
			RespCode: http.StatusOK, // this for call to rollback
		},
	})

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)
}

func TestGetAPIServiceByExternalAPIID(t *testing.T) {
	cloneServiceBody := serviceBody
	cloneServiceBody.PrimaryKey = "1234"
	client, httpClient := GetTestServiceClient()

	// bad
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // this for call to get the service
			RespCode: http.StatusBadRequest,
		},
	})
	svc, err := client.getAPIServiceByExternalAPIID(&cloneServiceBody)
	assert.NotNil(t, err)
	assert.Nil(t, svc)

	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // this for call to get the service
			RespCode: http.StatusNotFound,
		},
		{
			FileName: "./testdata/apiservice-list.json", // this for call to get the services
			RespCode: http.StatusOK,
		},
	})
	svc, err = client.getAPIServiceByExternalAPIID(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svc)
}

func TestUpdateService(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// tests for updating existing revision
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice-list.json", // for call to get the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingservicerevisions.json", // for call to get the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingserviceinstances.json", // for call to get instance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingconsumerinstances.json", // for call to check existance of the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
	})

	cloneServiceBody := serviceBody
	cloneServiceBody.APIUpdateSeverity = "MINOR"
	oas2Json, _ := os.Open("./testdata/petstore-swagger2.json") // OAS2
	defer oas2Json.Close()
	oas2Bytes, _ := ioutil.ReadAll(oas2Json)
	cloneServiceBody.SpecDefinition = oas2Bytes
	apiSvc, err := client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)
	assert.Equal(t, &cloneServiceBody.serviceContext.revisionName, &cloneServiceBody.serviceContext.instanceName)

	// this is a failure test
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json",
			RespCode: http.StatusRequestTimeout,
		},
	})

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)

	// tests for updating existing instance with same endpoint
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice-list.json", // for call to get the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingservicerevisions.json", // this for call to get the revision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingserviceinstances.json", // for call to get the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstancejson", // for call to update the serviceinstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/existingconsumerinstances.json", // for call to get the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
	})
	// Test oas2 object
	oas2Json, _ = os.Open("./testdata/petstore-swagger2.json") // OAS2
	defer oas2Json.Close()
	oas2Bytes, _ = ioutil.ReadAll(oas2Json)

	cloneServiceBody = serviceBody
	cloneServiceBody.SpecDefinition = oas2Bytes
	apiSvc, err = client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)
}
func TestRevision(t *testing.T) {
	client, _ := GetTestServiceClient()
	cloneServiceBody := serviceBody
	//Alt Revision
	cloneServiceBody.AltRevisionPrefix = "1.1.1"
	client.processRevision(&cloneServiceBody)
	assert.Contains(t, cloneServiceBody.serviceContext.revisionName, "1.1.1")
	// Normal Revision
	cloneServiceBody.AltRevisionPrefix = ""
	client.processRevision(&cloneServiceBody)
	assert.NotEqual(t, "", cloneServiceBody.serviceContext.revisionName)
}
func TestDeleteConsumerInstance(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	httpClient.ResponseCode = http.StatusRequestTimeout
	err := client.DeleteConsumerInstance("12345")
	assert.NotNil(t, err)
	assert.Contains(t, "[Error Code 1120] - error making a request to Amplify: status - 408", err.Error())

	httpClient.ResponseCode = http.StatusNoContent
	err = client.DeleteConsumerInstance("12345")
	assert.Nil(t, err)

	httpClient.ResponseCode = http.StatusOK
	err = client.DeleteConsumerInstance("12345")
	assert.Nil(t, err)
}

func TestGetConsumerInstancesByExternalAPIID(t *testing.T) {
	client, httpClient := GetTestServiceClient()

	// bad
	httpClient.SetResponse("./testdata/instancenotfound.json", http.StatusBadRequest)
	instances, err := client.GetConsumerInstancesByExternalAPIID("12345")
	assert.NotNil(t, err)
	assert.Nil(t, instances)

	// not found
	httpClient.SetResponse("./testdata/empty-list.json", http.StatusOK)
	instances, err = client.GetConsumerInstancesByExternalAPIID("12345")
	assert.NotNil(t, err)
	assert.Nil(t, instances)

	// good
	httpClient.SetResponse("./testdata/consumerinstancelist.json", http.StatusOK)
	instances, err = client.GetConsumerInstancesByExternalAPIID("12345")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(instances))
}

func TestDeleteServiceByAPIID(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	httpClient.ResponseCode = http.StatusRequestTimeout
	err := client.DeleteServiceByAPIID("12345")
	assert.NotNil(t, err)

	// empty list - not ok
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/empty-list.json", // for call to get the service
			RespCode: http.StatusOK,
		},
	})
	err = client.DeleteServiceByAPIID("12345")
	assert.NotNil(t, err)

	// list - ok
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice-list.json", // for call to get the service
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusNoContent, // delete OK
		},
	})
	err = client.DeleteServiceByAPIID("12345")
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

func TestUnstructuredConsumerInstanceData(t *testing.T) {
	// test the consumer instance handling unstrucutred data
	client, httpClient := GetTestServiceClient()
	serviceBody.AuthPolicy = "pass-through"

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

	// Test thrift object
	const filename = "multiplication.thrift"
	thriftFile, _ := os.Open("./testdata/" + filename) // OAS2
	thriftBytes, _ := ioutil.ReadAll(thriftFile)
	thriftFile.Close() // close now, no need to wait until the test is finished

	assetType := "Apache Thrift"
	contentType := "application/vnd.apache.thrift.compact"
	cloneServiceBody := serviceBody
	cloneServiceBody.ResourceType = Unstructured
	cloneServiceBody.SpecDefinition = thriftBytes
	cloneServiceBody.UnstructuredProps = &UnstructuredProperties{
		AssetType:   assetType,
		Filename:    filename,
		ContentType: contentType,
	}

	apiSvc, err := client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)

	// Get last request as consumerinstance
	var consInst v1alpha1.ConsumerInstance
	fmt.Println(string(httpClient.Requests[len(httpClient.Requests)-1].Body))
	json.Unmarshal(httpClient.Requests[len(httpClient.Requests)-1].Body, &consInst)

	// Only asset type set, label and asset type are equal
	assert.Equal(t, assetType, consInst.Spec.UnstructuredDataProperties.Type)
	assert.Equal(t, assetType, consInst.Spec.UnstructuredDataProperties.Label)
	assert.Equal(t, contentType, consInst.Spec.UnstructuredDataProperties.ContentType)
	assert.Equal(t, filename, consInst.Spec.UnstructuredDataProperties.FileName)

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

	label := "Apache Thrift"
	cloneServiceBody = serviceBody
	cloneServiceBody.ResourceType = Unstructured
	cloneServiceBody.SpecDefinition = thriftBytes
	cloneServiceBody.UnstructuredProps = &UnstructuredProperties{
		Label:       label,
		Filename:    filename,
		ContentType: contentType,
	}

	apiSvc, err = client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)

	// Get last request as consumerinstance
	consInst = v1alpha1.ConsumerInstance{}
	fmt.Println(string(httpClient.Requests[len(httpClient.Requests)-1].Body))
	json.Unmarshal(httpClient.Requests[len(httpClient.Requests)-1].Body, &consInst)

	// Only label type set, label and asset type are equal
	assert.Equal(t, label, consInst.Spec.UnstructuredDataProperties.Type)
	assert.Equal(t, label, consInst.Spec.UnstructuredDataProperties.Label)
	assert.Equal(t, contentType, consInst.Spec.UnstructuredDataProperties.ContentType)
	assert.Equal(t, filename, consInst.Spec.UnstructuredDataProperties.FileName)
}
