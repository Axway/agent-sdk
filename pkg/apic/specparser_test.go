package apic

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createSpecParser(specFile, specType string) (specResourceParser, error) {
	specFileDescriptor, _ := os.Open(specFile)
	specData, _ := ioutil.ReadAll(specFileDescriptor)
	specParser := newSpecResourceParser(specData, specType)
	err := specParser.parse()
	return specParser, err
}

func TestSpecDiscovery(t *testing.T) {
	// JSON OAS3 specification
	specParser, err := createSpecParser("./testdata/petstore-openapi3.json", "")
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Oas3, specProcessor.getResourceType())
	_, ok := specProcessor.(*Oas3SpecProcessor)
	assert.True(t, ok)

	// YAML OAS2 specification
	specParser, err = createSpecParser("./testdata/petstore-openapi2.yaml", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Oas2, specProcessor.getResourceType())
	_, ok = specProcessor.(*Oas2SpecProcessor)
	assert.True(t, ok)

	// JSON OAS2 specification
	specParser, err = createSpecParser("./testdata/petstore-swagger2.json", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Oas2, specProcessor.getResourceType())
	_, ok = specProcessor.(*Oas2SpecProcessor)
	assert.True(t, ok)

	// WSDL specification
	specParser, err = createSpecParser("./testdata/weather.xml", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Wsdl, specProcessor.getResourceType())
	_, ok = specProcessor.(*wsdlProcessor)
	assert.True(t, ok)

	// Protobuf specification
	specParser, err = createSpecParser("./testdata/petstore.proto", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Protobuf, specProcessor.getResourceType())
	_, ok = specProcessor.(*protobufProcessor)
	assert.True(t, ok)

	// AsyncAPI specification
	specParser, err = createSpecParser("./testdata/asyncapi-sample.yaml", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, AsyncAPI, specProcessor.getResourceType())
	_, ok = specProcessor.(*asyncAPIProcessor)
	assert.True(t, ok)

	// Unstructured specification
	specParser, err = createSpecParser("./testdata/multiplication.thrift", "")
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Unstructured, specProcessor.getResourceType())
	_, ok = specProcessor.(*unstructuredProcessor)
	assert.True(t, ok)
}

func TestSpecOAS3Processors(t *testing.T) {
	// JSON OAS3 specification
	specParser, err := createSpecParser("./testdata/petstore-openapi3.json", Protobuf)
	assert.NotNil(t, err)

	// JSON OAS3 specification
	specParser, err = createSpecParser("./testdata/petstore-openapi3.json", Oas3)
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Oas3, specProcessor.getResourceType())
	_, ok := specProcessor.(*Oas3SpecProcessor)
	assert.True(t, ok)

	endPoints, err := specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with oas3")
	assert.Len(t, endPoints, 3, "The returned end points array did not have exactly 3 endpoints")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The first returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(8080), endPoints[0].Port, "The first returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[0].Protocol, "The first returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "petstore.swagger.io", endPoints[1].Host, "The second returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(80), endPoints[1].Port, "The second returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[1].Protocol, "The second returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "petstore.swagger.io", endPoints[2].Host, "The third returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(443), endPoints[2].Port, "The third returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[2].Protocol, "The third returned end point had an unexpected value for it's protocol")

	specParser, err = createSpecParser("./testdata/petstore-openapi3-template-urls.json", Oas3)
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	endPoints, err = specProcessor.getEndpoints()

	type verification struct {
		Host     string
		Port     int32
		Protocol string
		Found    bool
	}

	possibleEndpoints := []verification{
		{
			Host:     "petstore.swagger.io",
			Port:     443,
			Protocol: "https",
		},
		{
			Host:     "petstore.swagger.io",
			Port:     80,
			Protocol: "http",
		},
		{
			Host:     "petstore-preprod.swagger.io",
			Port:     443,
			Protocol: "https",
		},
		{
			Host:     "petstore-preprod.swagger.io",
			Port:     80,
			Protocol: "http",
		},
		{
			Host:     "petstore-test.swagger.io",
			Port:     443,
			Protocol: "https",
		},
		{
			Host:     "petstore-test.swagger.io",
			Port:     80,
			Protocol: "http",
		},
	}

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with oas3 and templated URLs")
	assert.Len(t, endPoints, 6, "The returned end points array did not have exactly 6 endpoints")
	endpointNotFound := false
	for _, endpoint := range endPoints {
		found := false
		for i, possibleEndpoint := range possibleEndpoints {
			if possibleEndpoint.Found {
				continue // Can't find the same endpoint twice
			}
			if endpoint.Host == possibleEndpoint.Host && endpoint.Port == possibleEndpoint.Port && endpoint.Protocol == possibleEndpoint.Protocol {
				found = true
				possibleEndpoints[i].Found = true
				continue // No need to keep looking once we find it
			}
		}
		if !found {
			endpointNotFound = true
		}
	}

	// Check that all endpoints have been verified
	assert.False(t, endpointNotFound, "At least one endpoint returned was not expected")
	for _, possibleEndpoint := range possibleEndpoints {
		assert.True(t, possibleEndpoint.Found, "Did not find an endpoint with Host(%s), Port(%d), and Protocol(%s) in the returned endpoint array", possibleEndpoint.Host, possibleEndpoint.Port, possibleEndpoint.Protocol)
	}
}

