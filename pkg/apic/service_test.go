package apic

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

var methods = [5]string{"get", "post", "put", "patch", "delete"} // RestAPI methods

const (
	apikey      = "verify-api-key"
	passthrough = "pass-through"
	oauth       = "verify-oauth-token"
)

func determineAuthPolicyFromSwagger(swagger *[]byte) string {
	// Traverse the swagger looking for any route that has security set
	// return the security of the first route, if none- found return passthrough
	var authPolicy = passthrough

	gjson.GetBytes(*swagger, "paths").ForEach(func(_, pathObj gjson.Result) bool {
		for _, method := range methods {
			if pathObj.Get(fmt.Sprint(method, ".security.#.api_key")).Exists() {
				authPolicy = apikey
				return false
			}
			if pathObj.Get(fmt.Sprint(method, ".securityDefinitions.OAuthImplicit")).Exists() {
				authPolicy = oauth
				return false
			}
		}
		return authPolicy == passthrough // Return from path loop anonymous func, true = go to next item
	})

	if gjson.GetBytes(*swagger, "securityDefinitions.OAuthImplicit").Exists() {
		authPolicy = oauth
	}

	return authPolicy
}

func TestGetEndpointsBasedOnSwagger(t *testing.T) {

	c, _ := GetTestServiceClient()
	assert.NotNil(t, c)

	// Test oas2 json object
	oas2Json, _ := os.Open("./testdata/petstore-swagger2.json") // OAS2
	oas2Bytes, _ := ioutil.ReadAll(oas2Json)

	endPoints, err := c.getEndpointsBasedOnSwagger(oas2Bytes, Oas2)

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(443), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "/v2", endPoints[0].Routing.BasePath, "The base path was not parsed from the JSON as expected")

	// Test oas2 yaml object
	oas2Yaml, _ := os.Open("./testdata/petstore-openapi2.yaml") // OAS2
	oas2YamlBytes, _ := ioutil.ReadAll(oas2Yaml)
	endPoints, err = c.getEndpointsBasedOnSwagger(oas2YamlBytes, Oas2)

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(443), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "/v1", endPoints[0].Routing.BasePath, "The base path was not parsed from the JSON as expected")

	// Test oas3 object
	oas3Json, _ := os.Open("./testdata/petstore-openapi3.json") // OAS3
	oas3Bytes, _ := ioutil.ReadAll(oas3Json)

	endPoints, err = c.getEndpointsBasedOnSwagger(oas3Bytes, Oas3)

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas3")
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

	// Test oas3 object, with templated server URLs
	oas3Json2, _ := os.Open("./testdata/petstore-openapi3-template-urls.json") // OAS3
	oas3Bytes2, _ := ioutil.ReadAll(oas3Json2)

	endPoints, err = c.getEndpointsBasedOnSwagger(oas3Bytes2, Oas3)

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

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas3 and templated URLs")
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

	// Test wsdl object
	wsdlFile, _ := os.Open("./testdata/weather.xml") // wsdl
	wsdlBytes, _ := ioutil.ReadAll(wsdlFile)

	endPoints, err = c.getEndpointsBasedOnSwagger(wsdlBytes, Wsdl)

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with wsdl")
	assert.Len(t, endPoints, 2, "The returned end points array did not have exactly 2 endpoints")
	assert.Equal(t, "lbean006.lab.phx.axway.int", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, int32(8065), endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")
}

func TestSanitizeAPIName(t *testing.T) {
	name := sanitizeAPIName("Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc...De/f")
	assert.Equal(t, "abc--.de-f", name)
	name = sanitizeAPIName("Abc.D-ef")
	assert.Equal(t, "abc.d-ef", name)
	name = sanitizeAPIName("Abc.Def=")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName("A..bc.Def")
	assert.Equal(t, "a--bc.def", name)
}
