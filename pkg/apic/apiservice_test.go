package apic

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Axway/agent-sdk/pkg/util"

	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"

	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

const (
	testDocs             = `"docs"`
	testImageContentType = "image/jpeg"
	testAPIServiceFile   = "./testdata/apiservice.json"
	testAgentDetailsFile = "./testdata/agent-details-sr.json"
	testRevisionFile     = "./testdata/servicerevision.json"
	testInstanceFile     = "./testdata/serviceinstance.json"
	testRevisionListFile = "./testdata/servicerevisionlist.json"
	testEmptyListFile    = "./testdata/empty-list.json"
	testPetstoreSpec     = "./testdata/petstore-swagger2.json"
	testRevisionNameAlt  = "daleapi-1"
)

var serviceBody = ServiceBody{
	APIName:          "daleapi",
	Documentation:    []byte(testDocs),
	Image:            "abcde",
	ImageContentType: testImageContentType,
	ResourceType:     Oas2,
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
			FileName: testAPIServiceFile, // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: testAgentDetailsFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testAgentDetailsFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: testAgentDetailsFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			FileName: testAgentDetailsFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testAgentDetailsFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
	})

	// Test oas2 object
	oas2Bytes, err := os.ReadFile(testPetstoreSpec)
	assert.Nil(t, err)
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
			FileName: testAPIServiceFile, // this for call to create the service
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
			FileName: testAPIServiceFile, // this for call to create the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // this for call to create the serviceRevision
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: testEmptyListFile, // this for call to rollback apiservice
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
			FileName: testAPIServiceFile, // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: testRevisionFile, // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: testInstanceFile, // this for call to create the serviceInstance
			RespCode: http.StatusRequestTimeout,
		},
		{
			FileName: testEmptyListFile, // this for call to rollback apiservice
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
			FileName: testAPIServiceFile, // this for call to create the service
			RespCode: http.StatusCreated,
		},
		{
			FileName: testRevisionFile, // this for call to create the serviceRevision
			RespCode: http.StatusCreated,
		},
		{
			FileName: testInstanceFile, // this for call to create the serviceInstance
			RespCode: http.StatusCreated,
		},
		{
			RespCode: http.StatusOK, // this for call to rollback
		},
	})

	apiSvc, err = client.PublishService(&serviceBody)
	assert.NotNil(t, err)
	assert.Nil(t, apiSvc)
}

func TestPublishServiceRevisionOnly(t *testing.T) {
	oas2Bytes, err := os.ReadFile(testPetstoreSpec)
	require.NoError(t, err)

	tests := map[string]struct {
		revisionOnly bool
		responses    []api.MockResponse
	}{
		"full publish creates service, revision, and instance": {
			revisionOnly: false,
			responses: []api.MockResponse{
				{FileName: testAPIServiceFile, RespCode: http.StatusCreated},      // POST service
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // service subresource
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // spec hashes
				{FileName: testRevisionFile, RespCode: http.StatusCreated}, // POST revision
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // revision subresource
				{FileName: testInstanceFile, RespCode: http.StatusCreated}, // POST instance
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // instance subresource
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // spec hashes update
			},
		},
		"revision-only publish skips instance creation": {
			revisionOnly: true,
			responses: []api.MockResponse{
				{FileName: testAPIServiceFile, RespCode: http.StatusCreated},      // POST service
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // service subresource
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // spec hashes
				{FileName: testRevisionFile, RespCode: http.StatusCreated}, // POST revision
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // revision subresource
				{FileName: testAgentDetailsFile, RespCode: http.StatusOK},     // spec hashes update
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, httpClient := GetTestServiceClient()
			httpClient.SetResponses(tc.responses)

			body := serviceBody
			body.SpecDefinition = oas2Bytes
			body.revisionOnly = tc.revisionOnly

			apiSvc, err := client.PublishService(&body)
			assert.Nil(t, err)
			assert.NotNil(t, apiSvc)
		})
	}
}

func TestGetAPIServiceFromCache(t *testing.T) {
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
			FileName: testAPIServiceFile, // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testAPIServiceFile, // for call to update the service subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to update the service subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testAPIServiceFile, // for call to update the service subresource
			RespCode: http.StatusOK,
		},
	})

	cloneServiceBody := serviceBody
	cloneServiceBody.APIUpdateSeverity = "MINOR"
	oas2Bytes, err := os.ReadFile(testPetstoreSpec)
	assert.Nil(t, err)
	cloneServiceBody.SpecDefinition = oas2Bytes
	apiSvc, err := client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)

	fmt.Println("*********************")

	// tests for updating existing instance with same endpoint
	httpClient.SetResponses([]api.MockResponse{
		{
			FileName: testAPIServiceFile, // for call to update the service
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionListFile, // for call to update the service subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to get the serviceRevision count
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to get the serviceRevision count based on name
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to update the serviceRevision
			RespCode: http.StatusOK,
		},
		{
			FileName: testRevisionFile, // for call to update the serviceRevision subresource
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // for call to update the serviceinstance
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // for call to update the serviceinstance
			RespCode: http.StatusOK,
		},
		{
			FileName: testInstanceFile, // for call to update the serviceinstance subresource
			RespCode: http.StatusOK,
		},
	})
	// Test oas2 object
	oas2Bytes, err = os.ReadFile(testPetstoreSpec)
	assert.Nil(t, err)

	cloneServiceBody = serviceBody
	cloneServiceBody.SpecDefinition = oas2Bytes
	apiSvc, err = client.PublishService(&cloneServiceBody)
	assert.Nil(t, err)
	assert.NotNil(t, apiSvc)
}

