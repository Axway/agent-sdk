package apic

import (
	"net"
	"strconv"
	"strings"

	coreerrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

var validOA2Schemes = map[string]bool{"http": true, "https": true, "ws": true, "wss": true}

// oas2SpecProcessor parses and validates an OAS2 spec, and exposes methods to modify the content of the spec.
type oas2SpecProcessor struct {
	spec *oas2Swagger
}

func newOas2Processor(oas2Spec *oas2Swagger) *oas2SpecProcessor {
	return &oas2SpecProcessor{spec: oas2Spec}
}

func (p *oas2SpecProcessor) getResourceType() string {
	return Oas2
}

func (p *oas2SpecProcessor) getEndpoints() ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	swaggerHostElements := strings.Split(p.spec.Host, ":")
	host := swaggerHostElements[0]
	port := 0
	if len(swaggerHostElements) > 1 {
		swaggerPort, err := strconv.Atoi(swaggerHostElements[1])
		if err == nil {
			port = swaggerPort
		}
	}

	if host == "" {
		return nil, coreerrors.Wrap(ErrSetSpecEndPoints, "no host defined in the specification")
	}

	// If schemes are specified create endpoint for each scheme
	if len(p.spec.Schemes) > 0 {
		for _, protocol := range p.spec.Schemes {
			if !validOA2Schemes[protocol] {
				return nil, coreerrors.Wrap(ErrSetSpecEndPoints, "invalid endpoint scheme defined in specification")
			}
			endPoint := createEndpointDefinition(protocol, host, port, p.spec.BasePath)
			endPoints = append(endPoints, endPoint)
		}
	}

	// If no schemes are specified create endpoint with default scheme
	if len(endPoints) == 0 {
		endPoint := createEndpointDefinition("https", host, port, p.spec.BasePath)
		endPoints = append(endPoints, endPoint)
	}
	return endPoints, nil
}

func createEndpointDefinition(scheme, host string, port int, basePath string) EndpointDefinition {
	path := "/"
	if basePath != "" {
		path = basePath
	}
	// If a port is not given, use lookup the default
	if port == 0 {
		port, _ = net.LookupPort("tcp", scheme)
	}
	return EndpointDefinition{
		Host:     host,
		Port:     int32(port),
		Protocol: scheme,
		BasePath: path,
	}
}
