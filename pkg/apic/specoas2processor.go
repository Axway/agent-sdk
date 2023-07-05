package apic

import (
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util"
	coreerrors "github.com/Axway/agent-sdk/pkg/util/errors"
)

var validOA2Schemes = map[string]bool{"http": true, "https": true, "ws": true, "wss": true}

const (
	oasSecurityAPIKey = "apiKey"
	oasSecurityOauth  = "oauth2"
	oasSecurityBasic  = "basic"
	oasSecurityHttp   = "http"
)

// oas2SpecProcessor parses and validates an OAS2 spec, and exposes methods to modify the content of the spec.
type oas2SpecProcessor struct {
	spec         *oas2Swagger
	scopes       map[string]string
	authPolicies []string
	apiKeyInfo   []APIKeyInfo
}

func newOas2Processor(oas2Spec *oas2Swagger) *oas2SpecProcessor {
	return &oas2SpecProcessor{spec: oas2Spec}
}

func (p *oas2SpecProcessor) getResourceType() string {
	return Oas2
}

// GetVersion -
func (p *oas2SpecProcessor) GetVersion() string {
	return p.spec.Info.Version
}

// GetEndpoints -
func (p *oas2SpecProcessor) GetEndpoints() ([]EndpointDefinition, error) {
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

func (p *oas2SpecProcessor) ParseAuthInfo() {
	authPolicies := []string{}
	keyInfo := []APIKeyInfo{}
	scopes := make(map[string]string)
	for _, scheme := range p.spec.SecurityDefinitions {
		switch scheme.Type {
		case oasSecurityBasic:
			authPolicies = append(authPolicies, Basic)
		case oasSecurityAPIKey:
			authPolicies = append(authPolicies, Apikey)
			keyInfo = append(keyInfo, APIKeyInfo{
				Location: scheme.In,
				Name:     scheme.Name,
			})
		case oasSecurityOauth:
			authPolicies = append(authPolicies, Oauth)
			for scope, val := range scheme.Scopes {
				scopes[strings.TrimSpace(scope)] = strings.TrimSpace(val)
			}
		}
	}
	p.authPolicies = util.RemoveDuplicateValuesFromStringSlice(authPolicies)
	sort.Strings(p.authPolicies)
	p.apiKeyInfo = keyInfo
	p.scopes = scopes
}

func (p *oas2SpecProcessor) GetAuthPolicies() []string {
	return p.authPolicies
}

func (p *oas2SpecProcessor) GetOAuthScopes() map[string]string {
	return p.scopes
}

func (p *oas2SpecProcessor) GetAPIKeyInfo() []APIKeyInfo {
	return p.apiKeyInfo
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
