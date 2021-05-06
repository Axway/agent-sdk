package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var petstore3Json = `{
	"openapi": "3.0.1",
	"info": {
		"title": "petstore3"
	},
	"paths": {},
	"servers": [
		{
			"url": "https://google.com"
		}
	]
}`

var petstore3Yaml = `---
openapi: 3.0.1
info:
  title: petstore3
paths: {}
servers:
- url: https://google.com
`

func TestParseOAS3(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		spec     string
	}{
		{
			name:     "Should parse the OAS3 spec as json",
			hasError: false,
			spec:     petstore3Json,
		},
		{
			name:     "Should parse the OAS3 spec as yaml",
			hasError: false,
			spec:     petstore3Yaml,
		},
		{
			name:     "Should fail to parse the spec when the 'openapi' key is incorrect",
			hasError: true,
			spec: `{
				"openapi": "2.1.1",
				"info": {
					"title": "petstore3"
				},
				"paths": {},
				"servers": [
					{
						"url": "https://google.com"
					}
				]
			}`,
		},
		{
			name:     "Should fail to parse the spec when the 'paths' key is missing",
			hasError: true,
			spec: `{
				"openapi": "3.0.1",
				"info": {
					"title": "petstore3"
				},
				"servers": [
					{
						"url": "https://google.com"
					}
				]
			}`,
		},
		{
			name:     "Should fail to parse the spec when the 'info' key is missing",
			hasError: true,
			spec: `{
				"openapi": "3.0.1",
				"paths": {},
				"servers": [
					{
						"url": "https://google.com"
					}
				]
			}`,
		},
		{
			name:     "Should fail to parse the spec when the 'info.title' key is missing",
			hasError: true,
			spec: `{
				"openapi": "3.0.1",
				"info": {
				},
				"paths": {},
				"servers": [
					{
						"url": "https://google.com"
					}
				]
			}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseOAS3([]byte(tc.spec))
			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestSetServers(t *testing.T) {
	tests := []struct {
		name    string
		servers []string
	}{
		{
			name:    "Should update the servers field with the provided hosts",
			servers: []string{"http://abc.com", "https://123.com"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj, err := ParseOAS3([]byte(petstore3Json))
			assert.Nil(t, err)
			SetServers(tc.servers, obj)
			assert.Equal(t, len(tc.servers), len(obj.Servers))
			assert.Equal(t, tc.servers[0], obj.Servers[0].URL)
			assert.Equal(t, tc.servers[1], obj.Servers[1].URL)
		})
	}
}
