package apic

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type ramlProcessor struct {
	ramlDef map[string]interface{}
	spec    []byte
}

func newRamlProcessor(ramlDef map[string]interface{}, spec []byte) *ramlProcessor {
	return &ramlProcessor{ramlDef: ramlDef, spec: spec}
}

func (p *ramlProcessor) GetResourceType() string {
	return Raml
}

// GetVersion -
func (p *ramlProcessor) GetVersion() string {
	if version := p.ramlDef["version"]; version != nil {
		return version.(string)
	}
	return ""
}

// GetDescription -
func (p *ramlProcessor) GetDescription() string {
	if description := p.ramlDef["info"]; description != nil {
		return description.(string)
	}
	return ""
}

// GetEndPoints -
func (p *ramlProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	baseUri := p.ramlDef["baseUri"]
	if baseUri == nil {
		return nil, fmt.Errorf("No baseUri provided")
	}

	if params := p.ramlDef["baseUriParameters"]; params != nil {
		return nil, fmt.Errorf("Not implemented error")
	}

	return uriToEndpoints(baseUri.(string), p.getProtocols())
}

func (p *ramlProcessor) getProtocols() []string {
	if protocols := p.ramlDef["protocols"]; protocols != nil {
		if validProtocols, ok := protocols.([]string); ok {
			return validProtocols
		}
	}
	return nil
}

func uriToEndpoints(uri string, protocols []string) ([]EndpointDefinition, error) {
	parseURL, err := url.Parse(uri)
	endpoint := EndpointDefinition{}
	endpoint.Host = parseURL.Hostname()
	port, _ := strconv.Atoi(parseURL.Port())
	endpoint.Port = int32(port)
	endpoint.BasePath = parseURL.Path
	endpoint.Protocol = "HTTP"
	if strings.HasPrefix(uri, "HTTPS") {
		endpoint.Protocol = "HTTPS"
	}

	if len(protocols) == 0 {
		return []EndpointDefinition{endpoint}, err
	}
	if len(protocols) == 1 {
		endpoint.Protocol = protocols[0]
		return []EndpointDefinition{endpoint}, err
	}

	endpoint.Protocol = "HTTP"
	endpointCpy := endpointCopy(endpoint)
	endpointCpy.Protocol = "HTTPS"
	return []EndpointDefinition{endpoint, endpointCpy}, err
}

func endpointCopy(e EndpointDefinition) EndpointDefinition {
	ed := &e
	return *ed
}

func (p *ramlProcessor) GetSpecBytes() []byte {
	return p.spec
}
