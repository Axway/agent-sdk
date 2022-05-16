package apic

// TODO - this file should be able to be removed once Unified Catalog support has been removed
import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestServiceClient_buildConsumerInstance(t *testing.T) {
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

	inst := client.buildConsumerInstance(body, "name", "doc")

	assert.Equal(t, mv1a.ConsumerInstanceGVK(), inst.GroupVersionKind)
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

	sub := util.GetAgentDetails(inst)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "subresource_revision_key")
	assert.NotContains(t, sub, "instance_attribute")
}

func TestServiceClient_updateConsumerInstance(t *testing.T) {
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

	inst := &mv1a.ConsumerInstance{
		ResourceMeta: v1.ResourceMeta{
			Title: "oldname",
			Attributes: map[string]string{
				"old_attribute": "value",
			},
			Name: "name",
			Tags: []string{"old_tag"},
			Metadata: v1.Metadata{
				ResourceVersion: "123",
			},
		},
	}

	client.updateConsumerInstance(body, inst, "doc")

	assert.Equal(t, mv1a.ConsumerInstanceGVK(), inst.GroupVersionKind)
	assert.Equal(t, "name", inst.Name)
	assert.Equal(t, body.NameToPush, inst.Title)

	assert.Contains(t, inst.Tags, tags[0])
	assert.Contains(t, inst.Tags, tags[1])

	assert.Contains(t, inst.Attributes, "instance_attribute")
	assert.NotContains(t, inst.Attributes, "service_attribute")
	assert.NotContains(t, inst.Attributes, "revision_attribute")
	assert.NotContains(t, inst.Attributes, "old_attribute")
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
	assert.NotContains(t, sub, "instance_attribute")
}

func Test_buildConsumerInstanceNilAttributes(t *testing.T) {
	client, _ := GetTestServiceClient()
	body := &ServiceBody{}

	ci := client.buildConsumerInstance(body, "name", "doc")
	assert.NotNil(t, ci.Attributes)

	ci.Attributes = nil
	client.updateConsumerInstance(body, ci, "doc")
	assert.NotNil(t, ci.Attributes)
}
