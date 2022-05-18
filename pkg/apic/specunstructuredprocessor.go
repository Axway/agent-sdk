package apic

type unstructuredProcessor struct {
	spec []byte
}

func newUnstructuredSpecProcessor(resourceSpec []byte) SpecProcessor {
	return &unstructuredProcessor{spec: resourceSpec}
}

func (p *unstructuredProcessor) getResourceType() string {
	return Unstructured
}

// GetEndpoints -
func (p *unstructuredProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}
