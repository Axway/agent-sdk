package apic

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func createSpecParser(specFile, specType string) (SpecResourceParser, error) {
	specFileDescriptor, _ := os.Open(specFile)
	specData, _ := io.ReadAll(specFileDescriptor)
	specParser := NewSpecResourceParser(specData, specType)
	err := specParser.Parse()
	return specParser, err
}

func TestSpecDiscovery(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		inputType    string
		parseErr     bool
		expectedType string
	}{
		{
			name:      "Protobuf input type with OAS3 Spec",
			inputFile: "./testdata/petstore-openapi3.json",
			parseErr:  true,
			inputType: Protobuf,
		},
		{
			name:      "Protobuf input type with OAS2 Spec",
			inputFile: "./testdata/petstore-openapi2.yaml",
			parseErr:  true,
			inputType: AsyncAPI,
		},
		{
			name:      "OAS3 input type with WSDL Spec",
			inputFile: "./testdata/weather.xml",
			parseErr:  true,
			inputType: Oas3,
		},
		{
			name:      "AsyncAPI input type with Protobuf Spec",
			inputFile: "./testdata/petstore.proto",
			parseErr:  true,
			inputType: Oas2,
		},
		{
			name:      "Protobuf input type with AsyncAPI Spec",
			inputFile: "./testdata/asyncapi-sample.yaml",
			parseErr:  true,
			inputType: Wsdl,
		},
		{
			name:      "Raml input type with no valid raml version provided",
			inputFile: "./testdata/raml_invalid.raml",
			parseErr:  true,
			inputType: Raml,
		},
		{
			name:         "No input type bad OAS version creates Unstructured",
			inputFile:    "./testdata/petstore-openapi-bad-version.json",
			expectedType: Unstructured,
		},
		{
			name:         "No input type bad Swagger version creates Unstructured",
			inputFile:    "./testdata/petstore-swagger-bad-version.json",
			expectedType: Unstructured,
		},
		{
			name:         "No input type OAS3 Spec",
			inputFile:    "./testdata/petstore-openapi3.json",
			expectedType: Oas3,
		},
		{
			name:         "No input type OAS2 Spec",
			inputFile:    "./testdata/petstore-openapi2.yaml",
			expectedType: Oas2,
		},
		{
			name:         "No input type OAS2 swagger Spec",
			inputFile:    "./testdata/petstore-swagger2.json",
			expectedType: Oas2,
		},
		{
			name:         "No input type WSDL Spec",
			inputFile:    "./testdata/weather.xml",
			expectedType: Wsdl,
		},
		{
			name:         "No input type Protobuf Spec",
			inputFile:    "./testdata/petstore.proto",
			expectedType: Protobuf,
		},
		{
			name:         "No input type AsyncAPI Spec YAML",
			inputFile:    "./testdata/asyncapi-sample.yaml",
			expectedType: AsyncAPI,
		},
		{
			name:         "No input type Raml 1.0 spec",
			inputFile:    "./testdata/raml_10.raml",
			expectedType: Raml,
		},
		{
			name:         "No input type Raml 0.8 spec",
			inputFile:    "./testdata/raml_08.raml",
			expectedType: Raml,
		},
		{
			name:         "No input type Unstructured",
			inputFile:    "./testdata/multiplication.thrift",
			expectedType: Unstructured,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			specParser, err := createSpecParser(tc.inputFile, tc.inputType)
			if tc.parseErr {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
			specProcessor := specParser.GetSpecProcessor()
			assert.NotNil(t, specProcessor)
			assert.Equal(t, tc.expectedType, specProcessor.GetResourceType())
			if tc.expectedType != specProcessor.GetResourceType() {
				return
			}
			ok := false
			switch tc.expectedType {
			case Oas3:
				_, ok = specProcessor.(*oas3SpecProcessor)
				ValidateOAS3Processors(t, specParser)
			case Oas2:
				_, ok = specProcessor.(*oas2SpecProcessor)
				ValidateOAS2Processors(t, specParser, tc.inputFile)
			case Wsdl:
				_, ok = specProcessor.(*wsdlProcessor)
				ValidateWsdlProcessors(t, specParser)
			case Protobuf:
				_, ok = specProcessor.(*protobufProcessor)
				ValidateProtobufProcessors(t, specParser)
			case AsyncAPI:
				_, ok = specProcessor.(*asyncAPIProcessor)
				ValidateAsyncAPIProcessors(t, specParser)
			case Raml:
				_, ok = specProcessor.(*ramlProcessor)
				ValidateRamlProcessors(t, specParser, tc.inputFile)
			case Unstructured:
				_, ok = specProcessor.(*unstructuredProcessor)
			}
			assert.True(t, ok)
		})
	}
}