func TestPublishServiceError(t *testing.T) {
	client, httpClient := GetTestServiceClient()

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

func TestProcessRevision(t *testing.T) {
	tests := map[string]struct {
		httpResponses   []api.MockResponse
		serviceBody     ServiceBody
		expectedRevName string
	}{
		"publish new revision": {
			httpResponses: []api.MockResponse{
				{
					FileName: testRevisionFile, // for call to update the serviceRevision
					RespCode: http.StatusOK,
				},
				{
					FileName: testRevisionFile, // for call to update the serviceRevision x-agent-details
					RespCode: http.StatusOK,
				},
				{
					FileName: testRevisionFile, // for call to update the serviceRevision
					RespCode: http.StatusOK,
				},
				{
					FileName: testRevisionFile, // for call to update the serviceRevision x-agent-details
					RespCode: http.StatusOK,
				},
			},
			serviceBody: ServiceBody{
				APIName:          "daleapi",
				Documentation:    []byte(testDocs),
				Image:            "abcde",
				ImageContentType: testImageContentType,
				ResourceType:     Oas2,
				RestAPIID:        "12345",
			},
			expectedRevName: "daleapi",
		},
		"skip publish when no changes": {
			httpResponses: []api.MockResponse{
				{
					RespData: `[{"name": "daleapi","tags": ["tag1","tag2"]}]`,
					RespCode: http.StatusOK,
				},
			},
			serviceBody: ServiceBody{
				APIName:          "daleapi",
				Documentation:    []byte(testDocs),
				Image:            "abcde",
				ImageContentType: testImageContentType,
				ResourceType:     Oas2,
				RestAPIID:        "12345",
				specHash:         "abc123",
				specHashes: map[string]interface{}{
					"abc123": "daleapi",
				},
				serviceContext: serviceContext{
					serviceAction: updateAPI,
				},
			},
			expectedRevName: "daleapi",
		},
		"skip publish when previous revision found": {
			httpResponses: []api.MockResponse{
				{
					RespData: `[{"name": "daleapi","tags": ["tag1","tag2"]},{"name": "` + testRevisionNameAlt + `","tags": ["tag1","tag2"]}]`,
					RespCode: http.StatusOK,
				},
			},
			serviceBody: ServiceBody{
				APIName:          "daleapi",
				Documentation:    []byte(testDocs),
				Image:            "abcde",
				ImageContentType: testImageContentType,
				ResourceType:     Oas2,
				RestAPIID:        "12345",
				specHash:         "abc123",
				specHashes: map[string]interface{}{
					"abc123": testRevisionNameAlt,
				},
				serviceContext: serviceContext{
					serviceAction: updateAPI,
				},
			},
			expectedRevName: testRevisionNameAlt,
		},
		"find revision match using original spec hash": {
			httpResponses: []api.MockResponse{
				{
					RespData: `[{"name": "daleapi","tags": ["tag1","tag2"]},{"name": "` + testRevisionNameAlt + `","tags": ["tag1","tag2"]}]`,
					RespCode: http.StatusOK,
				},
			},
			serviceBody: ServiceBody{
				APIName:          "daleapi",
				Documentation:    []byte(testDocs),
				Image:            "abcde",
				ImageContentType: testImageContentType,
				ResourceType:     Oas2,
				RestAPIID:        "12345",
				specHash:         "abc1234",
				originalSpecHash: "abc123",
				specHashes: map[string]interface{}{
					"abc123": testRevisionNameAlt,
				},
				serviceContext: serviceContext{
					serviceAction: updateAPI,
				},
			},
			expectedRevName: testRevisionNameAlt,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, httpClient := GetTestServiceClient()
			httpClient.SetResponses(tc.httpResponses)

			client.processRevision(&tc.serviceBody)
			assert.NotEmpty(t, tc.serviceBody.serviceContext.revisionName)
			assert.Equal(t, tc.expectedRevName, tc.serviceBody.serviceContext.revisionName)
		})
	}
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

func TestBuildAPIService(t *testing.T) {
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

func TestUpdateAPIService(t *testing.T) {
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

func TestBuildAPIServiceNilAttributes(t *testing.T) {
	client, _ := GetTestServiceClient()
	body := &ServiceBody{}

	svc := client.buildAPIService(body)
	assert.NotNil(t, svc.Attributes)

	svc.Attributes = nil
	client.updateAPIService(body, svc)
	assert.NotNil(t, svc.Attributes)
}

func createAPIService(name, id string, refSvc string, dpType string, isDesign bool) *management.APIService {
	apiSvc := &management.APIService{
		ResourceMeta: apiv1.ResourceMeta{
			Name:  name,
			Title: name,
			SubResources: map[string]interface{}{
				defs.XAgentDetails: map[string]interface{}{
					defs.AttrExternalAPIID:   id,
					defs.AttrExternalAPIName: name,
				},
			},
		},
		Spec: management.ApiServiceSpec{},
	}
	if refSvc != "" || dpType != "" {
		apiSvc.Source = &management.ApiServiceSource{
			References: &management.ApiServiceSourceReferences{
				ApiService: refSvc,
			},
		}
		apiSvc.Source.DataplaneType = &management.ApiServiceSourceDataplaneType{}
		if isDesign {
			apiSvc.Source.DataplaneType.Design = dpType
		} else {
			apiSvc.Source.DataplaneType.Managed = dpType
		}
	}
	return apiSvc
}

func TestServiceSourceUpdates(t *testing.T) {
	// case 1 - new service, source managed dataplane, sub resource updated
	// case 2 - new service, source design dataplane, sub resource updated
	// case 3 - new service, source unmanaged dataplane with reference, sub resource updated
	// case 4 - existing service, no source, source updated
	// case 5 - existing service, existing source, different dataplane type, source updated
	// case 6 - existing service, existing source, different reference, source updated
	// case 7 - existing service, existing source, same dataplane type and same reference, no source updated
	testCases := []struct {
		name               string
		svcName            string
		managedDataplane   DataplaneType
		designDataplane    DataplaneType
		existingSvc        *management.APIService
		referenceService   string
		apiserverResponses []api.MockResponse
	}{
		{
			name:             "new service for managed dataplane",
			svcName:          "newSvcManaged",
			managedDataplane: AWS,
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to create the service
					RespCode: http.StatusCreated,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:            "new service for design dataplane",
			svcName:         "newSvcDesign",
			designDataplane: GitLab,
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to create the service
					RespCode: http.StatusCreated,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "new service for unmanaged dataplane with referenced service",
			svcName:          "newSvcUnmanaged",
			managedDataplane: Unclassified,
			referenceService: "refSvc",
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to create the service
					RespCode: http.StatusCreated,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing service with no source",
			svcName:          "existingSvcNoSource",
			managedDataplane: AWS,
			existingSvc:      createAPIService("existingSvcNoSource", "existingSvcNoSource", "", "", false),
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to update the service
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing service with different dataplane type",
			svcName:          "existingSvcDiffDpType",
			managedDataplane: AWS,
			existingSvc:      createAPIService("existingSvcDiffDpType", "existingSvcDiffDpType", "", Unidentified.String(), false),
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to update the service
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing service with different referenced service",
			svcName:          "existingSvcDiffRefSvc",
			managedDataplane: AWS,
			existingSvc:      createAPIService("existingSvcDiffRefSvc", "existingSvcDiffRefSvc", "existingRefSvc", AWS.String(), false),
			referenceService: "newRefSvc",
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to update the service
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing service with same source",
			svcName:          "existingSvcSameSource",
			managedDataplane: AWS,
			existingSvc:      createAPIService("existingSvcSameSource", "existingSvcSameSource", "refSvc", AWS.String(), false),
			referenceService: "refSvc",
			apiserverResponses: []api.MockResponse{
				{
					FileName: testAPIServiceFile, // call to update the service
					RespCode: http.StatusOK,
				},
				{
					FileName: testAPIServiceFile, // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				// no source subresource update
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client, httpClient := GetTestServiceClient()
			if test.existingSvc != nil {
				ri, _ := test.existingSvc.AsInstance()
				client.caches.AddAPIService(ri)
			}

			body := &ServiceBody{
				RestAPIID: test.svcName,
			}
			if test.managedDataplane != "" {
				body.dataplaneType = test.managedDataplane
			}
			if test.designDataplane != "" {
				body.dataplaneType = test.designDataplane
				body.isDesignDataplane = true
			}
			body.referencedServiceName = test.referenceService
			httpClient.SetResponses(test.apiserverResponses)

			svc, err := client.processService(body)
			assert.Nil(t, err)
			assert.NotNil(t, svc)
			assert.Equal(t, len(test.apiserverResponses), httpClient.RespCount)
		})
	}
}
