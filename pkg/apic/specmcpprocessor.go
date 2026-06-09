package apic

type mcpProcessor struct {
	spec []byte
}

func newMCPSpecProcessor(resourceSpec []byte) SpecProcessor {
	return &mcpProcessor{spec: resourceSpec}
}

func (p *mcpProcessor) GetResourceType() string {
	return Mcp
}

func (p *mcpProcessor) GetVersion() string {
	return ""
}

func (p *mcpProcessor) GetDescription() string {
	return ""
}

func (p *mcpProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}

func (p *mcpProcessor) GetSpecBytes() []byte {
	return p.spec
}
