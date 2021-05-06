package apic

type unstructuredProcessor struct {
	spec []byte
}

func newUnstructuredSpecProcessor(resourceSpec []byte) specProcessor {
	return &unstructuredProcessor{spec: resourceSpec}
}

func (p *unstructuredProcessor) getResourceType() string {
	return Unstructured
}

func (p *unstructuredProcessor) getEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}
