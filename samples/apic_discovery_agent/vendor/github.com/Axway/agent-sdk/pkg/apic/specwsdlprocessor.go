package apic

import (
	"net"
	"net/url"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/util/wsdl"
)

type wsdlProcessor struct {
	spec *wsdl.Definitions
}

func newWsdlProcessor(wsdlDef *wsdl.Definitions) *wsdlProcessor {
	return &wsdlProcessor{spec: wsdlDef}
}

func (p *wsdlProcessor) getResourceType() string {
	return Wsdl
}

func (p *wsdlProcessor) getEndpoints() ([]EndpointDefinition, error) {
	endPoints := []EndpointDefinition{}
	ports := p.spec.Service.Ports
	for _, val := range ports {
		loc := val.Address.Location
		fixed, err := url.Parse(loc)
		if err != nil {
			log.Errorf("Error parsing service location in WSDL to get endpoints: %v", err.Error())
			return nil, err
		}
		protocol := fixed.Scheme
		host := fixed.Hostname()
		portStr := fixed.Port()
		if portStr == "" {
			p, err := net.LookupPort("tcp", protocol)
			if err != nil {
				log.Errorf("Error finding port for endpoint: %v", err.Error())
				return nil, err
			}
			portStr = strconv.Itoa(p)
		}
		port, _ := strconv.Atoi(portStr)

		endPoint := EndpointDefinition{
			Host:     host,
			Port:     int32(port),
			Protocol: protocol,
			BasePath: fixed.Path,
		}
		if !p.contains(endPoints, endPoint) {
			endPoints = append(endPoints, endPoint)
		}
	}

	return endPoints, nil
}

func (p *wsdlProcessor) contains(endpts []EndpointDefinition, endpt EndpointDefinition) bool {
	for _, pt := range endpts {
		if pt == endpt {
			return true
		}
	}
	return false
}
