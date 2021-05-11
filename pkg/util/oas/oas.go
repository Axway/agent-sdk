package oas

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

// ParseOAS2 converts a JSON spec into an OpenAPI2 object.
func ParseOAS2(spec []byte) (*openapi2.T, error) {
	swaggerObj := &openapi2.T{}
	err := json.Unmarshal(spec, swaggerObj)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(swaggerObj.Swagger, "2.") {
		return nil, errors.New(oasParseError("2.0", "'swagger' must be version '2.0'."))
	}
	if swaggerObj.Info.Title == "" {
		return nil, errors.New(oasParseError("2.0", "'info.title' key not found."))
	}
	if swaggerObj.Paths == nil {
		return nil, errors.New(oasParseError("2.0", "'paths' key not found."))
	}
	return swaggerObj, nil
}

// ParseOAS3 converts a JSON or YAML spec into an OpenAPI3 object.
func ParseOAS3(spec []byte) (*openapi3.T, error) {
	oas3Obj, err := openapi3.NewLoader().LoadFromData(spec)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(oas3Obj.OpenAPI, "3.") {
		return nil, fmt.Errorf(oasParseError("3", ("'openapi' key is invalid.")))
	}
	if oas3Obj.Paths == nil {
		return nil, fmt.Errorf(oasParseError("3", "'paths' key not found."))
	}
	if oas3Obj.Info == nil {
		return nil, fmt.Errorf(oasParseError("3", "'info' key not found."))
	}
	if oas3Obj.Info.Title == "" {
		return nil, fmt.Errorf(oasParseError("3", "'info.title' key not found."))
	}
	return oas3Obj, nil
}

// SetOAS2HostDetails Updates the Host, BasePath, and Schemes fields on an OpenAPI2 object.
func SetOAS2HostDetails(spec *openapi2.T, endpointURL string) error {
	endpoint, err := url.Parse(endpointURL)
	if err != nil {
		return err
	}

	basePath := ""
	if endpoint.Path == "" {
		basePath = "/"
	} else {
		basePath = endpoint.Path
	}

	host := endpoint.Host
	schemes := []string{endpoint.Scheme}
	spec.Host = host
	spec.BasePath = basePath
	spec.Schemes = schemes
	return nil
}

// SetOAS3Servers replaces the servers array on the OpenAPI3 object.
func SetOAS3Servers(hosts []string, spec *openapi3.T) {
	var oas3Servers []*openapi3.Server
	for _, s := range hosts {
		oas3Servers = append(oas3Servers, &openapi3.Server{
			URL: s,
		})
	}
	if len(oas3Servers) > 0 {
		spec.Servers = oas3Servers
	}
}

func oasParseError(version string, msg string) string {
	return fmt.Sprintf("invalid openapi %s specification. %s", version, msg)
}
