package apic

type a2aProcessor struct {
	spec []byte
}

func newA2ASpecProcessor(resourceSpec []byte) SpecProcessor {
	return &a2aProcessor{spec: resourceSpec}
}

func (p *a2aProcessor) GetResourceType() string {
	return A2a
}

func (p *a2aProcessor) GetVersion() string {
	return ""
}

func (p *a2aProcessor) GetDescription() string {
	return ""
}

func (p *a2aProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}

func (p *a2aProcessor) GetSpecBytes() []byte {
	return p.spec
}
