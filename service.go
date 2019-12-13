package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"github.com/tidwall/gjson"
)

type serviceExecution int

const (
	addAPIServerSpec serviceExecution = iota + 1
	addAPIServerRevisionSpec
	addAPIServerInstanceSpec
	addCatalog
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

// CatalogItemImage -
type CatalogItemImage struct {
	DataType      string `json:"data,omitempty"`
	Base64Content string `json:"base64,omitempty"`
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
	Image                *CatalogItemImage   `json:"image,omitempty"`
	// categories
}

// APIServer -
type APIServer struct {
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Spec       interface{}            `json:"spec"`
}

// APIServiceSpec -
type APIServiceSpec struct {
	Description string `json:"description"`
}

// APIServiceRevisionSpec -
type APIServiceRevisionSpec struct {
	APIServiceRef string             `json:"apiServiceRef"`
	Definition    RevisionDefinition `json:"definition"`
}

// RevisionDefinition -
type RevisionDefinition struct {
	Type  string `json:"type,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// APIServerInstanceSpec -
type APIServerInstanceSpec struct {
	APIServiceRevisionRef string     `json:"apiServiceRevisionRef,omitempty"`
	InstanceEndPoint      []EndPoint `json:"endpoint,omitempty"`
}

// EndPoint -
type EndPoint struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

const (
	subscriptionSchema = "{\"type\": \"object\", \"$schema\": \"http://json-schema.org/draft-04/schema#\", \"description\": \"Subscription specification for API Key authentication\", \"x-axway-unique-keys\": \"APIC_APPLICATION_ID\", \"properties\": {\"applicationId\": {\"type\": \"string\", \"description\": \"Select an application\", \"x-axway-ref-apic\": \"APIC_APPLICATION_ID\"}}, \"required\":[\"applicationId\"]}"
)

// CreateService -
func (c *Client) CreateService(serviceBody ServiceBody) ([]byte, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return nil, fmt.Errorf("Unsuppored security policy '%v'. ", serviceBody.AuthPolicy)
	}
	if serviceBody.AgentMode == config.Connected {
		return createAPIServerBody(c, serviceBody)
	}
	return createCatalogBody(c, serviceBody)
}

func createCatalogBody(c *Client, serviceBody ServiceBody) ([]byte, error) {
	newCatalogItem := CatalogItemInit{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: serviceBody.AuthPolicy,
					URL:        serviceBody.URL,
				},
			},
		},

		Tags:       c.MapToStringArray(serviceBody.Tags),
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
			Version: serviceBody.Version,
			State:   "PUBLISHED",
			Properties: []CatalogRevisionProperty{
				{
					Key:   "documentation",
					Value: json.RawMessage(string(serviceBody.Documentation)),
				},
				{
					Key:   "swagger",
					Value: json.RawMessage(serviceBody.Swagger),
				},
			},
		},
	}

	return json.Marshal(newCatalogItem)
}

func isValidAuthPolicy(auth string) bool {
	for _, item := range ValidPolicies {
		if item == auth {
			return true
		}
	}
	return false
}

// createAPIServerBody - This function is being used by both the api server creation and api server revision creation
func createAPIServerBody(c *Client, serviceBody ServiceBody) ([]byte, error) {
	// Set tags as Attributes to retain key value pairs.  Add other pertinent data.
	attributes := make(map[string]interface{})
	for key, val := range serviceBody.Tags {
		v := val.(*string)
		attributes[key] = *v
	}

	// spec needs to adhere to environment schema
	var spec interface{}
	name := strings.ToLower(serviceBody.APIName) // name needs to be path friendly and follows this regex "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\

	switch serviceBody.ServiceExecution {
	case int(addAPIServerSpec):
		spec = APIServiceSpec{
			Description: serviceBody.Description,
		}
	case int(addAPIServerRevisionSpec):
		name = strings.ToLower(serviceBody.APIName) + strings.ToLower(serviceBody.Stage)
		revisionDefinition := RevisionDefinition{
			Type:  "swagger2",
			Value: serviceBody.Swagger,
		}
		spec = APIServiceRevisionSpec{
			APIServiceRef: strings.ToLower(serviceBody.APIName),
			Definition:    revisionDefinition,
		}
	case int(addAPIServerInstanceSpec):
		name = strings.ToLower(serviceBody.APIName) + strings.ToLower(serviceBody.Stage)
		host := gjson.Get(string(serviceBody.Swagger), "host").String()
		protocol := gjson.Get(string(serviceBody.Swagger), "schemes").String()

		endPoint := EndPoint{
			Host:     host,
			Port:     443, // TODO : this is a hard coded value as of now.  Port is not showing up in swagger at the time of check in
			Protocol: protocol,
		}
		spec = APIServerInstanceSpec{
			APIServiceRevisionRef: name,
			InstanceEndPoint:      []EndPoint{endPoint},
		}
	default:
		return nil, errors.New("Error getting execution service -- not set")
	}

	newtags := c.MapToStringArray(serviceBody.Tags)

	apiServerService := APIServer{
		Name:       name,
		Title:      serviceBody.NameToPush,
		Attributes: attributes,
		Spec:       spec,
		Tags:       newtags,
	}

	return json.Marshal(apiServerService)
}

// CreateCatalogItemBodyForUpdate -
func (c *Client) CreateCatalogItemBodyForUpdate(serviceBody ServiceBody) ([]byte, error) {
	newCatalogItem := CatalogItem{
		DefinitionType:    "API",
		DefinitionSubType: "swaggerv2",

		DefinitionRevision: 1,
		Name:               serviceBody.NameToPush,
		OwningTeamID:       serviceBody.TeamID,
		Description:        serviceBody.Description,
		Tags:               c.MapToStringArray(serviceBody.Tags),
		Visibility:         "RESTRICTED",  // default value
		State:              "UNPUBLISHED", //default
		LatestVersionDetails: CatalogItemRevision{
			Version: serviceBody.Version,
			State:   "PUBLISHED",
		},
	}

	return json.Marshal(newCatalogItem)
}

// ExecuteService - Used for both Adding and Updating catalog item.
// The Method will either be POST (add) or PUT (update)
func (c *Client) ExecuteService(service Service) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	return c.DeployAPI(service)
}

func isUnitTesting() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
