package apic

import (
	"github.com/emicklei/proto"
)

type protobufProcessor struct {
	protobufDef *proto.Proto
}

func newProtobufProcessor(protobufDef *proto.Proto) *protobufProcessor {
	return &protobufProcessor{protobufDef: protobufDef}
}

func (p *protobufProcessor) getResourceType() string {
	return Protobuf
}

// GetEndpoints -
func (p *protobufProcessor) GetEndpoints() ([]EndpointDefinition, error) {
	return []EndpointDefinition{}, nil
}
