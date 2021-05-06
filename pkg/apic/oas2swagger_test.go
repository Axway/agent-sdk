package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var spec = `{
	"basePath": "/v2",
	"host": "host.com",
	"schemes": [
			"http"
	],
	"swagger": "2.0",
	"info": {
			"title": "petstore2"
	},
	"paths": {}
}`

var specYaml = `---
basePath: "/v2"
host: host.com
schemes:
- http
swagger: '2.0'
info:
  title: petstore2
paths: {}
`

func TestParseOAS2(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		spec     string
	}{
		{
			name:     "Should parse the OAS2 spec as json",
			hasError: false,
			spec:     spec,
		},
		{
			name:     "Should parse the OAS2 spec as yaml",
			hasError: false,
			spec:     specYaml,
		},
		{
			name:     "Should fail to parse the spec when the 'swagger' key is incorrect",
			hasError: true,
			spec: `{
				"swagger": "1.1",
				"info": {
						"title": "petstore2"
				},
				"paths": {}
			}`,
		},
		{
			name:     "Should fail to parse the spec when the 'paths' key is missing",
			hasError: true,
			spec: `{
				"swagger": "2.0",
				"info": {
						"title": "petstore2"
				},
			}`,
		},
		{
			name:     "Should fail to parse the spec when the 'title' key is missing",
			hasError: true,
			spec: `{
				"swagger": "2.0"
			}`,
		},
	}
	for _, tc := range tests {
		_, err := ParseOAS2([]byte(tc.spec))
		if tc.hasError == false {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestSetOAS2HostDetails(t *testing.T) {
	tests := []struct {
		name        string
		endpointURL string
		host        string
		schemes     []string
		basePath    string
	}{
		{
			name:        "Should update the spec with the provided host",
			endpointURL: "https://newhost.com/v2",
			host:        "newhost.com",
			basePath:    "/v2",
			schemes:     []string{"https"},
		},
		{
			name:        "Should update the spec with the provided host, and set the basePath to '/'",
			endpointURL: "http://newhost.com",
			host:        "newhost.com",
			basePath:    "/",
			schemes:     []string{"http"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := ParseOAS2([]byte(spec))
			assert.Nil(t, err)
			err = SetHostDetails(obj, tc.endpointURL)
			assert.Equal(t, obj.Host, tc.host)
			assert.Equal(t, obj.BasePath, tc.basePath)
			assert.Equal(t, obj.Schemes, tc.schemes)
		})
	}
}
