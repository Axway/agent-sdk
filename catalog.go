package apic

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tidwall/gjson"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/config"
	"github.com/aws/aws-sdk-go/aws"
)

//CatalogPropertyValue -
type CatalogPropertyValue struct {
	URL        string `json:"url"`
	AuthPolicy string `json:"authPolicy"`
}

//CatalogProperty -
type CatalogProperty struct {
	Key   string               `json:"key"`
	Value CatalogPropertyValue `json:"value"`
}

//CatalogRevisionProperty -
type CatalogRevisionProperty struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

//CatalogItemInitRevision -
type CatalogItemInitRevision struct {
	ID         string                    `json:"id,omitempty"`
	Properties []CatalogRevisionProperty `json:"properties"`
	Number     int                       `json:"number,omitempty"`
	Version    string                    `json:"version"`
	State      string                    `json:"state"`
	Status     string                    `json:"status,omitempty"`
}

//CatalogItemRevision -
type CatalogItemRevision struct {
	ID string `json:"id,omitempty"`
	// metadata []CatalogRevisionProperty `json:"properties"`
	Number  int    `json:"number,omitempty"`
	Version string `json:"version"`
	State   string `json:"state"`
	Status  string `json:"status,omitempty"`
}

//CatalogSubscription -
type CatalogSubscription struct {
	Enabled         bool                      `json:"enabled"`
	AutoSubscribe   bool                      `json:"autoSubscribe"`
	AutoUnsubscribe bool                      `json:"autoUnsubscribe"`
	Properties      []CatalogRevisionProperty `json:"properties"`
}

//CatalogItemInit -
type CatalogItemInit struct {
	OwningTeamID       string                  `json:"owningTeamId"`
	DefinitionType     string                  `json:"definitionType"`
	DefinitionSubType  string                  `json:"definitionSubType"`
	DefinitionRevision int                     `json:"definitionRevision"`
	Name               string                  `json:"name"`
	Description        string                  `json:"description,omitempty"`
	Properties         []CatalogProperty       `json:"properties,omitempty"`
	Tags               []string                `json:"tags,omitempty"`
	Visibility         string                  `json:"visibility"` // default: RESTRICTED
	Subscription       CatalogSubscription     `json:"subscription,omitempty"`
	Revision           CatalogItemInitRevision `json:"revision,omitempty"`
	CategoryReferences string                  `json:"categoryReferences,omitempty"`
}

//CatalogItem -
type CatalogItem struct {
	ID                 string `json:"id"`
	OwningTeamID       string `json:"owningTeamId"`
	DefinitionType     string `json:"definitionType"`
	DefinitionSubType  string `json:"definitionSubType"`
	DefinitionRevision int    `json:"definitionRevision"`
	Name               string `json:"name"`
	// relationships
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	// metadata
	Visibility string `json:"visibility"` // default: RESTRICTED
	State      string `json:"state"`      // default: UNPUBLISHED
	Access     string `json:"access,omitempty"`
	// availableRevisions
	LatestVersion        int                 `json:"latestVersion,omitempty"`
	TotalSubscriptions   int                 `json:"totalSubscriptions,omitempty"`
	LatestVersionDetails CatalogItemRevision `json:"latestVersionDetails,omitempty"`
	// image
	// categories
}

var methods = [5]string{"get", "post", "put", "patch", "delete"} // RestAPI methods
const (
	subscriptionSchema = "{\"type\": \"object\", \"$schema\": \"http://json-schema.org/draft-04/schema#\", \"description\": \"Subscription specification for API Key authentication\", \"x-axway-unique-keys\": \"APIC_APPLICATION_ID\", \"properties\": {\"applicationId\": {\"type\": \"string\", \"description\": \"Select an application\", \"x-axway-ref-apic\": \"APIC_APPLICATION_ID\"}}, \"required\":[\"applicationId\"]}"
	apikey             = "verify-api-key"
	passthrough        = "pass-through"
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
		}
		return authPolicy == passthrough // Return from path loop anonymous func, true = go to next item
	})

	return authPolicy
}

// CreateCatalogItemBodyForAdd -
func CreateCatalogItemBodyForAdd(apiID, apiName, stageName string, swagger []byte, stageTags []string) ([]byte, error) {
	fmt.Println(config.GetConfig().AWSConfig.GetRegion())
	region := config.GetConfig().AWSConfig.GetRegion()
	nameToPush := fmt.Sprintf("%v (Stage: %v)", apiName, stageName)
	desc := gjson.Get(string(swagger), "info.description")
	documentation := desc.Str
	if documentation == "" {
		documentation = "API imported from AWS APIGateway"
	}
	docBytes, err := json.Marshal(documentation)
	if err != nil {
		return nil, err
	}

	newCatalogItem := CatalogItemInit{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               nameToPush,
		OwningTeamID:       config.GetConfig().CentralConfig.GetTeamID(),
		Description:        "API From AWS APIGateway (RestApiId: " + apiID + ", StageName: " + stageName + ")",
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: determineAuthPolicyFromSwagger(&swagger),
					// URL is of the form https://<restApiId>.execute-api.<awsRegion>.amazonaws.com/<stageName>
					URL: "https://" + apiID + ".execute-api." + region + ".amazonaws.com/" + stageName,
				},
			},
		},
		Tags:       stageTags,
		Visibility: "RESTRICTED", // default value
		Subscription: CatalogSubscription{
			Enabled:         true,
			AutoSubscribe:   true,
			AutoUnsubscribe: true,
			Properties: []CatalogRevisionProperty{{
				Key:   "profile",
				Value: json.RawMessage([]byte(subscriptionSchema)),
			}},
		},
		Revision: CatalogItemInitRevision{
			Version: "1.0.0",
			State:   "PUBLISHED",
			Properties: []CatalogRevisionProperty{
				{
					Key:   "documentation",
					Value: json.RawMessage(docBytes),
				},
				{
					Key:   "swagger",
					Value: json.RawMessage(swagger),
				},
			},
		},
	}

	return json.Marshal(newCatalogItem)
}

// CreateCatalogItemBodyForUpdate -
func CreateCatalogItemBodyForUpdate(apiID, apiName, stageName string, stageTags []string) ([]byte, error) {
	nameToPush := fmt.Sprintf("%v (Stage: %v)", apiName, stageName)

	newCatalogItem := CatalogItem{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               nameToPush,
		OwningTeamID:       config.GetConfig().CentralConfig.GetTeamID(),
		Description:        "API From AWS APIGateway Updated (RestApiId: " + apiID + ", StageName: " + stageName + ")",
		Tags:               stageTags,
		Visibility:         "RESTRICTED",  // default value
		State:              "UNPUBLISHED", //default
		LatestVersionDetails: CatalogItemRevision{
			Version: "1.0.1",
			State:   "PUBLISHED",
		},
	}

	return json.Marshal(newCatalogItem)
}

// AddCatalogItem -
func AddCatalogItem(catalogBuffer []byte, agentMode corecfg.AgentMode) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	url := config.GetConfig().CentralConfig.GetCatalogItemsURL()
	return DeployAPI("POST", catalogBuffer, agentMode, url)

}

// UpdateCatalogItem -
func UpdateCatalogItem(catalogBuffer []byte, itemID *string, agentMode corecfg.AgentMode) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "", nil
	}

	url := config.GetConfig().CentralConfig.GetCatalogItemsURL() + "/" + aws.StringValue(itemID)
	return DeployAPI("PUT", catalogBuffer, agentMode, url)

}

func isUnitTesting() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
