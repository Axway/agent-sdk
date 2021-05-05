package apic

import (
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParser(t *testing.T) {
	spec := `{"openapi":"3.0.1","components":{"securitySchemes":{"Oauth":{"description":"This API uses OAuth 2 with the implicit grant flow","flows":{"implicit":{"authorizationUrl":"dummy.io","scopes":{}}},"type":"oauth2"}}},"info":{"title":"","version":""},"paths":null,"servers":[{"url":"https://abc.com"}]}`
	parser := NewSpecResourceParser([]byte(spec), "")
	err := parser.Parse()
	assert.Nil(t, err)
	ep, err := parser.SpecProcessor.getEndpoints()
	assert.Nil(t, err)
	logrus.Info(ep)

	oasParser, ok := parser.SpecProcessor.(*Oas3SpecProcessor)
	if !ok {
		logrus.Info("Not an OAS 3 Spec Processor")
	}
	logrus.Info(oasParser)
	// oasSpec := oasParser.spec
}

func RemoveOAS3Policies(spec *openapi3.Swagger) {
	spec.Components.SecuritySchemes = nil
}

func RemoveOAS2Policies(spec *openapi2.Swagger) {
	spec.SecurityDefinitions = nil
}
