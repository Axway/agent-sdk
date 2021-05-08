package apic

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"gopkg.in/yaml.v2"
)

// oas2Swagger Wrapper type for the openapi2.T struct
type oas2Swagger struct {
	openapi2.T
}

// UnmarshalYAML - custom unmarshaler for oas2 swagger to ensure keys, at the top level, are lowercased
func (o *oas2Swagger) UnmarshalYAML(unmarshal func(interface{}) error) error {
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
	err = yaml.Unmarshal(newBytes, &newVal)
	if err != nil {
		return err
	}

	o.T = newVal
	return nil
}
