package apic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	corecfg "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/config"
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

func createServiceClient() *ServiceClient {
	cfg := &corecfg.CentralConfiguration{
		TeamID: "test",
		Auth: &corecfg.AuthConfiguration{
			URL:      "http://localhost:8888",
			Realm:    "Broker",
			ClientID: "dummy",
		},
	}
	c := New(cfg)
	return c.(*ServiceClient)
}
func TestCreateCatalogItemBodyForAdd(t *testing.T) {
	// set the config values
	c := createServiceClient()
	tags := make(map[string]interface{})
	tags["key1"] = "val1"
	tags["key2"] = "val2"

	jsonFile1, _ := os.Open("./testdata/swagger1.json") // No Security
	swaggerFile1, _ := ioutil.ReadAll(jsonFile1)
	authPolicy := determineAuthPolicyFromSwagger(&swaggerFile1)
	desc := gjson.Get(string(swaggerFile1), "info.description")
	documentation := desc.Str
	if documentation == "" {
		documentation = "API imported from AWS API Gateway"
	}
	docBytes, _ := json.Marshal(documentation)

	serviceBody := ServiceBody{
		NameToPush:    "Beano",
		APIName:       "serviceapi1",
		URL:           "https://restapiID.execute-api.eu-west.amazonaws.com/stage",
		Description:   "API From AWS API Gateway (RestApiId: restapiID, StageName: stage",
		Version:       "1.0.0",
		AuthPolicy:    authPolicy,
		Swagger:       swaggerFile1,
		Documentation: docBytes,
		Tags:          tags,
	}

	catalogBytes1, _ := c.marshalCatalogItemInit(serviceBody)

	var catalogItem1 CatalogItemInit
	json.Unmarshal(catalogBytes1, &catalogItem1)

	// Validate the security is pass-through
	if catalogItem1.Properties[0].Value.AuthPolicy != "pass-through" {
		t.Error("swagger1.json has no security, therefore the AuthPolicy should have been pass-through. Found: ", catalogItem1.Properties[0].Value.AuthPolicy)
	}

	jsonFile2, _ := os.Open("./testdata/swagger2.json") // API Key
	swaggerFile2, _ := ioutil.ReadAll(jsonFile2)
	authPolicy = determineAuthPolicyFromSwagger(&swaggerFile2)
	desc = gjson.Get(string(swaggerFile2), "info.description")
	documentation = desc.Str
	if documentation == "" {
		documentation = "API imported from AWS API Gateway"
	}
	docBytes, _ = json.Marshal(documentation)
	serviceBody = ServiceBody{
		NameToPush:    "Beano",
		APIName:       "serviceapi1",
		URL:           "https://restapiID.execute-api.eu-west.amazonaws.com/stage",
		Description:   "API From AWS API Gateway (RestApiId: restapiID, StageName: stage",
		Version:       "1.0.0",
		AuthPolicy:    authPolicy,
		Swagger:       swaggerFile2,
		Documentation: docBytes,
		Tags:          tags,
	}

	catalogBytes2, _ := c.marshalCatalogItemInit(serviceBody)

	var catalogItem2 CatalogItemInit
	json.Unmarshal(catalogBytes2, &catalogItem2)

	// Validate the security is verify-api-key
	if catalogItem2.Properties[0].Value.AuthPolicy != "verify-api-key" {
		t.Error("swagger2.json has security, therefore the AuthPolicy should have been verify-api-key. Found: ", catalogItem2.Properties[0].Value.AuthPolicy)
	}

	jsonFile3, _ := os.Open("./testdata/swagger3.json") // Oauth
	swaggerFile3, _ := ioutil.ReadAll(jsonFile3)
	authPolicy = determineAuthPolicyFromSwagger(&swaggerFile3)
	desc = gjson.Get(string(swaggerFile1), "info.description")
	documentation = desc.Str
	if documentation == "" {
		documentation = "API imported from Axway API Gateway"
	}
	docBytes, _ = json.Marshal(documentation)
	serviceBody = ServiceBody{
		NameToPush:    "Beano",
		APIName:       "serviceapi1",
		URL:           "https://restapiID.execute-api.eu-west.amazonaws.com/stage",
		Description:   "API From Axway API Gateway (RestApiId: restapiID, StageName: stage",
		Version:       "1.0.0",
		AuthPolicy:    authPolicy,
		Swagger:       swaggerFile3,
		Documentation: docBytes,
		Tags:          tags,
	}

	catalogBytes3, _ := c.marshalCatalogItemInit(serviceBody)

	var catalogItem3 CatalogItemInit
	json.Unmarshal(catalogBytes3, &catalogItem3)

	// Validate the security is verify-api-key
	if catalogItem3.Properties[0].Value.AuthPolicy != "verify-oauth-token" {
		t.Error("swagger3.json has security, therefore the AuthPolicy should have been verify-oauth-token. Found: ", catalogItem3.Properties[0].Value.AuthPolicy)
	}
}

func TestGetEndpointsBasedOnSwagger(t *testing.T) {

	c := createServiceClient()

	// Test oas2 object
	oas2Json, _ := os.Open("./testdata/petstore-swagger2.json") // OAS2
	oas2Bytes, _ := ioutil.ReadAll(oas2Json)

	endPoints, err := c.getEndpointsBasedOnSwagger(oas2Bytes, "oas2")

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas2")
	assert.Len(t, endPoints, 1, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The returned end point had an unexpected value for it's host")
	assert.Equal(t, 443, endPoints[0].Port, "The returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[0].Protocol, "The returned end point had an unexpected value for it's protocol")

	// Test oas2 object
	oas3Json, _ := os.Open("./testdata/petstore-openapi3.json") // OAS3
	oas3Bytes, _ := ioutil.ReadAll(oas3Json)

	endPoints, err = c.getEndpointsBasedOnSwagger(oas3Bytes, "oas3")

	assert.Nil(t, err, "An unexpected Error was returned from getEndpointsBasedOnSwagger with oas3")
	assert.Len(t, endPoints, 3, "The returned end points array did not have exactly 1 endpoint")
	assert.Equal(t, "petstore.swagger.io", endPoints[0].Host, "The first returned end point had an unexpected value for it's host")
	assert.Equal(t, 8080, endPoints[0].Port, "The first returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[0].Protocol, "The first returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "petstore.swagger.io", endPoints[1].Host, "The second returned end point had an unexpected value for it's host")
	assert.Equal(t, 80, endPoints[1].Port, "The second returned end point had an unexpected value for it's port")
	assert.Equal(t, "http", endPoints[1].Protocol, "The second returned end point had an unexpected value for it's protocol")
	assert.Equal(t, "petstore.swagger.io", endPoints[2].Host, "The third returned end point had an unexpected value for it's host")
	assert.Equal(t, 443, endPoints[2].Port, "The third returned end point had an unexpected value for it's port")
	assert.Equal(t, "https", endPoints[2].Protocol, "The third returned end point had an unexpected value for it's protocol")
}
