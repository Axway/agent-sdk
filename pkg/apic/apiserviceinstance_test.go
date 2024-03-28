package apic

import (
	"net/http"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestServiceClient_buildAPIServiceInstance(t *testing.T) {
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
	ep := []management.ApiServiceInstanceSpecEndpoint{
		{
			Host:     "abc.com",
			Port:     80,
			Protocol: "http",
			Routing: management.ApiServiceInstanceSpecRouting{
				BasePath: "/base",
			},
		},
		{
			Host:     "123.com",
			Port:     443,
			Protocol: "https",
			Routing: management.ApiServiceInstanceSpecRouting{
				BasePath: "/path",
			},
		},
	}
	inst := client.buildAPIServiceInstance(body, "name", ep)

	assert.Equal(t, management.APIServiceInstanceGVK(), inst.GroupVersionKind)
	assert.Equal(t, "name", inst.Name)
	assert.Equal(t, body.NameToPush, inst.Title)

	assert.Contains(t, inst.Tags, tags[0])
	assert.Contains(t, inst.Tags, tags[1])

	assert.Contains(t, inst.Attributes, "instance_attribute")
	assert.NotContains(t, inst.Attributes, "service_attribute")
	assert.NotContains(t, inst.Attributes, "revision_attribute")
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, inst.Attributes, defs.AttrCreatedBy)

	assert.Equal(t, inst.Spec.Endpoint, ep)

	sub := util.GetAgentDetails(inst)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
	assert.NotContains(t, sub, "revision_attribute")
}

func TestServiceClient_updateAPIServiceInstance(t *testing.T) {
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
	ep := []management.ApiServiceInstanceSpecEndpoint{
		{
			Host:     "abc.com",
			Port:     80,
			Protocol: "http",
			Routing: management.ApiServiceInstanceSpecRouting{
				BasePath: "/base",
			},
		},
		{
			Host:     "123.com",
			Port:     443,
			Protocol: "https",
			Routing: management.ApiServiceInstanceSpecRouting{
				BasePath: "/path",
			},
		},
	}
	inst := &management.APIServiceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			Title: "oldname",
			Attributes: map[string]string{
				"old_attr": "value",
			},
			Name:     "name",
			Tags:     []string{"old_tag"},
			Metadata: apiv1.Metadata{ResourceVersion: ""},
		},
	}
	inst = client.updateAPIServiceInstance(body, inst, ep)

	assert.Equal(t, management.APIServiceInstanceGVK(), inst.GroupVersionKind)
	assert.Empty(t, inst.Metadata.ResourceVersion)
	assert.Equal(t, "name", inst.Name)
	assert.Equal(t, body.NameToPush, inst.Title)

	assert.Contains(t, inst.Tags, tags[0])
	assert.Contains(t, inst.Tags, tags[1])
	assert.NotContains(t, inst.Tags, "old_tag")

	assert.Equal(t, inst.Spec.Endpoint, ep)

	assert.Contains(t, inst.Attributes, "instance_attribute")
	assert.NotContains(t, inst.Attributes, "service_attribute")
	assert.NotContains(t, inst.Attributes, "revision_attribute")
	assert.NotContains(t, inst.Attributes, "old_attr")
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, inst.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, inst.Attributes, defs.AttrCreatedBy)

	sub := util.GetAgentDetails(inst)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
	assert.NotContains(t, sub, "revision_attribute")
}

func Test_buildAPIServiceInstanceNilAttributes(t *testing.T) {
	client, _ := GetTestServiceClient()
	body := &ServiceBody{}
	ep := []management.ApiServiceInstanceSpecEndpoint{}
	inst := client.buildAPIServiceInstance(body, "name", ep)
	assert.NotNil(t, inst.Attributes)

	inst.Attributes = nil

	inst = client.updateAPIServiceInstance(body, inst, ep)
	assert.NotNil(t, inst.Attributes)
}

func createAPIServiceInstance(name, id string, refInstance string, dpType string, isDesign bool) *management.APIServiceInstance {
	instance := &management.APIServiceInstance{
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
		Spec: management.ApiServiceInstanceSpec{},
	}
	if refInstance != "" || dpType != "" {
		instance.Source = &management.ApiServiceInstanceSource{
			References: &management.ApiServiceInstanceSourceReferences{
				ApiServiceInstance: refInstance,
			},
		}
		instance.Source.DataplaneType = &management.ApiServiceInstanceSourceDataplaneType{}
		if isDesign {
			instance.Source.DataplaneType.Design = dpType
		} else {
			instance.Source.DataplaneType.Managed = dpType
		}
	}
	return instance
}

