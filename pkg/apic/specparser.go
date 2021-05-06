package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/wsdl"
	"github.com/emicklei/proto"
	"gopkg.in/yaml.v2"
)

type SpecProcessor interface {
	getEndpoints() ([]EndpointDefinition, error)
	getResourceType() string
}

type specResourceParser struct {
	resourceSpecType string
	resourceSpec     []byte
	SpecProcessor    SpecProcessor
}

func newSpecResourceParser(resourceSpec []byte, resourceSpecType string) specResourceParser {
	return specResourceParser{resourceSpec: resourceSpec, resourceSpecType: resourceSpecType}
}

func (s *specResourceParser) parse() error {
	if s.resourceSpecType == "" {
		s.discoverSpecTypeAndCreateProcessor()
	} else {
		err := s.createProcessorWithResourceType()
		if err != nil {
			return err
		}
	}

	if s.SpecProcessor == nil {
		s.SpecProcessor = newUnstructuredSpecProcessor(s.resourceSpec)
	}
	return nil
}

func (s *specResourceParser) discoverSpecTypeAndCreateProcessor() {
	s.SpecProcessor, _ = s.discoverYAMLAndJSONSpec()
	if s.SpecProcessor == nil {
		s.SpecProcessor, _ = s.parseWSDLSpec()
	}
	if s.SpecProcessor == nil {
		s.SpecProcessor, _ = s.parseProtobufSpec()
	}
}

func (s *specResourceParser) createProcessorWithResourceType() error {
	var err error
	switch s.resourceSpecType {
	case Wsdl:
		s.SpecProcessor, err = s.parseWSDLSpec()
	case Oas2:
		s.SpecProcessor, err = s.parseOAS2Spec()
	case Oas3:
		s.SpecProcessor, err = s.parseOAS3Spec()
	case Protobuf:
		s.SpecProcessor, err = s.parseProtobufSpec()
	case AsyncAPI:
		s.SpecProcessor, err = s.parseAsyncAPISpec()
	}
	return err
}

func (s *specResourceParser) getSpecProcessor() SpecProcessor {
	return s.SpecProcessor
}

func (s *specResourceParser) discoverYAMLAndJSONSpec() (SpecProcessor, error) {
	specDef := make(map[string]interface{})
	// lowercase the byte array to ensure keys we care about are parsed
	err := yaml.Unmarshal(s.resourceSpec, &specDef)
	if err != nil {
		err := json.Unmarshal(s.resourceSpec, &specDef)
		if err != nil {
			return nil, err
		}
	}

	specType, ok := specDef["openapi"]
	if ok {
		openapi := specType.(string)
		if strings.HasPrefix(openapi, "3.") {
			return s.parseOAS3Spec()
		}
		if strings.HasPrefix(openapi, "2.") {
			return s.parseOAS2Spec()
		}
		return nil, errors.New("Invalid openapi specification")
	}

	specType, ok = specDef["swagger"]
	if ok {
		swagger := specType.(string)
		if swagger == "2.0" {
			return s.parseOAS2Spec()
		}
		return nil, errors.New("Invalid swagger 2.0 specification")
	}

	specType, ok = specDef["asyncapi"]
	if ok {
		return newAsyncAPIProcessor(specDef), nil
	}
	return nil, errors.New("Unknown yaml or json based specification")
}

func (s *specResourceParser) parseWSDLSpec() (SpecProcessor, error) {
	def, err := wsdl.Unmarshal(s.resourceSpec)
	if err != nil {
		return nil, err
	}
	return newWsdlProcessor(def), nil
}

func (s *specResourceParser) parseOAS2Spec() (SpecProcessor, error) {
	return NewOas2Processor(s.resourceSpec)
}

func (s *specResourceParser) parseOAS3Spec() (SpecProcessor, error) {
	return NewOas3Processor(s.resourceSpec)
}

func (s *specResourceParser) parseAsyncAPISpec() (SpecProcessor, error) {
	specDef := make(map[string]interface{})
	// lowercase the byte array to ensure keys we care about are parsed
	err := yaml.Unmarshal(s.resourceSpec, &specDef)
	if err != nil {
		err := json.Unmarshal(s.resourceSpec, &specDef)
		if err != nil {
			return nil, err
		}
	}
	_, ok := specDef["asyncapi"]
	if ok {
		return newAsyncAPIProcessor(specDef), nil
	}
	return nil, errors.New("Invalid asyncapi specification")
}

func (s *specResourceParser) parseProtobufSpec() (SpecProcessor, error) {
	reader := bytes.NewReader(s.resourceSpec)
	parser := proto.NewParser(reader)
	definition, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	if len(definition.Elements) > 0 {
		return newProtobufProcessor(definition), nil
	}
	return nil, errors.New("Invalid protobuf specification")

}
