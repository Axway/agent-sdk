package apic

import (
	"strconv"
	"strings"
)

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
	port := 443
	if len(swaggerHostElements) > 1 {
		swaggerPort, err := strconv.Atoi(swaggerHostElements[1])
		if err == nil {
			port = swaggerPort
		}
	}
	for _, protocol := range p.spec.Schemes {
		endPoint := EndpointDefinition{
			Host:     host,
			Port:     int32(port),
			Protocol: protocol,
			BasePath: p.spec.BasePath,
		}
		endPoints = append(endPoints, endPoint)
	}
	return endPoints, nil
}
