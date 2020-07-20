package apic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	coreapi "git.ecd.axway.int/apigov/apic_agents_sdk/pkg/api"
	"git.ecd.axway.int/apigov/apic_agents_sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// SubscriptionSchema -
type SubscriptionSchema interface {
	AddProperty(name, dataType, description, apicRefField string, isRequired bool, enums []string)
	AddUniqueKey(keyName string)
	GetSubscriptionName() string
	mapStringInterface() (map[string]interface{}, error)
	rawJSON() (json.RawMessage, error)
}

type subscriptionSchemaPropertyDefinition struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	APICRef     string   `json:"x-axway-ref-apic,omitempty"`
}

type subscriptionSchema struct {
	SubscriptionName  string                                          `json:"-"`
	SchemaType        string                                          `json:"type"`
	SchemaVersion     string                                          `json:"$schema"`
	SchemaDescription string                                          `json:"description"`
	Properties        map[string]subscriptionSchemaPropertyDefinition `json:"properties"`
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
		Properties:        make(map[string]subscriptionSchemaPropertyDefinition),
		Required:          make([]string, 0),
	}
}

// AddProperty -
func (ss *subscriptionSchema) AddProperty(name, dataType, description, apicRefField string, isRequired bool, enums []string) {
	newProp := subscriptionSchemaPropertyDefinition{
		Type:        dataType,
		Description: description,
		APICRef:     apicRefField,
	}

	if len(enums) > 0 {
		newProp.Enum = enums
	}
	ss.Properties[name] = newProp

	if isRequired {
		ss.Required = append(ss.Required, name)
	}
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
	var stringMap map[string]interface{}

	schemaBuffer, err := json.Marshal(ss)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(schemaBuffer, &stringMap)
	if err != nil {
		return nil, err
	}

	return stringMap, nil
}

// RegisterSubscriptionSchema - Adds a new subscription schema for the specified auth type. In publishToEnvironment mode
// creates a API Server resource for subscription definition
func (c *ServiceClient) RegisterSubscriptionSchema(subscriptionSchema SubscriptionSchema) error {
	c.RegisteredSubscriptionSchema = subscriptionSchema

	// nothing to do if not environment mode
	if !c.cfg.IsPublishToEnvironmentMode() {
		return nil
	}

	// Add API Server resource - SubscriptionDefinition
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
		return err
	}
	if !(response.Code == http.StatusCreated || response.Code == http.StatusConflict) {
		logResponseErrors(response.Body)
		return errors.New(strconv.Itoa(response.Code))
	}
	if response.Code == http.StatusConflict {
		request = coreapi.Request{
			Method:  coreapi.PUT,
			URL:     c.cfg.GetAPIServerSubscriptionDefinitionURL() + "/" + subscriptionSchema.GetSubscriptionName(),
			Headers: headers,
			Body:    buffer,
		}

		response, err := c.apiClient.Send(request)
		if err != nil {
			return err
		}
		if !(response.Code == http.StatusOK) {
			logResponseErrors(response.Body)
			return errors.New(strconv.Itoa(response.Code))
		}
	}
	return nil
}

// UpdateSubscriptionSchema - Updates a subscription schema in Publish to environment mode
// creates a API Server resource for subscription definition
func (c *ServiceClient) UpdateSubscriptionSchema(subscriptionSchema SubscriptionSchema) error {
	c.RegisteredSubscriptionSchema = subscriptionSchema
	if c.cfg.IsPublishToEnvironmentMode() {
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
			return err
		}
		if !(response.Code == http.StatusOK) {
			logResponseErrors(response.Body)
			return errors.New(strconv.Itoa(response.Code))
		}
	}
	return nil
}

func (c *ServiceClient) marshalSubscriptionDefinition(subscriptionSchema SubscriptionSchema) ([]byte, error) {
	catalogSubscriptionSchema, err := subscriptionSchema.mapStringInterface()
	if err != nil {
		return nil, err
	}
	spec := v1alpha1.ConsumerSubscriptionDefinitionSpec{
		Schema: v1alpha1.ConsumerSubscriptionDefinitionSpecSchema{
			Properties: []v1alpha1.ConsumerSubscriptionDefinitionSpecSchemaProperties{
				{
					Key:   "profile",
					Value: catalogSubscriptionSchema,
				},
			},
		},
	}

	apiServerService := APIServer{
		Name:       subscriptionSchema.GetSubscriptionName(),
		Title:      "Subscription definition created by agent",
		Attributes: nil,
		Spec:       spec,
		Tags:       nil,
	}

	return json.Marshal(apiServerService)
}
