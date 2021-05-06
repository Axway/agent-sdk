package apic

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	coreerrors "github.com/Axway/agent-sdk/pkg/util/errors"
	"gopkg.in/yaml.v2"
)

var validOA2Schemes = map[string]bool{"http": true, "https": true, "ws": true, "wss": true}

// Oas2SpecProcessor parses and validates an OAS2 spec, and exposes methods to modify the content of the spec.
type Oas2SpecProcessor struct {
	spec *oas2Swagger
}

// NewOas2Processor parses a spec into an Openapi2 object, and then creates an Oas2SpecProcessor.
func NewOas2Processor(spec []byte) (*Oas2SpecProcessor, error) {
	swaggerObj := &oas2Swagger{}
	// lowercase the byte array to ensure keys we care about are parsed
	err := yaml.Unmarshal(spec, swaggerObj)
	if err != nil {
		err := json.Unmarshal(spec, swaggerObj)
		if err != nil {
			return nil, err
		}
	}
	if swaggerObj.Info.Title == "" {
		return nil, errors.New("Invalid openapi 2.0 specification")
	}
	return &Oas2SpecProcessor{spec: swaggerObj}, nil
}

func (p *Oas2SpecProcessor) getResourceType() string {
	return Oas2
}

func (p *Oas2SpecProcessor) getEndpoints() ([]EndpointDefinition, error) {
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

// SetHostDetails Sets the Host, BasePath, and Schemes field on an OAS 2 spec from the provided endpointURL.
func (p *Oas2SpecProcessor) SetHostDetails(endpointURL string) error {
	endpoint, err := url.Parse(endpointURL)
	if err != nil {
		return err
	}

	basePath := ""
	if endpoint.Path == "" {
		basePath = endpoint.Path
	} else {
		basePath = "/"
	}

	host := endpoint.Host
	schemes := []string{endpoint.Scheme}
	p.spec.Host = host
	p.spec.BasePath = basePath
	p.spec.Schemes = schemes
	return nil
}

// Marshal Converts the Openapi2 struct back into bytes. Call this after modifying the content of the spec.
func (p *Oas2SpecProcessor) Marshal() ([]byte, error) {
	return json.Marshal(p.spec)
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
