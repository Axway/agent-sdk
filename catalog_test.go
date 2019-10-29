package apic

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestCreateCatalogItemBodyForAdd(t *testing.T) {
	jsonFile1, _ := os.Open("./testdata/swagger1.json") // No Security
	swaggerFile1, _ := ioutil.ReadAll(jsonFile1)
	catalogBytes1, _ := CreateCatalogItemBodyForAdd("123", "Test", "stage", swaggerFile1, []string{})

	var catalogItem1 CatalogItemInit
	json.Unmarshal(catalogBytes1, &catalogItem1)

	// Validate the security is pass-through
	if catalogItem1.Properties[0].Value.AuthPolicy != "pass-through" {
		t.Error("swagger1.json has no security, threrefore the AuthPolicy should have been pass-through. Found: ", catalogItem1.Properties[0].Value.AuthPolicy)
	}

	jsonFile2, _ := os.Open("./testdata/swagger2.json") // API Key
	swaggerFile2, _ := ioutil.ReadAll(jsonFile2)
	catalogBytes2, _ := CreateCatalogItemBodyForAdd("123", "Test", "stage", swaggerFile2, []string{})

	var catalogItem2 CatalogItemInit
	json.Unmarshal(catalogBytes2, &catalogItem2)

	// Validate the security is verify-api-key
	if catalogItem2.Properties[0].Value.AuthPolicy != "verify-api-key" {
		t.Error("swagger2.json has security, threrefore the AuthPolicy should have been verify-api-key. Found: ", catalogItem1.Properties[0].Value.AuthPolicy)
	}
}