func TestLoadRamlAsYaml(t *testing.T) {
	var v map[string]interface{}
	yamlFile, err := os.ReadFile("./testdata/raml_08.raml")
	assert.Nil(t, err)

	err = yaml.Unmarshal(yamlFile, &v)
	assert.Nil(t, err)

	yamlFile, err = os.ReadFile("./testdata/raml_10.raml")
	assert.Nil(t, err)

	err = yaml.Unmarshal(yamlFile, &v)
	assert.Nil(t, err)
}

func ValidateOAS3Processors(t *testing.T, specParser SpecResourceParser) {
	// JSON OAS3 specification
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()

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
	specProcessor = specParser.GetSpecProcessor()
	assert.NotNil(t, specProcessor)
	endPoints, err = specProcessor.GetEndpoints()

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

func ValidateOAS2Processors(t *testing.T, specParser SpecResourceParser, inputFile string) {
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	if inputFile == "./testdata/petstore-swagger2.json" {
		assert.Equal(t, int32(443), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
		assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
		assert.Equal(t, "/v2", endPoints[0].BasePath, "The base path was not parsed from the JSON as expected")
	} else {
		assert.Equal(t, int32(80), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
		assert.Equal(t, "http", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
		assert.Equal(t, "/v1", endPoints[0].BasePath, "The base path was not parsed from the JSON as expected")
	}
}

func ValidateWsdlProcessors(t *testing.T, specParser SpecResourceParser) {
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with wsdl")
	assert.Len(t, endPoints, 2, "The returned end points array did not have exactly 2 endpoints")
	assert.Equal(t, "beano.com", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(8065), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
}

func ValidateProtobufProcessors(t *testing.T, specParser SpecResourceParser) {
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with protobuf")
	assert.Len(t, endPoints, 0, "The returned end points array is not empty")
}

func ValidateAsyncAPIProcessors(t *testing.T, specParser SpecResourceParser) {
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()

	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with asyncapi")
	assert.Equal(t, "api.company.com", endPoints[0].Host)
	assert.Equal(t, int32(5676), endPoints[0].Port)
	assert.Equal(t, "mqtt", endPoints[0].Protocol)
	assert.Equal(t, "", endPoints[0].BasePath)
	assert.Equal(t, map[string]interface{}{
		"solace": map[string]interface{}{
			"msgVpn":  "apim-test",
			"version": "0.2.0",
		},
	}, endPoints[0].Details)
}

func ValidateRamlProcessors(t *testing.T, specParser SpecResourceParser, inputFile string) {
	specProcessor := specParser.GetSpecProcessor()
	endPoints, err := specProcessor.GetEndpoints()
	description := specProcessor.GetDescription()
	version := specProcessor.GetVersion()
	assert.Nil(t, err, "An unexpected Error was returned from getEndpoints with raml")
	if inputFile == "./testdata/raml_10.raml" {
		for i := range endPoints {
			assert.True(t, isInList(endPoints[i].Protocol, []string{"http", "https"}))
			assert.True(t, isInList(endPoints[i].Port, []int32{80, 443}))
		}
		assert.Equal(t, "na1.salesforce.com", endPoints[0].Host)
		assert.Equal(t, "/services/data/v3/chatter", endPoints[0].BasePath)
		assert.Equal(t, "Grand Theft Auto:Vice City", description)
		assert.Equal(t, "v3", version)
	} else if inputFile == "./testdata/raml_08.raml" {
		assert.Equal(t, "Sonny Forelli", description)
		assert.Equal(t, "1.0", version)
		assert.Equal(t, "example.local", endPoints[0].Host)
		assert.Equal(t, endPoints[0].Protocol, "https")
		assert.Equal(t, endPoints[0].Port, int32(8000))
		assert.Equal(t, "/api", endPoints[0].BasePath)
	}
}

func isInList[T comparable](actual T, validValues []T) bool {
	for i := range validValues {
		if validValues[i] == actual {
			return true
		}
	}
	return false
}
