package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util/oas"

	"github.com/Axway/agent-sdk/pkg/util/wsdl"
	"github.com/emicklei/proto"
	"gopkg.in/yaml.v2"
)

type specProcessor interface {
	getEndpoints() ([]EndpointDefinition, error)
	getResourceType() string
}

type oasSpecProcessor interface {
	getAuthInfo() ([]string, []APIKeyInfo)
}

type specResourceParser struct {
	resourceSpecType string
	resourceSpec     []byte
	specProcessor    specProcessor
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

	if s.specProcessor == nil {
		s.specProcessor = newUnstructuredSpecProcessor(s.resourceSpec)
	}
	return nil
}

func (s *specResourceParser) discoverSpecTypeAndCreateProcessor() {
	s.specProcessor, _ = s.discoverYAMLAndJSONSpec()
	if s.specProcessor == nil {
		s.specProcessor, _ = s.parseWSDLSpec()
	}
	if s.specProcessor == nil {
		s.specProcessor, _ = s.parseProtobufSpec()
	}
}

func (s *specResourceParser) createProcessorWithResourceType() error {
	var err error
	switch s.resourceSpecType {
	case Wsdl:
		s.specProcessor, err = s.parseWSDLSpec()
	case Oas2:
		s.specProcessor, err = s.parseOAS2Spec()
	case Oas3:
		s.specProcessor, err = s.parseOAS3Spec()
	case Protobuf:
		s.specProcessor, err = s.parseProtobufSpec()
	case AsyncAPI:
		s.specProcessor, err = s.parseAsyncAPISpec()
	}
	return err
}

func (s *specResourceParser) getSpecProcessor() specProcessor {
	return s.specProcessor
}

func (s *specResourceParser) discoverYAMLAndJSONSpec() (specProcessor, error) {
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

func (s *specResourceParser) parseWSDLSpec() (specProcessor, error) {
	def, err := wsdl.Unmarshal(s.resourceSpec)
	if err != nil {
		return nil, err
	}
	return newWsdlProcessor(def), nil
}

func (s *specResourceParser) parseOAS2Spec() (specProcessor, error) {
	swaggerObj := &oas2Swagger{}
	// lowercase the byte array to ensure keys we care about are parsed

	err := yaml.Unmarshal(s.resourceSpec, swaggerObj)
	if err != nil {
		err := json.Unmarshal(s.resourceSpec, swaggerObj)
		if err != nil {
			return nil, err
		}
	}
	if swaggerObj.Info.Title == "" {
		return nil, errors.New("Invalid openapi 2.0 specification")
	}
	return newOas2Processor(swaggerObj), nil
}

func (s *specResourceParser) parseOAS3Spec() (specProcessor, error) {
	oas3Obj, err := oas.ParseOAS3(s.resourceSpec)
	if err != nil {
		return nil, err
	}
	return newOas3Processor(oas3Obj), nil
}

func (s *specResourceParser) parseAsyncAPISpec() (specProcessor, error) {
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

func (s *specResourceParser) parseProtobufSpec() (specProcessor, error) {
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
