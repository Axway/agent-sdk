package apic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
	RestAPIID:        "12345",
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
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
	})

	// Setup category cache
	for _, category := range testCategories {
		newID := uuid.New().String()
		categoryInstance := &apiv1.ResourceInstance{
			ResourceMeta: apiv1.ResourceMeta{
				Name:  newID,
				Title: category,
			},
			Spec: map[string]interface{}{},
		}
		client.caches.GetCategoryCache().SetWithSecondaryKey(newID, category, categoryInstance)
	}

	// Test oas2 object
	oas2Json, _ := os.Open("./testdata/petstore-swagger2.json") // OAS2
	defer oas2Json.Close()
	oas2Bytes, _ := ioutil.ReadAll(oas2Json)
	cloneServiceBody := serviceBody
	cloneServiceBody.SpecDefinition = oas2Bytes

	apiSvc, err := client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)
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

func Test_getAPIServiceFromCache(t *testing.T) {
	cloneServiceBody := serviceBody
	cloneServiceBody.APIName = "fake-name"
	cloneServiceBody.RestAPIID = "123"
	client, _ := GetTestServiceClient()

	// Should return nil for the service and error when the api is not in the cache
	svc, err := client.getAPIServiceFromCache(&cloneServiceBody)
	assert.Nil(t, err)
	assert.Nil(t, svc)

	// Should return the service and no error
	apiSvc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Name:  "abc",
			Title: "abc",
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:   cloneServiceBody.RestAPIID,
					defs.AttrExternalAPIName: serviceBody.APIName,
				},
			},
		},
		Spec: management.ApiServiceSpec{},
	}
	// should return the resource when found by the external api id
	ri, _ := apiSvc.AsInstance()
	client.caches.AddAPIService(ri)
	svc, err = client.getAPIServiceFromCache(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svc)

	// should return the resource when found by the primary key
	cloneServiceBody.PrimaryKey = "555"
	err = util.SetAgentDetailsKey(apiSvc, defs.AttrExternalAPIPrimaryKey, cloneServiceBody.PrimaryKey)
	assert.Nil(t, err)

	ri, _ = apiSvc.AsInstance()
	client.caches.AddAPIService(ri)
	svc, err = client.getAPIServiceFromCache(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svc)

	// should return the resource when primary key is not found but external api id is
	cloneServiceBody.PrimaryKey = "4563"
	ri, _ = apiSvc.AsInstance()
	client.caches.AddAPIService(ri)
	svc, err = client.getAPIServiceFromCache(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, svc)

	// should return the nil for the error and resource when primary key and external api id are not found
	cloneServiceBody.RestAPIID = "4563"
	ri, _ = apiSvc.AsInstance()
	client.caches.AddAPIService(ri)
	svc, err = client.getAPIServiceFromCache(&cloneServiceBody)
	assert.Nil(t, err)
	assert.Nil(t, svc)
}

func TestUpdateService(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	// tests for updating existing revision
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apiservice.json", // for call to update the service subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the service subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance subresource
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

	fmt.Println("*********************")

	// tests for updating existing instance with same endpoint
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/apiservice.json", // for call to update the service subresource
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
			FileName: "./testdata/servicerevision.json", // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceinstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceinstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // for call to update the serviceinstance subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // for call to update the consumerInstance subresource
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

func Test_PublishServiceError(t *testing.T) {
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
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
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
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
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
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
	})
	// this is a failure test
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusRequestTimeout,
		},
	})

	apiSvc, err := client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)
}

func Test_processRevision(t *testing.T) {
	client, _ := GetTestServiceClient()
	cloneServiceBody := serviceBody
	// Alt Revision
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
	ri := &apiv1.ResourceInstance{ResourceMeta: apiv1.ResourceMeta{
		GroupVersionKind: apiv1.GroupVersionKind{},
		Name:             "name",
		Title:            "title",
	}}
	util.SetAgentDetailsKey(ri, defs.AttrExternalAPIID, "12345")
	client.caches.AddAPIService(ri)

	httpClient.SetResponse("./testdata/consumerinstancelist.json", http.StatusOK)
	instances, err = client.GetConsumerInstancesByExternalAPIID("12345")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(instances))
}

