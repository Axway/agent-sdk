package apic

import (
	"encoding/json"
	"net/http"

	coreapi "github.com/Axway/agent-sdk/pkg/api"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	corecfg "github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

// SubscriptionSchema -
type SubscriptionSchema interface {
	AddProperty(name, dataType, description, apicRefField string, isRequired bool, enums []string)
	GetProperty(name string) *SubscriptionSchemaPropertyDefinition
	AddUniqueKey(keyName string)
	GetSubscriptionName() string
	mapStringInterface() (map[string]interface{}, error)
	rawJSON() (json.RawMessage, error)
}

// SubscriptionSchemaPropertyDefinition -
type SubscriptionSchemaPropertyDefinition struct {
	Type          string   `json:"type"`
	Description   string   `json:"description"`
	Enum          []string `json:"enum,omitempty"`
	ReadOnly      bool     `json:"readOnly,omitempty"`
	Format        string   `json:"format,omitempty"`
	APICRef       string   `json:"x-axway-ref-apic,omitempty"`
	Name          string   `json:"-"`
	Required      bool     `json:"-"`
	SortEnums     bool     `json:"-"`
	FirstEnumItem string   `json:"-"`
}

type subscriptionSchema struct {
	SubscriptionName  string                                          `json:"-"`
	SchemaType        string                                          `json:"type"`
	SchemaVersion     string                                          `json:"$schema"`
	SchemaDescription string                                          `json:"description"`
	Properties        map[string]SubscriptionSchemaPropertyDefinition `json:"properties"`
	Required          []string                                        `json:"required,omitempty"`
	UniqueKeys        []string                                        `json:"x-axway-unique-keys,omitempty"`
}

// NewSubscriptionSchema -
func NewSubscriptionSchema(name string) SubscriptionSchema {
	return &subscriptionSchema{
		SubscriptionName:  name,
		SchemaType:        "object",
		SchemaVersion:     "http://json-schema.org/draft-04/schema#",
		SchemaDescription: "Subscription specification for authentication",
		Properties:        make(map[string]SubscriptionSchemaPropertyDefinition),
		Required:          make([]string, 0),
	}
}

// AddProperty -
func (ss *subscriptionSchema) AddProperty(name, dataType, description, apicRefField string, isRequired bool, enums []string) {
	newProp := SubscriptionSchemaPropertyDefinition{
		Type:        dataType,
		Description: description,
		APICRef:     apicRefField,
	}

	if len(enums) > 0 {
		newProp.Enum = enums
	}
	ss.Properties[name] = newProp

	// required slice can't contain duplicates!
	if isRequired && !util.StringSliceContains(ss.Required, name) {
		ss.Required = append(ss.Required, name)
	}
}

// GetProperty -
func (ss *subscriptionSchema) GetProperty(name string) *SubscriptionSchemaPropertyDefinition {
	if val, ok := ss.Properties[name]; ok {
		return &val
	}
	return nil
}

// GetSubscriptionName -
func (ss *subscriptionSchema) GetSubscriptionName() string {
	return ss.SubscriptionName
}

// AddUniqueKey -
func (ss *subscriptionSchema) AddUniqueKey(keyName string) {
	ss.UniqueKeys = append(ss.UniqueKeys, keyName)
}

// rawJSON -
func (ss *subscriptionSchema) rawJSON() (json.RawMessage, error) {
	schemaBuffer, err := json.Marshal(ss)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(schemaBuffer), nil
}

// mapStringInterface -
func (ss *subscriptionSchema) mapStringInterface() (map[string]interface{}, error) {
	schemaBuffer, err := json.Marshal(ss)
	if err != nil {
		return nil, err
	}

	var stringMap map[string]interface{}
	json.Unmarshal(schemaBuffer, &stringMap)
	if err != nil {
		return nil, err
	}

	return stringMap, nil
}

// RegisterSubscriptionSchema - Adds a new subscription schema for the specified auth type. In publishToEnvironment mode
// creates a API Server resource for subscription definition
func (c *ServiceClient) RegisterSubscriptionSchema(subscriptionSchema SubscriptionSchema, update bool) error {
	c.RegisteredSubscriptionSchema = subscriptionSchema

	//Add API Server resource - SubscriptionDefinition
	buffer, err := c.marshalSubscriptionDefinition(subscriptionSchema)

	headers, err := c.createHeader()
	if err != nil {
		return err
	}

	request := coreapi.Request{
		Method:  coreapi.POST,
		URL:     c.cfg.GetAPIServerSubscriptionDefinitionURL(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return agenterrors.Wrap(ErrSubscriptionSchemaCreate, err.Error())
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		readResponseErrors(response.Code, response.Body)
		return agenterrors.Wrap(ErrSubscriptionSchemaResp, coreapi.POST).FormatError(response.Code)
	}
	if response.Code == http.StatusConflict && update {
		// Call update if a conflict was returned
		return c.UpdateSubscriptionSchema(subscriptionSchema)
	}

	return nil
}

// UpdateSubscriptionSchema - Updates a subscription schema in Publish to environment mode
// creates a API Server resource for subscription definition
func (c *ServiceClient) UpdateSubscriptionSchema(subscriptionSchema SubscriptionSchema) error {
	c.RegisteredSubscriptionSchema = subscriptionSchema

	// Add API Server resource - SubscriptionDefinition
	buffer, err := c.marshalSubscriptionDefinition(subscriptionSchema)

	headers, err := c.createHeader()
	if err != nil {
		return err
	}
	request := coreapi.Request{
		Method:  coreapi.PUT,
		URL:     c.cfg.GetAPIServerSubscriptionDefinitionURL() + "/" + subscriptionSchema.GetSubscriptionName(),
		Headers: headers,
		Body:    buffer,
	}

	response, err := c.apiClient.Send(request)
	if err != nil {
		return agenterrors.Wrap(ErrSubscriptionSchemaCreate, err.Error())
	}
	if !(response.Code == http.StatusOK) {
		readResponseErrors(response.Code, response.Body)
		return agenterrors.Wrap(ErrSubscriptionSchemaResp, coreapi.PUT).FormatError(response.Code)
	}

	return nil
}

func (c *ServiceClient) marshalSubscriptionDefinition(subscriptionSchema SubscriptionSchema) ([]byte, error) {
	catalogSubscriptionSchema, err := subscriptionSchema.mapStringInterface()
	if err != nil {
		return nil, err
	}

	webhooks := make([]string, 0)
	if c.cfg.GetSubscriptionConfig().GetSubscriptionApprovalMode() == corecfg.WebhookApproval {
		webhooks = append(webhooks, DefaultSubscriptionWebhookName)
	}
	spec := v1alpha1.ConsumerSubscriptionDefinitionSpec{
		Webhooks: webhooks,
		Schema: v1alpha1.ConsumerSubscriptionDefinitionSpecSchema{
			Properties: []v1alpha1.ConsumerSubscriptionDefinitionSpecSchemaProperties{
				{
					Key:   profileKey,
					Value: catalogSubscriptionSchema,
				},
			},
		},
	}

	apiServerService := v1alpha1.ConsumerSubscriptionDefinition{
		ResourceMeta: v1.ResourceMeta{
			GroupVersionKind: v1alpha1.ConsumerSubscriptionDefinitionGVK(),
			Name:             subscriptionSchema.GetSubscriptionName(),
			Title:            "Subscription definition created by agent",
			Attributes:       nil,
			Tags:             nil,
		},
		Spec: spec,
	}

	return json.Marshal(apiServerService)
}

func (c *ServiceClient) getProfilePropValue(subscriptionDef *v1alpha1.ConsumerSubscriptionDefinition) map[string]interface{} {
	for _, prop := range subscriptionDef.Spec.Schema.Properties {
		if prop.Key == profileKey {
			return prop.Value
		}
	}
	return nil
}
