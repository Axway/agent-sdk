package apic

import (
	"encoding/json"
	"os"
	"strings"
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
	Tags               map[string]interface{}  `json:"tags,omitempty"`
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
	Description string                 `json:"description,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
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

const (
	subscriptionSchema = "{\"type\": \"object\", \"$schema\": \"http://json-schema.org/draft-04/schema#\", \"description\": \"Subscription specification for API Key authentication\", \"x-axway-unique-keys\": \"APIC_APPLICATION_ID\", \"properties\": {\"applicationId\": {\"type\": \"string\", \"description\": \"Select an application\", \"x-axway-ref-apic\": \"APIC_APPLICATION_ID\"}}, \"required\":[\"applicationId\"]}"
)

// CreateCatalogItemBodyForAdd -
func (c *Client) CreateCatalogItemBodyForAdd(bodyForAdd CatalogItemBodyAddParam) ([]byte, error) {
	newCatalogItem := CatalogItemInit{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               bodyForAdd.NameToPush,
		OwningTeamID:       bodyForAdd.TeamID,
		Description:        bodyForAdd.Description,
		Properties: []CatalogProperty{
			{
				Key: "accessInfo",
				Value: CatalogPropertyValue{
					AuthPolicy: bodyForAdd.AuthPolicy,
					URL:        bodyForAdd.URL,
				},
			},
		},
		Tags:       bodyForAdd.Tags,
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
			Version: bodyForAdd.Version,
			State:   "PUBLISHED",
			Properties: []CatalogRevisionProperty{
				{
					Key:   "documentation",
					Value: json.RawMessage(string(bodyForAdd.Documentation)),
				},
				{
					Key:   "swagger",
					Value: json.RawMessage(bodyForAdd.Swagger),
				},
			},
		},
	}

	return json.Marshal(newCatalogItem)
}

// CreateCatalogItemBodyForUpdate -
func (c *Client) CreateCatalogItemBodyForUpdate(bodyForUpdate CatalogItemBodyUpdateParam) ([]byte, error) {
	newCatalogItem := CatalogItem{
		DefinitionType:     "API",
		DefinitionSubType:  "swaggerv2",
		DefinitionRevision: 1,
		Name:               bodyForUpdate.NameToPush,
		OwningTeamID:       bodyForUpdate.TeamID,
		Description:        bodyForUpdate.Description,
		Tags:               bodyForUpdate.Tags,
		Visibility:         "RESTRICTED",  // default value
		State:              "UNPUBLISHED", //default
		LatestVersionDetails: CatalogItemRevision{
			Version: bodyForUpdate.Version,
			State:   "PUBLISHED",
		},
	}

	return json.Marshal(newCatalogItem)
}

// AddCatalogItem -
func (c *Client) AddCatalogItem(addCatalogItem AddCatalogItemParam) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	return c.DeployAPI("POST", addCatalogItem.Buffer, addCatalogItem.AgentMode, addCatalogItem.URL)

}

// UpdateCatalogItem -
func (c *Client) UpdateCatalogItem(updateCatalogItem UpdateCatalogItemParam) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "", nil
	}

	return c.DeployAPI("PUT", updateCatalogItem.Buffer, updateCatalogItem.AgentMode, updateCatalogItem.URL)

}

// AddCatalogItemImage -
func (c *Client) AddCatalogItemImage(addCatalogImage AddCatalogItemImageParam) (string, error) {
	if addCatalogImage.Image != "" {
		catalogImage := CatalogItemImage{
			DataType:      addCatalogImage.ImageContentType,
			Base64Content: addCatalogImage.Image,
		}
		catalogItemImageBuffer, _ := json.Marshal(catalogImage)

		//TODO for Dale.  This needs to change and be set in the agent of v7
		url := c.cfg.GetCatalogItemImage(addCatalogImage.CatalogItemID)
		return c.DeployAPI("POST", catalogItemImageBuffer, addCatalogImage.AgentMode, url)
	}
	return "", nil
}

func isUnitTesting() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}