func TestInstanceSourceUpdates(t *testing.T) {
	// case 1 - new instance, source managed dataplane, sub resource updated
	// case 2 - new instance, source design dataplane, sub resource updated
	// case 3 - new instance, source unmanaged dataplane with reference, sub resource updated
	// case 4 - existing instance, no source, source updated
	// case 5 - existing instance, existing source, different dataplane type, source updated
	// case 6 - existing instance, existing source, different reference, source updated
	// case 7 - existing instance, existing source, same dataplane type and same reference, no source updated
	testCases := []struct {
		name               string
		instanceName       string
		managedDataplane   DataplaneType
		designDataplane    DataplaneType
		existingInstance   *management.APIServiceInstance
		referenceInstance  string
		apiserverResponses []api.MockResponse
	}{
		{
			name:             "new instance for managed dataplane",
			instanceName:     "newInstManaged",
			managedDataplane: AWS,
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/serviceinstance.json", // call to create the instance
					RespCode: http.StatusCreated,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:            "new instance for design dataplane",
			instanceName:    "newInstDesign",
			designDataplane: GitLab,
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/serviceinstance.json", // call to create the instance
					RespCode: http.StatusCreated,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:              "new instance for unmanaged dataplane with referenced instance",
			instanceName:      "newInstUnmanaged",
			managedDataplane:  Unclassified,
			referenceInstance: "refInst",
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/serviceinstance.json", // call to create the instance
					RespCode: http.StatusCreated,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing instance with no source",
			instanceName:     "daleapi",
			managedDataplane: AWS,
			existingInstance: createAPIServiceInstance("daleapi", "2f5f92f0-f5e4-44fb-bc84-599c27b3497a", "", "", false),
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/serviceinstance.json", // call to get the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:             "existing instance with different dataplane type",
			instanceName:     "existingInstance",
			managedDataplane: AWS,
			existingInstance: createAPIServiceInstance("existingInstance", "existingInstance", "", Unclassified.String(), false),
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/existingserviceinstances.json", // call to get the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:              "existing instance with different referenced instance",
			instanceName:      "existingInstance",
			managedDataplane:  Unclassified,
			existingInstance:  createAPIServiceInstance("existingInstance", "existingInstance", "refInstance", Unclassified.String(), false),
			referenceInstance: "newRefInstance",
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/existingserviceinstances.json", // call to get the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update source subresource
					RespCode: http.StatusOK,
				},
			},
		},
		{
			name:              "existing instance with same source",
			instanceName:      "existingInstance",
			managedDataplane:  Unclassified,
			existingInstance:  createAPIServiceInstance("existingInstance", "existingInstance", "refInstance", Unclassified.String(), false),
			referenceInstance: "refInstance",
			apiserverResponses: []api.MockResponse{
				{
					FileName: "./testdata/existingserviceinstances.json", // call to get the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update the instance
					RespCode: http.StatusOK,
				},
				{
					FileName: "./testdata/serviceinstance.json", // call to update x-agent-details subresource
					RespCode: http.StatusOK,
				},
				// no source subresource update
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client, httpClient := GetTestServiceClient()
			body := &ServiceBody{
				RestAPIID: test.instanceName,
			}

			if test.existingInstance != nil {
				ri, _ := test.existingInstance.AsInstance()
				client.caches.AddAPIServiceInstance(ri)
				body.serviceContext.serviceAction = updateAPI
				body.serviceContext.revisionCount = 1
			}

			if test.managedDataplane != "" {
				body.dataplaneType = test.managedDataplane
			}
			if test.designDataplane != "" {
				body.dataplaneType = test.designDataplane
				body.isDesignDataplane = true
			}
			body.referencedInstanceName = test.referenceInstance
			httpClient.SetResponses(test.apiserverResponses)

			err := client.processInstance(body)
			assert.Nil(t, err)
			assert.Equal(t, len(test.apiserverResponses), httpClient.RespCount)
		})
	}
}
