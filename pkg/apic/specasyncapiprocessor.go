package apic

import (
	"net/url"
	"strconv"
	"strings"
)

type asyncAPIProcessor struct {
	asyncapiDef map[string]interface{}
}

func newAsyncAPIProcessor(asyncapiDef map[string]interface{}) *asyncAPIProcessor {
	return &asyncAPIProcessor{asyncapiDef: asyncapiDef}
}

func (p *asyncAPIProcessor) getResourceType() string {
	return AsyncAPI
}

// GetEndPoints -
func (p *asyncAPIProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	endpoints := make([]EndpointDefinition, 0)
	var err error
	servers := p.asyncapiDef["servers"]
	if servers != nil {
		if serverList, ok := servers.(map[string]interface{}); ok {
			endpoints, err = p.parseServerList(serverList)
			if err != nil {
				return nil, err
			}
		}
	}

	return endpoints, nil
}

func (p *asyncAPIProcessor) parseServerList(serverList map[string]interface{}) ([]EndpointDefinition, error) {
	endpoints := make([]EndpointDefinition, 0)
	for _, value := range serverList {
		serverObjInterface, ok := value.(map[string]interface{})
		if ok {
			endpoint, err := p.parseServerObject(serverObjInterface)
			if err != nil {
				return nil, err
			}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, nil
}

func (p *asyncAPIProcessor) parseServerObject(serverObjInterface map[string]interface{}) (EndpointDefinition, error) {
	var err error
	protocol := ""
	serverURL := ""
	var serverVariables map[string]string
	serverDetails := map[string]interface{}{}
	for key, valueInterface := range serverObjInterface {
		value, ok := valueInterface.(string)
		if ok {
			if key == "protocol" {
				protocol = value
			}
			if key == "url" {
				serverURL = value
			}
		}
		if key == "variables" {
			variablesInterface, ok := valueInterface.(map[string]interface{})
			if ok {
				serverVariables, _ = p.parseVariables(variablesInterface)
			}
		}
		if key == "bindings" {
			serverDetails = valueInterface.(map[string]interface{})
		}
	}
	endpoint := EndpointDefinition{}
	endpoint.Protocol = protocol
	// variable substitution
	for varName, varValue := range serverVariables {
		serverURL = strings.ReplaceAll(serverURL, "{"+varName+"}", varValue)
	}

	parseURL, err := url.Parse(protocol + "://" + serverURL)
	endpoint.Host = parseURL.Hostname()
	port, _ := strconv.Atoi(parseURL.Port())
	endpoint.Port = int32(port)
	endpoint.BasePath = parseURL.Path
	endpoint.Details = serverDetails
	return endpoint, err
}

func (p *asyncAPIProcessor) parseVariables(variablesObjInterface map[string]interface{}) (map[string]string, error) {
	serverVars := make(map[string]string)
	for varName, varObject := range variablesObjInterface {
		varObjectInterface, ok := varObject.(map[string]interface{})
		if ok {
			varValue := p.parseVariableObject(varObjectInterface)
			serverVars[varName] = varValue
		}
	}
	return serverVars, nil
}

func (p *asyncAPIProcessor) parseVariableObject(serverObjInterface map[string]interface{}) string {
	varDefaultValue := ""
	for key, valueInterface := range serverObjInterface {
		if key == "default" {
			value, ok := valueInterface.(string)
			if ok {
				varDefaultValue = value
			}
		}
	}
	return varDefaultValue
}
