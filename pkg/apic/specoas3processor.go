package apic

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	coreerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/getkin/kin-openapi/openapi3"
)

// oas3SpecProcessor parses and validates an OAS3 spec, and exposes methods to modify the content of the spec.
type oas3SpecProcessor struct {
	spec *openapi3.T
}

func newOas3Processor(oas3Obj *openapi3.T) *oas3SpecProcessor {
	return &oas3SpecProcessor{spec: oas3Obj}
}

func (p *oas3SpecProcessor) getResourceType() string {
	return Oas3
}

func (p *oas3SpecProcessor) getEndpoints() ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	if len(p.spec.Servers) > 0 {
		var err error
		endPoints, err = p.parseEndpoints(p.spec.Servers)
		if err != nil {
			return nil, coreerrors.Wrap(ErrSetSpecEndPoints, err.Error())
		}
		return endPoints, nil
	}
	if len(endPoints) == 0 {
		return nil, coreerrors.Wrap(ErrSetSpecEndPoints, "no server endpoints defined")
	}
	return endPoints, nil
}

func (p *oas3SpecProcessor) parseEndpoints(servers []*openapi3.Server) ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	for _, server := range servers {
		// Add the URL string to the array
		allURLs := []string{
			server.URL,
		}

		defaultURL := ""
		var err error
		if server.Variables != nil {
			defaultURL, allURLs, err = p.handleURLSubstitutions(server, allURLs)
			if err != nil {
				return nil, err
			}
		}

		parsedEndPoints, err := p.parseURLsIntoEndpoints(defaultURL, allURLs)
		if err != nil {
			return nil, err
		}
		endPoints = append(endPoints, parsedEndPoints...)
	}
	return endPoints, nil
}

func (p *oas3SpecProcessor) handleURLSubstitutions(server *openapi3.Server, allURLs []string) (string, []string, error) {
	defaultURL := server.URL
	// Handle substitutions
	for serverKey, serverVar := range server.Variables {
		newURLs := []string{}
		if serverVar.Default == "" {
			err := fmt.Errorf("Server variable in OAS3 %s does not have a default value, spec not valid", serverKey)
			log.Errorf(err.Error())
			return "", nil, err
		}
		defaultURL = strings.ReplaceAll(defaultURL, fmt.Sprintf("{%s}", serverKey), serverVar.Default)
		if len(serverVar.Enum) == 0 {
			newURLs = p.processURLSubstitutions(allURLs, newURLs, serverKey, serverVar.Default)
		} else {
			for _, enumVal := range serverVar.Enum {
				newURLs = p.processURLSubstitutions(allURLs, newURLs, serverKey, enumVal)
			}
		}
		allURLs = newURLs
	}

	return defaultURL, allURLs, nil
}

func (p *oas3SpecProcessor) processURLSubstitutions(allURLs, newURLs []string, varName, varValue string) []string {
	for _, template := range allURLs {
		newURLs = append(newURLs, strings.ReplaceAll(template, fmt.Sprintf("{%s}", varName), varValue))
	}
	return newURLs
}

func (p *oas3SpecProcessor) parseURL(urlStr string) (*url.URL, error) {
	urlObj, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	if urlObj.Scheme == "" {
		urlObj, err = p.parseURL("https://" + urlStr)
	}
	return urlObj, err
}

func (p *oas3SpecProcessor) parseURLsIntoEndpoints(defaultURL string, allURLs []string) ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	for _, urlStr := range allURLs {
		if urlStr == "" {
			return nil, fmt.Errorf("server definition cannot have empty url")
		}
		urlObj, err := p.parseURL(urlStr)
		if err != nil {
			return nil, err
		}
		if urlObj.Hostname() == "" {
			err = fmt.Errorf("could not parse url: %s", urlStr)
			return nil, err
		}
		port := 0
		if urlObj.Port() != "" {
			port, _ = strconv.Atoi(urlObj.Port())
		}
		endPoint := createEndpointDefinition(urlObj.Scheme, urlObj.Hostname(), port, urlObj.Path)
		// If the URL is the default URL put it at the front of the array
		if urlStr == defaultURL {
			newEndPoints := []EndpointDefinition{endPoint}
			for _, oldEndpoint := range endPoints {
				newEndPoints = append(newEndPoints, oldEndpoint)
			}
			endPoints = newEndPoints
		} else {
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

// SetServers replaces the servers array on the Openapi3 object.
func SetServers(hosts []string, spec *openapi3.T) {
	var oas3Servers []*openapi3.Server
	for _, s := range hosts {
		oas3Servers = append(oas3Servers, &openapi3.Server{
			URL: s,
		})
	}
	if len(oas3Servers) > 0 {
		spec.Servers = oas3Servers
	}
}

// ParseOAS3 converts a JSON or YAML spec into an OpenAPI3 object
func ParseOAS3(spec []byte) (*openapi3.T, error) {
	oas3Obj, err := openapi3.NewLoader().LoadFromData(spec)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(oas3Obj.OpenAPI, "3.") {
		return nil, fmt.Errorf(oasParseError("3", ("'openapi' key is invalid.")))
	}
	if oas3Obj.Paths == nil {
		return nil, fmt.Errorf(oasParseError("3", "'paths' key not found."))
	}
	if oas3Obj.Info == nil {
		return nil, fmt.Errorf(oasParseError("3", "'info' key not found."))
	}
	if oas3Obj.Info.Title == "" {
		return nil, fmt.Errorf(oasParseError("3", "'info.title' key not found."))
	}
	return oas3Obj, nil
}

func oasParseError(version string, msg string) string {
	return fmt.Sprintf("invalid openapi %s specification. %s", version, msg)
}
