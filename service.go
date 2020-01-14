package apic

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/config"
	"github.com/tidwall/gjson"
)

type actionType int

const (
	addAPI    actionType = iota
	updateAPI            = iota
	deleteAPI            = iota
)

type serviceExecution int

const (
	addAPIServerSpec serviceExecution = iota + 1
	addAPIServerRevisionSpec
	addAPIServerInstanceSpec
	addCatalog
)

//ServiceBody -
type ServiceBody struct {
	NameToPush       string `json:",omitempty"`
	APIName          string `json:",omitempty"`
	RestAPIID        string `json:",omitempty"`
	URL              string `json:",omitempty"`
	Stage            string `json:",omitempty"`
	TeamID           string `json:",omitempty"`
	Description      string `json:",omitempty"`
	Version          string `json:",omitempty"`
	AuthPolicy       string `json:",omitempty"`
	Swagger          []byte `json:",omitempty"`
	Documentation    []byte `json:",omitempty"`
	Tags             map[string]interface{}
	Buffer           []byte            `json:",omitempty"`
	AgentMode        corecfg.AgentMode `json:",omitempty"`
	ServiceExecution int               `json:"omitempty"`
}

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
	APIService string             `json:"apiService"`
	Definition RevisionDefinition `json:"definition"`
}

// RevisionDefinition -
type RevisionDefinition struct {
	Type  string `json:"type,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// APIServerInstanceSpec -
type APIServerInstanceSpec struct {
	APIServiceRevision string     `json:"apiServiceRevision,omitempty"`
	InstanceEndPoint   []EndPoint `json:"endpoint,omitempty"`
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

func (c *Client) deployService(serviceBody ServiceBody, method, url string) (string, error) {
	buffer, err := c.CreateService(serviceBody)
	if err != nil {
		log.Error("Error creating service item: ", err)
		return "", err
	}

	return c.DeployAPI(method, url, buffer)
}

// AddToAPICServer -
func (c *Client) AddToAPICServer(serviceBody ServiceBody) (string, error) {

	itemID := ""

	// Verify if the api already exists
	if c.IsNewAPI(serviceBody) {
		// add api
		serviceBody.ServiceExecution = int(addAPIServerSpec)
		itemID, err := c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesURL())
		if err != nil {
			log.Errorf("Error adding API %v, stage %v", serviceBody.APIName, serviceBody.Stage)
			return itemID, err
		}
	}

	// add api revision
	serviceBody.ServiceExecution = int(addAPIServerRevisionSpec)
	itemID, err := c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesRevisionsURL())
	if err != nil {
		log.Errorf("Error adding API revision %v, stage %v", serviceBody.APIName, serviceBody.Stage)
	}

	// add api instance
	serviceBody.ServiceExecution = int(addAPIServerInstanceSpec)
	itemID, err = c.deployService(serviceBody, http.MethodPost, c.cfg.GetAPIServerServicesInstancesURL())
	if err != nil {
		log.Errorf("Error adding API %v, stage %v", serviceBody.APIName, serviceBody.Stage)
	}

	return itemID, err
}

// AddToAPIC -
func (c *Client) AddToAPIC(serviceBody ServiceBody) (string, error) {
	return c.deployService(serviceBody, http.MethodPost, c.cfg.GetCatalogItemsURL())
}

// UpdateToAPIC -
func (c *Client) UpdateToAPIC(serviceBody ServiceBody, url string) (string, error) {
	return c.deployService(serviceBody, http.MethodPut, url)
}

// CreateService -
func (c *Client) CreateService(serviceBody ServiceBody) ([]byte, error) {
	if !isValidAuthPolicy(serviceBody.AuthPolicy) {
		return nil, fmt.Errorf("Unsuppored security policy '%v'. ", serviceBody.AuthPolicy)
	}
	if serviceBody.AgentMode == corecfg.Connected {
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

	// Add attributes from config
	attribsToPublish := config.GetConfig().AWSConfig.GetAttributesToPublish()
	attribsToPublishArray := strings.Split(attribsToPublish, ",")
	for _, attrib := range attribsToPublishArray {
		s := strings.Split(strings.TrimSpace(attrib), "=")
		left, right := s[0], s[1]
		attributes[left] = right
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
			APIService: strings.ToLower(serviceBody.APIName),
			Definition: revisionDefinition,
		}
	case int(addAPIServerInstanceSpec):
		endPoints := []EndPoint{}
		name = strings.ToLower(serviceBody.APIName) + strings.ToLower(serviceBody.Stage)
		host := gjson.Get(string(serviceBody.Swagger), "host").String()

		// Iterate through protocols and create endpoints for intances
		protocols := gjson.Get(string(serviceBody.Swagger), "schemes")
		schemes := make([]string, 0)
		json.Unmarshal([]byte(protocols.Raw), &schemes)
		for _, protocol := range schemes {
			endPoint := EndPoint{
				Host:     host,
				Port:     443, // TODO : this is a hard coded value as of now.  Port is not showing up in swagger at the time of check in
				Protocol: protocol,
			}
			endPoints = append(endPoints, endPoint)
		}

		spec = APIServerInstanceSpec{
			APIServiceRevision: name,
			InstanceEndPoint:   endPoints,
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
