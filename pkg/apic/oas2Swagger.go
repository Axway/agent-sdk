package apic

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"gopkg.in/yaml.v2"
)

// Oas2Swagger Wrapper type for the openapi2.T struct
type Oas2Swagger struct {
	openapi2.T
}

// UnmarshalYAML - custom unmarshaler for oas2 swagger to ensure keys, at the top level, are lowercased
func (o *Oas2Swagger) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// first unmarshall it into a map[string]interface
	var keyInterface map[string]interface{}
	err := unmarshal(&keyInterface)
	if err != nil {
		return err
	}

	// now loop the keys and lowercase them all
	for key, val := range keyInterface {
		if strings.ToLower(key) == key {
			continue
		}
		// store the val in the lowercase key
		keyInterface[strings.ToLower(key)] = val
		// delete the non-lowercase val
		delete(keyInterface, key)
	}

	// convert keyInterface back to byte array
	newBytes, err := yaml.Marshal(keyInterface)
	if err != nil {
		return err
	}

	// unmarshal new byte array
	var newVal openapi2.T
	yaml.Unmarshal(newBytes, &newVal)

	o.T = newVal
	return nil
}

// ParseOAS2 converts a spec into an OpenAPI2 object
func ParseOAS2(spec []byte) (*Oas2Swagger, error) {
	swaggerObj := &Oas2Swagger{}
	// lowercase the byte array to ensure keys we care about are parsed
	err := yaml.Unmarshal(spec, swaggerObj)
	if err != nil {
		err := json.Unmarshal(spec, swaggerObj)
		if err != nil {
			return nil, err
		}
	}
	if !strings.Contains(swaggerObj.T.Swagger, "2.") {
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

// SetHostDetails Updates the Host, BasePath, and Schemes fields on an Oas2Swagger object
func SetHostDetails(spec *Oas2Swagger, endpointURL string) error {
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
