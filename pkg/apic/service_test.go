package apic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

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
