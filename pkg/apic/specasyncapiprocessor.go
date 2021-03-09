package apic

import (
	"net/url"
	"strconv"
	"strings"
)

type asyncAPIProcessor struct {
	asyncapiDef map[string]interface{}
	spec        []byte
}

func newAsyncAPIProcessor(asyncapiDef map[string]interface{}) *asyncAPIProcessor {
	return &asyncAPIProcessor{asyncapiDef: asyncapiDef}
}

func (p *asyncAPIProcessor) getResourceType() string {
	return AsyncAPI
}

func (p *asyncAPIProcessor) getEndpoints() ([]EndpointDefinition, error) {
	endpoints := make([]EndpointDefinition, 0)
	var err error
	servers := p.asyncapiDef["servers"]
	if servers != nil {
		if serverList, ok := servers.(map[interface{}]interface{}); ok {
			endpoints, err = p.parseServerList(serverList)
			if err != nil {
				return nil, err
			}
		}
	}

	return endpoints, nil
}

func (p *asyncAPIProcessor) parseServerList(serverList map[interface{}]interface{}) ([]EndpointDefinition, error) {
	endpoints := make([]EndpointDefinition, 0)
	for _, value := range serverList {
		serverObjInterface, ok := value.(map[interface{}]interface{})
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

func (p *asyncAPIProcessor) parseServerObject(serverObjInterface map[interface{}]interface{}) (EndpointDefinition, error) {
	var err error
	protocol := ""
	serverURL := ""
	var serverVariables map[string]string
	for keyInterface, valueInterface := range serverObjInterface {
		key := keyInterface.(string)
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
			variablesInterface, ok := valueInterface.(map[interface{}]interface{})
			if ok {
				serverVariables, _ = p.parseVariables(variablesInterface)
			}
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
	return endpoint, err
}

func (p *asyncAPIProcessor) parseVariables(variablesObjInterface map[interface{}]interface{}) (map[string]string, error) {
	serverVars := make(map[string]string)
	for varNameInt, varObject := range variablesObjInterface {
		varName := varNameInt.(string)
		varObjectInterface, ok := varObject.(map[interface{}]interface{})
		if ok {
			varValue := p.parseVariableObject(varObjectInterface)
			serverVars[varName] = varValue
		}
	}
	return serverVars, nil
}

func (p *asyncAPIProcessor) parseVariableObject(serverObjInterface map[interface{}]interface{}) string {
	varDefaultValue := ""
	for keyInterface, valueInterface := range serverObjInterface {
		key := keyInterface.(string)
		if key == "default" {
			value, ok := valueInterface.(string)
			if ok {
				varDefaultValue = value
			}
		}
	}
	return varDefaultValue
}
