package raml

import (
	"gopkg.in/yaml.v3"
)

type RamlObject map[string]interface{}

func Unmarshal(bytes []byte) (RamlObject, error) {
	var obj RamlObject
	err := yaml.Unmarshal(bytes, &obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
