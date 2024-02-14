package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/oas"

	"github.com/Axway/agent-sdk/pkg/util/wsdl"
	"github.com/emicklei/proto"
	"gopkg.in/yaml.v3"
)

const (
	mimeApplicationJSON = "application/json"
	mimeApplicationYAML = "application/yaml"
)

// SpecProcessor -
type SpecProcessor interface {
	GetVersion() string
	GetEndpoints() ([]EndpointDefinition, error)
	GetDescription() string
	GetSpecBytes() []byte
	GetResourceType() string
}

// OasSpecProcessor -
type OasSpecProcessor interface {
	ParseAuthInfo()
	GetAPIKeyInfo() []APIKeyInfo
	GetOAuthScopes() map[string]string
	GetAuthPolicies() []string
	StripSpecAuth()
	GetTitle() string
	GetSecurityBuilder() SecurityBuilder
	AddSecuritySchemes(map[string]interface{})
}

// SpecResourceParser -
type SpecResourceParser struct {
	resourceSpecType    string
	resourceContentType string
	resourceSpec        []byte
	specProcessor       SpecProcessor
	specHash            uint64
}

// NewSpecResourceParser -
func NewSpecResourceParser(resourceSpec []byte, resourceSpecType string) SpecResourceParser {
	hash, _ := util.ComputeHash(resourceSpec)
	return SpecResourceParser{resourceSpec: resourceSpec, resourceSpecType: resourceSpecType, specHash: hash}
}

// Parse -
func (s *SpecResourceParser) Parse() error {
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

func (s *SpecResourceParser) getResourceContentType() string {
	return s.resourceContentType
}

func (s *SpecResourceParser) discoverSpecTypeAndCreateProcessor() {
	s.specProcessor, _ = s.discoverYAMLAndJSONSpec()
	if s.specProcessor == nil {
		s.specProcessor, _ = s.parseWSDLSpec()
	}
	if s.specProcessor == nil {
		s.specProcessor, _ = s.parseProtobufSpec()
	}
}

func (s *SpecResourceParser) createProcessorWithResourceType() error {
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
	case GraphQL:
		s.specProcessor, err = s.parseGraphQLSpec()
	}
	return err
}

// GetSpecProcessor -
func (s *SpecResourceParser) GetSpecProcessor() SpecProcessor {
	return s.specProcessor
}

func (s *SpecResourceParser) discoverYAMLAndJSONSpec() (SpecProcessor, error) {
	specDef := make(map[string]interface{})
	// lowercase the byte array to ensure keys we care about are parsed
	contentType := mimeApplicationJSON
	err := json.Unmarshal(s.resourceSpec, &specDef)
	if err != nil {
		contentType = mimeApplicationYAML
		err := yaml.Unmarshal(s.resourceSpec, &specDef)
		if err != nil {
			return nil, err
		}
	}

	specType, ok := specDef["openapi"]
	if ok {
		openapi := specType.(string)
		if strings.HasPrefix(openapi, "3.") {
			s.resourceContentType = contentType
			return s.parseOAS3Spec()
		}
		if strings.HasPrefix(openapi, "2.") {
			return s.parseOAS2Spec()
		}
		return nil, errors.New("invalid openapi specification")
	}

	specType, ok = specDef["swagger"]
	if ok {
		swagger := specType.(string)
		if swagger == "2.0" {
			return s.parseOAS2Spec()
		}
		return nil, errors.New("invalid swagger 2.0 specification")
	}

	_, ok = specDef["asyncapi"]
	if ok {
		s.resourceContentType = contentType
		return newAsyncAPIProcessor(specDef, s.resourceSpec), nil
	}
	return nil, errors.New("unknown yaml or json based specification")
}

func (s *SpecResourceParser) parseWSDLSpec() (SpecProcessor, error) {
	def, err := wsdl.Unmarshal(s.resourceSpec)
	if err != nil {
		return nil, err
	}
	return newWsdlProcessor(def, s.resourceSpec), nil
}

func (s *SpecResourceParser) parseGraphQLSpec() (SpecProcessor, error) {
	return newGraphQLSpecProcessor(s.resourceSpec), nil
}

func (s *SpecResourceParser) parseOAS2Spec() (SpecProcessor, error) {
	swaggerObj := &oas2Swagger{}
	// lowercase the byte array to ensure keys we care about are parsed
	contentType := mimeApplicationJSON
	err := json.Unmarshal(s.resourceSpec, swaggerObj)
	if err != nil {
		contentType = mimeApplicationYAML
		err := yaml.Unmarshal(s.resourceSpec, swaggerObj)
		if err != nil {
			return nil, err
		}
	}
	if swaggerObj.Info.Title == "" {
		return nil, errors.New("invalid openapi 2.0 specification")
	}
	s.resourceContentType = contentType
	return newOas2Processor(swaggerObj), nil
}

func (s *SpecResourceParser) parseOAS3Spec() (SpecProcessor, error) {
	oas3Obj, err := oas.ParseOAS3(s.resourceSpec)
	if err != nil {
		return nil, err
	}
	return newOas3Processor(oas3Obj), nil
}

func (s *SpecResourceParser) parseAsyncAPISpec() (SpecProcessor, error) {
	specDef := make(map[string]interface{})
	// lowercase the byte array to ensure keys we care about are parsed
	contentType := mimeApplicationJSON
	err := json.Unmarshal(s.resourceSpec, &specDef)
	if err != nil {
		contentType = mimeApplicationYAML
		err := yaml.Unmarshal(s.resourceSpec, &specDef)
		if err != nil {
			return nil, err
		}
	}
	_, ok := specDef["asyncapi"]
	if ok {
		s.resourceContentType = contentType
		return newAsyncAPIProcessor(specDef, s.resourceSpec), nil
	}
	return nil, errors.New("invalid asyncapi specification")
}

func (s *SpecResourceParser) parseProtobufSpec() (SpecProcessor, error) {
	reader := bytes.NewReader(s.resourceSpec)
	parser := proto.NewParser(reader)
	definition, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	if len(definition.Elements) > 0 {
		return newProtobufProcessor(definition, s.resourceSpec), nil
	}
	return nil, errors.New("invalid protobuf specification")

}
