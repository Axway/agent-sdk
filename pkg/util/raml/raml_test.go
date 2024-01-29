package raml

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadRamlAsYaml(t *testing.T) {
	yamlFile, err := os.ReadFile("../../apic/testdata/raml_08.raml")
	if err != nil {
		fmt.Printf("Probably invalid path. Error: %v", err)
	}
	assert.Nil(t, err)

	_, err = Unmarshal(yamlFile)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v", err)
	}
	assert.Nil(t, err)

	yamlFile, err = os.ReadFile("../../apic/testdata/raml_10.raml")
	if err != nil {
		fmt.Printf("Probably invalid path. Error: %v", err)
	}
	assert.Nil(t, err)

	ramlObj, err := Unmarshal(yamlFile)
	if err != nil {
		fmt.Printf("Unmarshal Error: %v", err)
	}
	assert.Nil(t, err)

	fmt.Print(ramlObj)
}
