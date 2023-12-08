package apic

import (
	"github.com/emicklei/proto"
)

type protobufProcessor struct {
	protobufDef *proto.Proto
	spec        []byte
}

func newProtobufProcessor(protobufDef *proto.Proto, spec []byte) *protobufProcessor {
	return &protobufProcessor{protobufDef: protobufDef, spec: spec}
}

func (p *protobufProcessor) GetResourceType() string {
	return Protobuf
}

// GetVersion -
func (p *protobufProcessor) GetVersion() string {
	return ""
}

// GetDescription -
func (p *protobufProcessor) GetDescription() string {
	return ""
}

// GetEndpoints -
func (p *protobufProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}

// GetSpecBytes -
func (p *protobufProcessor) GetSpecBytes() []byte {
	return p.spec
}
