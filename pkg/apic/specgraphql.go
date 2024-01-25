package apic

type graphQLProcessor struct {
	spec []byte
}

func newGraphQLSpecProcessor(resourceSpec []byte) SpecProcessor {
	return &graphQLProcessor{spec: resourceSpec}
}

func (p *graphQLProcessor) GetResourceType() string {
	return GraphQL
}

func (p *graphQLProcessor) GetVersion() string {
	return ""
}

func (p *graphQLProcessor) GetDescription() string {
	return ""
}

func (p *graphQLProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}

func (p *graphQLProcessor) GetSpecBytes() []byte {
	return p.spec
}
