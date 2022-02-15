package apic

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
		AgentMode:          0,
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
	ep := []mv1a.ApiServiceInstanceSpecEndpoint{
		{
			Host:     "abc.com",
			Port:     80,
			Protocol: "http",
			Routing: mv1a.ApiServiceInstanceSpecRouting{
				BasePath: "/base",
			},
		},
		{
			Host:     "123.com",
			Port:     443,
			Protocol: "https",
			Routing: mv1a.ApiServiceInstanceSpecRouting{
				BasePath: "/path",
			},
		},
	}
	inst := client.buildAPIServiceInstance(body, "name", ep)

	assert.Equal(t, mv1a.APIServiceInstanceGVK(), inst.GroupVersionKind)
	assert.Equal(t, "name", inst.Name)
	assert.Equal(t, body.NameToPush, inst.Title)

	assert.Contains(t, inst.Tags, tags[0])
	assert.Contains(t, inst.Tags, tags[1])

	assert.Contains(t, inst.Attributes, "instance_attribute")
	assert.Contains(t, inst.Attributes, "service_attribute")
	assert.NotContains(t, inst.Attributes, "revision_attribute")

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
		AgentMode:          0,
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
	ep := []mv1a.ApiServiceInstanceSpecEndpoint{
		{
			Host:     "abc.com",
			Port:     80,
			Protocol: "http",
			Routing: mv1a.ApiServiceInstanceSpecRouting{
				BasePath: "/base",
			},
		},
		{
			Host:     "123.com",
			Port:     443,
			Protocol: "https",
			Routing: mv1a.ApiServiceInstanceSpecRouting{
				BasePath: "/path",
			},
		},
	}
	inst := &mv1a.APIServiceInstance{
		ResourceMeta: v1.ResourceMeta{
			Title: "oldname",
			Attributes: map[string]string{
				"old": "value",
			},
			Name:     "name",
			Tags:     []string{"old_tag"},
			Metadata: v1.Metadata{ResourceVersion: ""},
		},
	}
	inst = client.updateAPIServiceInstance(body, inst, ep)

	assert.Equal(t, mv1a.APIServiceInstanceGVK(), inst.GroupVersionKind)
	assert.Empty(t, inst.Metadata.ResourceVersion)
	assert.Equal(t, "name", inst.Name)
	assert.Equal(t, body.NameToPush, inst.Title)

	assert.Contains(t, inst.Tags, tags[0])
	assert.Contains(t, inst.Tags, tags[1])
	assert.NotContains(t, inst.Tags, "old_tag")

	assert.Equal(t, inst.Spec.Endpoint, ep)

	assert.Contains(t, inst.Attributes, "instance_attribute")
	assert.Contains(t, inst.Attributes, "service_attribute")
	assert.NotContains(t, inst.Attributes, "revision_attribute")

	sub := util.GetAgentDetails(inst)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
}
