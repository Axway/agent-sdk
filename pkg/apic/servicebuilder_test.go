package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const longDescription = `This is a sample Pet Store Server based on the OpenAPI 3.0 specification.  You can find out more about
Swagger at [http://swagger.io](http://swagger.io). In the third iteration of the pet store, we've switched to the design first approach!
You can now help us improve the API whether it's by making changes to the definition itself or to the code.
That way, with time, we can improve the API in general, and expose some of the new features in OAS3.

Some useful links:
- [The Pet Store repository](https://github.com/swagger-api/swagger-petstore)
- [The source API definition for the Pet Store](https://github.com/swagger-api/swagger-petstore/blob/master/src/main/resources/openapi.yaml)`

func TestNewServiceBodyBuilder(t *testing.T) {
	svcBody, err := NewServiceBodyBuilder().Build()
	assert.Nil(t, err)
	assert.NotNil(t, svcBody)

	// test all the default values
	assert.Equal(t, Passthrough, svcBody.AuthPolicy)
	assert.Equal(t, Unstructured, svcBody.ResourceType)
	assert.Equal(t, PublishedState, svcBody.State)
	assert.Equal(t, PublishedStatus, svcBody.Status)
}

func TestServiceBodySetters(t *testing.T) {
	tags := map[string]interface{}{"tag1": "t1", "tag2": "t2"}
	attribs := map[string]string{"attrib1": "a1", "attrib2": "a2"}
	serviceBuilder := NewServiceBodyBuilder()
	sb, err := serviceBuilder.
		SetTitle("sbtitle").
		SetAPIName("testAPI").
		SetID("1234").
		SetStage("teststage").
		SetURL("https://1234.execute-api.us-region.amazonaws.com/teststage").
		SetDescription(longDescription).
		SetVersion("1.0.0").
		SetAuthPolicy("Oauth").
		SetAPISpec([]byte{}).
		SetDocumentation([]byte("documentation")).
		SetAPIUpdateSeverity("MAJOR").
		SetStatus(UnpublishedStatus).
		SetState(PublishedStatus).
		SetSubscriptionName("testsubscription").
		SetAPISpec([]byte{}).
		AddServiceEndpoint("https", "test.com", 443, "/test").
		SetImage("image").
		SetImageContentType("image/jpeg").
		SetResourceType("foobar").
		SetTags(tags).
		SetServiceAttribute(attribs).
		SetUnstructuredContentType("application/zip").
		SetUnstructuredFilename("test.zip").
		SetUnstructuredLabel("Label").
		SetUnstructuredType("Type").
		Build()

	assert.Nil(t, err)
	assert.NotNil(t, sb)
	assert.Equal(t, "sbtitle", sb.NameToPush)
	assert.Equal(t, "testAPI", sb.APIName)
	assert.Equal(t, "1234", sb.RestAPIID)
	assert.Equal(t, "teststage", sb.Stage)
	assert.Equal(t, "https://1234.execute-api.us-region.amazonaws.com/teststage", sb.URL)
	assert.Equal(t, 350, len(sb.Description))

	description := longDescription[0:maxDescriptionLength-len(strEllipse)] + strEllipse
	assert.Equal(t, description, sb.Description)
	assert.Equal(t, "1.0.0", sb.Version)
	assert.Equal(t, "Oauth", sb.AuthPolicy)
	assert.Equal(t, UnpublishedStatus, sb.Status)
	assert.Equal(t, PublishedStatus, sb.State)

	assert.Equal(t, []byte{}, sb.SpecDefinition)
	assert.Equal(t, []byte("documentation"), sb.Documentation)

	assert.Equal(t, "image/jpeg", sb.ImageContentType)
	assert.Equal(t, "image", sb.Image)
	assert.Equal(t, Unstructured, sb.ResourceType)

	assert.Equal(t, "MAJOR", sb.APIUpdateSeverity)
	assert.Len(t, sb.Tags, 2)
	assert.Equal(t, "t2", sb.Tags["tag2"])
	assert.Len(t, sb.ServiceAttributes, 2)
	assert.Equal(t, "a2", sb.ServiceAttributes["attrib2"])
	assert.Equal(t, "application/zip", sb.UnstructuredProps.ContentType)
	assert.Equal(t, "test.zip", sb.UnstructuredProps.Filename)
	assert.Equal(t, "Label", sb.UnstructuredProps.Label)
	assert.Equal(t, "Type", sb.UnstructuredProps.AssetType)
}

func TestServiceBodyWithParseError(t *testing.T) {
	serviceBuilder := NewServiceBodyBuilder()
	_, err := serviceBuilder.SetResourceType(Oas3).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.NotNil(t, err)

	_, err = serviceBuilder.SetResourceType(Oas2).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.NotNil(t, err)

	_, err = serviceBuilder.SetResourceType(Wsdl).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.NotNil(t, err)

	_, err = serviceBuilder.SetResourceType(Protobuf).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.NotNil(t, err)

	_, err = serviceBuilder.SetResourceType(AsyncAPI).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.NotNil(t, err)

	_, err = serviceBuilder.SetResourceType(Unstructured).SetAPISpec([]byte("{\"test\":\"123\"}")).Build()
	assert.Nil(t, err)
}