func TestDeleteServiceByAPIID(t *testing.T) {
	client, httpClient := GetTestServiceClient()
	httpClient.ResponseCode = http.StatusRequestTimeout
	err := client.DeleteServiceByName("12345")
	assert.NotNil(t, err)

	// list - ok
	httpClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusNoContent, // delete OK
		},
	})
	svc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Name:  "abc",
			Title: "abc",
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID: "12345",
				},
			},
		},
		Spec: management.ApiServiceSpec{},
	}
	ri, _ := svc.AsInstance()
	client.caches.AddAPIService(ri)
	err = client.DeleteServiceByName("12345")
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
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
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

	// Get second to last request as consumerinstance
	var consInst management.ConsumerInstance
	err = json.Unmarshal(httpClient.Requests[len(httpClient.Requests)-2].Body, &consInst)
	assert.Nil(t, err)

	// Only asset type set, label and asset type are equal
	assert.Equal(t, assetType, consInst.Spec.UnstructuredDataProperties.Type)
	assert.Equal(t, assetType, consInst.Spec.UnstructuredDataProperties.Label)
	assert.Equal(t, contentType, consInst.Spec.UnstructuredDataProperties.ContentType)
	assert.Equal(t, filename, consInst.Spec.UnstructuredDataProperties.FileName)

	fmt.Println("*************************")
	// this should be a full go right path
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: "./testdata/apiservice.json", // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/servicerevision.json", // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/serviceinstance.json", // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/consumerinstance.json", // this for call to create the consumerInstance
			RespCode: http.StatusOK,
		},
		{
			FileName: "./testdata/agent-details-sr.json", // this for call to create the service
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
	consInst = management.ConsumerInstance{}
	fmt.Println(string(httpClient.Requests[len(httpClient.Requests)-2].Body))
	err = json.Unmarshal(httpClient.Requests[len(httpClient.Requests)-2].Body, &consInst)
	assert.Nil(t, err)

	// Only label type set, label and asset type are equal
	assert.Equal(t, label, consInst.Spec.UnstructuredDataProperties.Type)
	assert.Equal(t, label, consInst.Spec.UnstructuredDataProperties.Label)
	assert.Equal(t, contentType, consInst.Spec.UnstructuredDataProperties.ContentType)
	assert.Equal(t, filename, consInst.Spec.UnstructuredDataProperties.FileName)
}

func TestServiceClient_buildAPIService(t *testing.T) {
	body := &ServiceBody{
		Description:      "description",
		ImageContentType: "content-type",
		Image:            "image-data",
		NameToPush:       "nametopush",
		APIName:          "apiname",
		RestAPIID:        "restapiid",
		PrimaryKey:       "primarykey",
		Stage:            "staging",
		Version:          "v1",
		Tags: map[string]interface{}{
			"tag1": "value1",
			"tag2": "value2",
		},
		CreatedBy:          "createdby",
		ServiceAttributes:  map[string]string{"service_attribute": "value"},
		RevisionAttributes: map[string]string{"revision_attribute": "value"},
		InstanceAttributes: map[string]string{"instance_attribute": "value"},
		ServiceAgentDetails: map[string]interface{}{
			"subresource_svc_key": "value",
		},
		InstanceAgentDetails: map[string]interface{}{
			"subresource_instance_key": "value",
		},
		RevisionAgentDetails: map[string]interface{}{
			"subresource_revision_key": "value",
		},
	}

	tags := []string{"tag1_value1", "tag2_value2"}

	client, _ := GetTestServiceClient()
	svc := client.buildAPIService(body)

	assert.Equal(t, management.APIServiceGVK(), svc.GroupVersionKind)
	assert.Empty(t, svc.Name)
	assert.Equal(t, body.NameToPush, svc.Title)
	assert.Contains(t, svc.Tags, tags[0])
	assert.Contains(t, svc.Tags, tags[1])
	assert.Equal(t, body.ServiceAttributes, svc.Attributes)
	assert.Equal(t, body.ImageContentType, svc.Spec.Icon.ContentType)
	assert.Equal(t, body.Image, svc.Spec.Icon.Data)
	assert.Equal(t, body.Description, svc.Spec.Description)
	assert.Equal(t, body.ServiceAttributes, svc.Attributes)

	assert.Contains(t, svc.Attributes, "service_attribute")
	assert.NotContains(t, svc.Attributes, "revision_attribute")
	assert.NotContains(t, svc.Attributes, "instance_attribute")
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, svc.Attributes, defs.AttrCreatedBy)

	sub := util.GetAgentDetails(svc)
	// stage is not set for api services
	assert.Empty(t, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.NotContains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
}