func TestSpecOAS2Processors(t *testing.T) {
	specParser, err := createSpecParser("./testdata/petstore-swagger2.json", Protobuf)
	assert.NotNil(t, err)

	// JSON OAS3 specification
	specParser, err = createSpecParser("./testdata/petstore-swagger2.json", Oas2)
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Oas2, specProcessor.getResourceType())
	_, ok := specProcessor.(*Oas2SpecProcessor)
	assert.True(t, ok)

	endPoints, err := specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(443), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "/v2", endPoints[0].BasePath, "The base path was not parsed from the JSON as expected")

	specParser, err = createSpecParser("./testdata/petstore-openapi2.yaml", Oas2)
	assert.Nil(t, err)
	specProcessor = specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	endPoints, err = specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(80), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "/v1", endPoints[0].BasePath, "The base path was not parsed from the JSON as expected")
}

func TestSpecWsdlProcessors(t *testing.T) {
	specParser, err := createSpecParser("./testdata/weather.xml", Oas3)
	assert.NotNil(t, err)

	// JSON OAS3 specification
	specParser, err = createSpecParser("./testdata/weather.xml", Wsdl)
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Wsdl, specProcessor.getResourceType())
	_, ok := specProcessor.(*wsdlProcessor)
	assert.True(t, ok)

	endPoints, err := specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with wsdl")
	assert.Len(t, endPoints, 2, "The returned end points array did not have exactly 2 endpoints")
	assert.Equal(t, "beano.com", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(8065), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
}

func TestSpecProtobufProcessors(t *testing.T) {
	specParser, err := createSpecParser("./testdata/petstore.proto", AsyncAPI)
	assert.NotNil(t, err)

	// JSON OAS3 specification
	specParser, err = createSpecParser("./testdata/petstore.proto", Protobuf)
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, Protobuf, specProcessor.getResourceType())
	_, ok := specProcessor.(*protobufProcessor)
	assert.True(t, ok)

	endPoints, err := specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with protobuf")
	assert.Len(t, endPoints, 0, "The returned end points array is not empty")
}

func TestSpecAsyncAPIProcessors(t *testing.T) {
	specParser, err := createSpecParser("./testdata/asyncapi-sample.yaml", Protobuf)
	assert.NotNil(t, err)

	// JSON OAS3 specification
	specParser, err = createSpecParser("./testdata/asyncapi-sample.yaml", AsyncAPI)
	assert.Nil(t, err)
	specProcessor := specParser.getSpecProcessor()
	assert.NotNil(t, specProcessor)
	assert.Equal(t, AsyncAPI, specProcessor.getResourceType())
	_, ok := specProcessor.(*asyncAPIProcessor)
	assert.True(t, ok)

	endPoints, err := specProcessor.getEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with asyncapi")
	assert.Equal(t, "api.company.com", endPoints[0].Host)
	assert.Equal(t, int32(5676), endPoints[0].Port)
	assert.Equal(t, "mqtt", endPoints[0].Protocol)
	assert.Equal(t, "", endPoints[0].BasePath)
}
