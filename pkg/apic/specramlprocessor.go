package apic

import (
	"fmt"
	"net"
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

func (p *ramlProcessor) GetVersion() string {
	if version := p.ramlDef["version"]; version != nil {
		switch v := version.(type) {
		// yaml Unmarshalling converts the underlying interface values which can be floats like 2.0 to 2
		case float64:
			if float64(int(v)) == v {
				return fmt.Sprintf("%v.0", v)
			}
			return fmt.Sprintf("%v", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func (p *ramlProcessor) GetDescription() string {
	if description := p.ramlDef["description"]; description != nil {
		return fmt.Sprintf("%v", description)
	}
	return ""
}

func (p *ramlProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	baseUri := p.ramlDef["baseUri"]
	if baseUri == nil {
		return nil, fmt.Errorf("no baseUri provided")
	}

	if params := p.ramlDef["baseUriParameters"]; params != nil {
		return nil, fmt.Errorf("not implemented error")
	}

	return p.uriToEndpoints(baseUri.(string), p.getProtocols())
}

func (p *ramlProcessor) GetSpecBytes() []byte {
	return p.spec
}

func (p *ramlProcessor) getProtocols() []string {
	if protocols := p.ramlDef["protocols"]; protocols != nil {
		// in case [HTTP, HTTPS] is provided
		if ramlProtocols, ok := protocols.([]interface{}); ok {
			return validateRamlProtocols(ramlProtocols)
		}
		// in case just HTTP is provided
		if ramlProtocols, ok := protocols.(string); ok {
			return validateRamlProtocols([]interface{}{ramlProtocols})
		}
	}
	return nil
}

func (p *ramlProcessor) uriToEndpoints(uri string, protocols []string) ([]EndpointDefinition, error) {
	parseURL, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.ToLower(parseURL.Scheme), "http") {
		return p.uriToEndpoints("https://"+uri, protocols)
	}
	endpoints := []EndpointDefinition{}
	ep := EndpointDefinition{}
	// currently accepting only version as a dynamic value to the endpoints
	if version := p.ramlDef["version"]; version != nil {
		parseURL.Path = strings.Replace(parseURL.Path, "{version}", fmt.Sprintf("%v", version), 1)
	}

	ep.Host = parseURL.Hostname()
	ep.BasePath = parseURL.Path
	ep.Protocol = parseURL.Scheme

	port, _ := strconv.Atoi(parseURL.Port())
	if port == 0 {
		port, _ = net.LookupPort("tcp", ep.Protocol)
	}
	ep.Port = int32(port)

	if len(protocols) == 0 {
		return append(endpoints, ep), err
		// Overrides the protocol from the URI, but does not override the port.
	} else if len(protocols) == 1 {
		ep.Protocol = strings.ToLower(protocols[0])
		if port == 0 {
			port, _ = net.LookupPort("tcp", ep.Protocol)
		}
		ep.Port = int32(port)
		return append(endpoints, ep), err
	}
	// With multiple protocols provided, ignores the port from the url.
	for i := range protocols {
		epCpy := endpointCopy(ep)
		port, _ = net.LookupPort("tcp", protocols[i])
		epCpy.Port = int32(port)
		epCpy.Protocol = strings.ToLower(protocols[i])
		endpoints = append(endpoints, epCpy)
	}

	return endpoints, err
}

func endpointCopy(e EndpointDefinition) EndpointDefinition {
	ed := &e
	return *ed
}

func validateRamlProtocols(protocols []interface{}) []string {
	stringProtocols := []string{}
	for i := range protocols {
		p, ok := protocols[i].(string)
		if !ok {
			return []string{}
		}
		if strings.ToUpper(p) != "HTTPS" && strings.ToUpper(p) != "HTTP" {
			return []string{}
		}
		stringProtocols = append(stringProtocols, p)
	}
	return stringProtocols
}
