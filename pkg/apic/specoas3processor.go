package apic

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/getkin/kin-openapi/openapi3"
)

type oas3SpecProcessor struct {
	spec *openapi3.Swagger
}

func newOas3Processor(oas3Obj *openapi3.Swagger) *oas3SpecProcessor {
	return &oas3SpecProcessor{spec: oas3Obj}
}

func (p *oas3SpecProcessor) getResourceType() string {
	return Oas3
}

func (p *oas3SpecProcessor) getEndpoints() ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	for _, server := range p.spec.Servers {
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
		if serverVar.Default == nil {
			err := fmt.Errorf("Server variable in OAS3 %s does not have a default value, spec not valid", serverKey)
			log.Errorf(err.Error())
			return "", nil, err
		}
		defaultURL = strings.ReplaceAll(defaultURL, fmt.Sprintf("{%s}", serverKey), serverVar.Default.(string))
		if len(serverVar.Enum) == 0 {
			newURLs = p.processURLSubstutions(allURLs, newURLs, serverKey, serverVar.Default.(string))
		} else {
			for _, enumVal := range serverVar.Enum {
				newURLs = p.processURLSubstutions(allURLs, newURLs, serverKey, enumVal.(string))
			}
		}
		allURLs = newURLs
	}

	return defaultURL, allURLs, nil
}

func (p *oas3SpecProcessor) processURLSubstutions(allURLs, newURLs []string, varName, varValue string) []string {
	for _, template := range allURLs {
		newURLs = append(newURLs, strings.ReplaceAll(template, fmt.Sprintf("{%s}", varName), varValue))
	}
	return newURLs
}

func (p *oas3SpecProcessor) parseURLsIntoEndpoints(defaultURL string, allURLs []string) ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	for _, urlStr := range allURLs {
		urlObj, err := url.Parse(urlStr)
		if err != nil {
			err := fmt.Errorf("Could not parse url: %s", urlStr)
			log.Errorf(err.Error())
			return nil, err
		}
		// If a port is not given, use lookup the default
		var port int
		if urlObj.Port() == "" {
			port, _ = net.LookupPort("tcp", urlObj.Scheme)
		} else {
			port, _ = strconv.Atoi(urlObj.Port())
		}

		endPoint := EndpointDefinition{
			Host:     urlObj.Hostname(),
			Port:     int32(port),
			Protocol: urlObj.Scheme,
			BasePath: urlObj.Path,
		}

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
