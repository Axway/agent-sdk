package apic

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/auth"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"

	"fmt"
)

// "https://apicentral.tempenv.apicentral-k8s.axwaytest.net/api/unifiedCatalog/v1/catalogItems"

type CatalogPropertyValue struct {
	URL        string `json:"url"`
	AuthPolicy string `json:"authPolicy"`
}

type CatalogProperty struct {
	Key   string               `json:"key"`
	Value CatalogPropertyValue `json:"value"`
}

type CatalogRevisionProperty struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type CatalogRevision struct {
	Properties []CatalogRevisionProperty `json:"properties"`
	Version    string                    `json:"version"`
	State      string                    `json:"state"`
}

type CatalogSubscription struct {
	Enabled         bool                      `json:"enabled"`
	AutoSubscribe   bool                      `json:"autoSubscribe"`
	AutoUnsubscribe bool                      `json:"autoUnsubscribe"`
	Properties      []CatalogRevisionProperty `json:"properties"`
}

type CatalogItem struct {
	DefinitionType     string `json:"definitionType"`
	DefinitionSubType  string `json:"definitionSubType"`
	DefinitionRevision int    `json:"definitionRevision"`

	Name         string `json:"name"`
	OwningTeamId string `json:"owningTeamId"`
	Description  string `json:"description,omitempty"`

	Properties   []CatalogProperty   `json:"properties,omitempty"`
	Revision     CatalogRevision     `json:"revision,omitempty"`
	Subscription CatalogSubscription `json:"subscription,omitempty"`
}

var tokenRequester *apicauth.PlatformTokenGetter

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: time.Second * 10,
}

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

func CreateCatalogItem(apiID, stageName string, swagger []byte, documentation []byte) ([]byte, error) {
	newCatalogItem := CatalogItem{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               apiID + "_" + stageName,
		OwningTeamId:       apicConfig.GetTeamID(),
		Description:        "API From AWS APIGateway (RestApiId: " + apiID + ", StageName: " + stageName + ")",
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: apicConfig.GetAuthPolicy(),
					URL:        "https://f0d9c067be62dc9adeb44b57bc0eeaa601631b47.cloudapp-enterprise.appcelerator.com/music/v2",
				},
			},
		},
		Revision: CatalogRevision{
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
		Subscription: CatalogSubscription{
			Enabled:         true,
			AutoSubscribe:   true,
			AutoUnsubscribe: true,
			Properties: []CatalogRevisionProperty{{
				Key:   "profile",
				Value: json.RawMessage([]byte(subscriptionSchema)),
			}},
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

func AddCatalogItem(catalogBuffer []byte) error {
	/**
	* https://apicentral.tempenv.apicentral-k8s.axwaytest.net/api/unifiedCatalog/v1/catalogItems
	**/

	request, err := apicRequest("POST", apicConfig.GetApicURL()+"/api/unifiedCatalog/v1/catalogItems", bytes.NewBuffer(catalogBuffer))
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	detail := make(map[string]*json.RawMessage)
	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated) {

		json.NewDecoder(response.Body).Decode(&detail)
		for k, v := range detail {
			fmt.Println(k)
			buffer, _ := v.MarshalJSON()
			fmt.Println(string(buffer))
		}
		return errors.New(response.Status)
	}
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&detail)
	for k, v := range detail {
		fmt.Println(k)
		buffer, _ := v.MarshalJSON()
		fmt.Println(string(buffer))
	}
	return nil
}
