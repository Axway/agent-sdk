package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceBodyBuilder(t *testing.T) {
	svcBody, err := NewServiceBodyBuilder().Build()
	assert.Nil(t, err)
	assert.NotNil(t, svcBody)

	// test all the default values
	assert.Equal(t, Passthrough, svcBody.AuthPolicy)
	assert.Equal(t, Oas3, svcBody.ResourceType)
	assert.Equal(t, PublishedState, svcBody.State)
	assert.Equal(t, PublishedStatus, svcBody.Status)
}
func TestServiceBodySetters(t *testing.T) {
	tags := map[string]interface{}{"tag1": "t1", "tag2": "t2"}
	attribs := map[string]string{"attrib1": "a1", "attrib2": "a2"}
	sb, err := NewServiceBodyBuilder().
		SetTitle("sbtitle").
		SetAPIName("testAPI").
		SetID("1234").
		SetStage("teststage").
		SetURL("https://1234.execute-api.us-region.amazonaws.com/teststage").
		SetDescription("test description").
		SetVersion("1.0.0").
		SetAuthPolicy("Oauth").
		SetAPISpec([]byte{}).
		SetDocumentation([]byte("documentation")).
		SetAPIUpdateSeverity("MAJOR").
		SetStatus(UnpublishedStatus).
		SetState(PublishedStatus).
		SetSubscriptionName("testsubscription").
		SetAPISpec([]byte{}).
		SetImage("image").
		SetImageContentType("image/jpeg").
		SetResourceType("foobar").
		SetTags(tags).
		SetServiceAttribute(attribs).
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, sb)
	assert.Equal(t, "sbtitle", sb.NameToPush)
	assert.Equal(t, "testAPI", sb.APIName)
	assert.Equal(t, "1234", sb.RestAPIID)
	assert.Equal(t, "teststage", sb.Stage)
	assert.Equal(t, "https://1234.execute-api.us-region.amazonaws.com/teststage", sb.URL)
	assert.Equal(t, "test description", sb.Description)
	assert.Equal(t, "1.0.0", sb.Version)
	assert.Equal(t, "Oauth", sb.AuthPolicy)
	assert.Equal(t, UnpublishedStatus, sb.Status)
	assert.Equal(t, PublishedStatus, sb.State)

	assert.Equal(t, []byte{}, sb.Swagger)
	assert.Equal(t, []byte("documentation"), sb.Documentation)

	assert.Equal(t, "image/jpeg", sb.ImageContentType)
	assert.Equal(t, "image", sb.Image)
	assert.Equal(t, "foobar", sb.ResourceType)

	assert.Equal(t, "MAJOR", sb.APIUpdateSeverity)
	assert.Len(t, sb.Tags, 2)
	assert.Equal(t, "t2", sb.Tags["tag2"])
	assert.Len(t, sb.ServiceAttributes, 2)
	assert.Equal(t, "a2", sb.ServiceAttributes["attrib2"])
}