func TestServiceClient_updateAPIService(t *testing.T) {
	body := &ServiceBody{
		Description:      "description",
		ImageContentType: "content-type",
		Image:            "image-data",
		NameToPush:       "nametopush",
		APIName:          "apiname",
		RestAPIID:        "restapiid",
		PrimaryKey:       "primarykey",
		Stage:            "staging",
		Version:          "v1",
		Tags: map[string]interface{}{
			"tag1": "value1",
			"tag2": "value2",
		},
		CreatedBy:          "createdby",
		ServiceAttributes:  map[string]string{"service_attribute": "value"},
		RevisionAttributes: map[string]string{"revision_attribute": "value"},
		InstanceAttributes: map[string]string{"instance_attribute": "value"},
		ServiceAgentDetails: map[string]interface{}{
			"subresource_svc_key": "value",
		},
		InstanceAgentDetails: map[string]interface{}{
			"subresource_instance_key": "value",
		},
		RevisionAgentDetails: map[string]interface{}{
			"subresource_revision_key": "value",
		},
	}

	svc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Metadata: apiv1.Metadata{
				ResourceVersion: "123",
			},
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					"old_subresource_svc_key": "old_val",
				},
			},
		},
	}

	tags := []string{"tag1_value1", "tag2_value2"}

	client, _ := GetTestServiceClient()
	client.updateAPIService(body, svc)

	assert.Equal(t, management.APIServiceGVK(), svc.GroupVersionKind)
	assert.Empty(t, svc.Metadata.ResourceVersion)
	assert.Empty(t, svc.Name)

	assert.Equal(t, body.NameToPush, svc.Title)
	assert.Contains(t, svc.Tags, tags[0])
	assert.Contains(t, svc.Tags, tags[1])
	assert.Equal(t, body.ServiceAttributes, svc.Attributes)
	assert.NotContains(t, svc.Attributes, "instance_attribute")
	assert.NotContains(t, svc.Attributes, "revision_attribute")

	assert.Equal(t, body.ImageContentType, svc.Spec.Icon.ContentType)
	assert.Equal(t, body.Image, svc.Spec.Icon.Data)
	assert.Equal(t, body.Description, svc.Spec.Description)

	assert.Contains(t, svc.Attributes, "service_attribute")
	assert.NotContains(t, svc.Attributes, "revision_attribute")
	assert.NotContains(t, svc.Attributes, "instance_attribute")
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, svc.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, svc.Attributes, defs.AttrCreatedBy)

	sub := util.GetAgentDetails(svc)
	assert.Empty(t, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "old_subresource_svc_key")
	assert.NotContains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
}

func Test_buildAPIServiceNilAttributes(t *testing.T) {
	client, _ := GetTestServiceClient()
	body := &ServiceBody{}

	svc := client.buildAPIService(body)
	assert.NotNil(t, svc.Attributes)

	svc.Attributes = nil
	client.updateAPIService(body, svc)
	assert.NotNil(t, svc.Attributes)
}
