package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/auth"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
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

var tokenRequester *apicauth.PlatformTokenGetter

var httpClient = http.DefaultClient

func init() {
	tokenURL := auth.GetAuthConfig().GetTokenURL()
	aud := auth.GetAuthConfig().GetRealmURL()
	priKey := auth.GetAuthConfig().GetPrivateKey()
	pubKey := auth.GetAuthConfig().GetPublicKey()
	keyPwd := auth.GetAuthConfig().GetKeyPwd()
	clientID := auth.GetAuthConfig().GetClientID()
	authTimeout := auth.GetAuthConfig().GetAuthTimeout()
	tokenRequester = apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout)
}

const subscriptionSchema = "{\"type\": \"object\", \"$schema\": \"http://json-schema.org/draft-04/schema#\", \"description\": \"Subscription specification for API Key authentication\", \"x-axway-unique-keys\": \"APIC_APPLICATION_ID\", \"properties\": {\"applicationId\": {\"type\": \"string\", \"description\": \"Select an application\", \"x-axway-ref-apic\": \"APIC_APPLICATION_ID\"}}, \"required\":[\"applicationId\"]}"

// CreateCatalogItemBodyForAdd -
func CreateCatalogItemBodyForAdd(apiID, apiName, stageName string, swagger []byte, documentation []byte) ([]byte, error) {
	region := os.Getenv("AWS_REGION")
	nameToPush := fmt.Sprintf("%v (Stage: %v)", apiName, stageName)

	newCatalogItem := CatalogItemInit{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               nameToPush,
		OwningTeamID:       apicConfig.GetTeamID(),
		Description:        "API From AWS APIGateway (RestApiId: " + apiID + ", StageName: " + stageName + ")",
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: apicConfig.GetAuthPolicy(),
					// URL is of the form https://<restApiId>.execute-api.<awsRegion>.amazonaws.com/<stageName>
					URL: "https://" + apiID + ".execute-api." + region + ".amazonaws.com/" + stageName,
				},
			},
		},
		Tags:       []string{"tag1", "tag2"}, // todo - where do these come from?
		Visibility: "RESTRICTED",             // default value
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
					Value: json.RawMessage(string(documentation)),
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
func CreateCatalogItemBodyForUpdate(apiID, apiName, stageName string) ([]byte, error) {
	nameToPush := fmt.Sprintf("%v (Stage: %v)", apiName, stageName)

	newCatalogItem := CatalogItem{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               nameToPush,
		OwningTeamID:       apicConfig.GetTeamID(),
		Description:        "API From AWS APIGateway Updated (RestApiId: " + apiID + ", StageName: " + stageName + ")",
		Tags:               []string{"tag1", "tag2", "tag3"}, // todo - where do these come from?
		Visibility:         "RESTRICTED",                     // default value
		State:              "UNPUBLISHED",                    //default
		LatestVersionDetails: CatalogItemRevision{
			Version: "1.0.1",
			State:   "PUBLISHED",
		},
	}

	return json.Marshal(newCatalogItem)
}

func apicRequest(method, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	var token string
	if token, err = tokenRequester.GetToken(); err != nil {
		return nil, err
	}

	request.Header.Add("X-Axway-Tenant-Id", apicConfig.GetTenantID())
	request.Header.Add("Authorization", "Bearer "+token)
	return request, nil
}

// AddCatalogItem -
func AddCatalogItem(catalogBuffer []byte) (string, error) {
	/**
	* https://apicentral.tempenv.apicentral-k8s.axwaytest.net/api/unifiedCatalog/v1/catalogItems
	**/

	request, err := apicRequest("POST", apicConfig.GetApicURL()+"/api/unifiedCatalog/v1/catalogItems", bytes.NewBuffer(catalogBuffer))
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/json")

	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	detail := make(map[string]*json.RawMessage)
	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated) {

		json.NewDecoder(response.Body).Decode(&detail)
		for k, v := range detail {
			buffer, _ := v.MarshalJSON()
			log.Debugf("HTTP response key %v: %v", k, string(buffer))
		}
		return "", errors.New(response.Status)
	}
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&detail)
	itemID := ""
	for k, v := range detail {
		buffer, _ := v.MarshalJSON()
		if k == "id" {
			itemID = string(buffer)
		}
		log.Debugf("HTTP response key %v: %v", k, string(buffer))
	}
	return strconv.Unquote(itemID)
}

// UpdateCatalogItem -
func UpdateCatalogItem(catalogBuffer []byte, itemID *string) error {
	/**
	* https://apicentral.tempenv.apicentral-k8s.axwaytest.net/api/unifiedCatalog/v1/catalogItems
	**/
	request, err := apicRequest("PUT", apicConfig.GetApicURL()+"/api/unifiedCatalog/v1/catalogItems/"+aws.StringValue(itemID), bytes.NewBuffer(catalogBuffer))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return nil
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	detail := make(map[string]*json.RawMessage)
	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated) {

		json.NewDecoder(response.Body).Decode(&detail)
		for k, v := range detail {
			buffer, _ := v.MarshalJSON()
			log.Debugf("HTTP response key %v: %v", k, string(buffer))
		}
		return errors.New(response.Status)
	}
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&detail)
	for k, v := range detail {
		buffer, _ := v.MarshalJSON()
		log.Debugf("HTTP response key %v: %v", k, string(buffer))
	}
	return nil
}

func isUnitTesting() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
