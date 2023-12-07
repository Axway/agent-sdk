package apic

type unstructuredProcessor struct {
	spec []byte
}

func newUnstructuredSpecProcessor(resourceSpec []byte) SpecProcessor {
	return &unstructuredProcessor{spec: resourceSpec}
}

func (p *unstructuredProcessor) GetResourceType() string {
	return Unstructured
}

// GetVersion -
func (p *unstructuredProcessor) GetVersion() string {
	return ""
}

// GetDescription -
func (p *unstructuredProcessor) GetDescription() string {
	return ""
}

// GetEndpoints -
func (p *unstructuredProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}

// GetSpecBytes -
func (p *unstructuredProcessor) GetSpecBytes() []byte {
	return p.spec
}
