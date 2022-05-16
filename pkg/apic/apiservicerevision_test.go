package apic

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1a "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAPIServiceRevisionTitle(t *testing.T) {
	testCases := []struct {
		name     string
		format   string
		apiName  string
		stage    string
		label    string
		count    int
		expected string
	}{
		{
			name:     "No Stage",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Stage - default label",
			apiName:  "API-Name",
			stage:    "PROD",
			count:    5,
			expected: "API-Name \\(Stage\\: PROD\\) - \\d{4}/\\d{2}/\\d{2} - r 5",
		},
		{
			name:     "Stage - new label",
			apiName:  "API-Name",
			stage:    "PROD",
			label:    "Portal",
			count:    3,
			expected: "API-Name \\(Portal\\: PROD\\) - \\d{4}/\\d{2}/\\d{2} - r 3",
		},
		{
			name:     "Bad Date - default",
			format:   "{{.APIServiceName}} - {{.Date:YYY/MM/DD}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "New Date Format",
			format:   "{{.APIServiceName}} - {{.Date:YYYY-MM-DD}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}-\\d{2}-\\d{2}",
		},
		{
			name:     "Deprecated Date",
			format:   "{{.APIServiceName}} - {{date}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Bar Variable - default",
			format:   "{{.APIServiceName1}} - {{date}} - r {{.Revision}}",
			apiName:  "API-Name",
			count:    1,
			expected: "API-Name - \\d{4}/\\d{2}/\\d{2} - r 1",
		},
		{
			name:     "Stage - new format",
			format:   "{{.Stage}} - {{.APIServiceName}} - {{.Date:MM/DD/YYYY}} - r {{.Revision}}",
			apiName:  "API-Name",
			stage:    "MyStage",
			label:    "Test",
			count:    6,
			expected: "MyStage - API-Name - \\d{2}/\\d{2}/\\d{4} - r 6",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// create the service client
			c := ServiceClient{
				cfg: config.NewCentralConfig(config.DiscoveryAgent),
			}
			c.cfg.(*config.CentralConfiguration).APIServiceRevisionPattern = test.format

			s := &ServiceBody{
				APIName: test.apiName,
				Stage:   test.stage,
				serviceContext: serviceContext{
					revisionCount: test.count,
				},
				StageDescriptor: "Stage", // default
			}
			if test.label != "" {
				s.StageDescriptor = test.label
			}

			title := c.updateAPIServiceRevisionTitle(s)
			assert.Regexp(t, test.expected, title)
		})
	}
}

func Test_buildAPIServiceRevision(t *testing.T) {
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
		serviceContext: serviceContext{serviceName: "service-context-name"},
	}

	tags := []string{"tag1_value1", "tag2_value2"}

	client, _ := GetTestServiceClient()
	revision := client.buildAPIServiceRevision(body, "name")

	assert.Equal(t, mv1a.APIServiceRevisionGVK(), revision.GroupVersionKind)
	assert.Equal(t, "name", revision.Name)
	assert.Contains(t, revision.Title, body.APIName)
	assert.Contains(t, revision.Title, body.Stage)
	assert.Contains(t, revision.Tags, tags[0])
	assert.Contains(t, revision.Tags, tags[1])

	assert.Contains(t, revision.Attributes, "revision_attribute")
	assert.NotContains(t, revision.Attributes, "service_attribute")
	assert.NotContains(t, revision.Attributes, "instance_attribute")
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, revision.Attributes, defs.AttrCreatedBy)

	assert.Equal(t, Unstructured, revision.Spec.Definition.Type)
	assert.Equal(t, body.serviceContext.serviceName, revision.Spec.ApiService)

	sub := util.GetAgentDetails(revision)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_revision_key")
	assert.NotContains(t, sub, "subresource_instance_key")
}

func Test_updateAPIServiceRevision(t *testing.T) {
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
		serviceContext: serviceContext{serviceName: "service-context-name"},
	}

	tags := []string{"tag1_value1", "tag2_value2"}

	client, _ := GetTestServiceClient()
	revision := &mv1a.APIServiceRevision{
		ResourceMeta: v1.ResourceMeta{
			Name:       "name",
			Title:      "oldtitle",
			Tags:       []string{"oldtag1"},
			Attributes: map[string]string{"old_attribute": "old_value"},
			Metadata: v1.Metadata{
				ResourceVersion: "123",
			},
		},
	}
	revision = client.updateAPIServiceRevision(body, revision)

	assert.Equal(t, mv1a.APIServiceRevisionGVK(), revision.GroupVersionKind)
	assert.Empty(t, revision.Metadata.ResourceVersion)
	assert.Equal(t, "name", revision.Name)

	assert.Contains(t, revision.Tags, tags[0])
	assert.Contains(t, revision.Tags, tags[1])
	assert.NotContains(t, revision.Tags, "oldtag1")

	assert.Contains(t, revision.Attributes, "revision_attribute")
	assert.NotContains(t, revision.Attributes, "service_attribute")
	assert.NotContains(t, revision.Attributes, "instance_attribute")
	assert.NotContains(t, revision.Attributes, "old_attribute")
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIStage)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIPrimaryKey)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIID)
	assert.NotContains(t, revision.Attributes, defs.AttrExternalAPIName)
	assert.NotContains(t, revision.Attributes, defs.AttrCreatedBy)

	assert.Equal(t, Unstructured, revision.Spec.Definition.Type)
	assert.Equal(t, body.serviceContext.serviceName, revision.Spec.ApiService)

	sub := util.GetAgentDetails(revision)
	assert.Equal(t, body.Stage, sub[defs.AttrExternalAPIStage])
	assert.Equal(t, body.PrimaryKey, sub[defs.AttrExternalAPIPrimaryKey])
	assert.Equal(t, body.RestAPIID, sub[defs.AttrExternalAPIID])
	assert.Equal(t, body.APIName, sub[defs.AttrExternalAPIName])
	assert.Equal(t, body.CreatedBy, sub[defs.AttrCreatedBy])
	assert.Contains(t, sub, "subresource_svc_key")
	assert.Contains(t, sub, "subresource_revision_key")
	assert.NotContains(t, sub, "subresource_instance_key")
	assert.NotContains(t, sub, "revision_attribute")
}

func Test_buildAPIServiceRevisionNilAttributes(t *testing.T) {
	client, _ := GetTestServiceClient()
	body := &ServiceBody{}

	rev := client.buildAPIServiceRevision(body, "name")
	assert.NotNil(t, rev.Attributes)

	rev.Attributes = nil
	rev = client.updateAPIServiceRevision(body, rev)
	assert.NotNil(t, rev.Attributes)
}
