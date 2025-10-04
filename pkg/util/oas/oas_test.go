package oas

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

var petstore2Json = `{
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
			name:     "Should parse OpenAPI 3.1 spec using pb33f/libopenapi",
			hasError: false,
			spec: `{
				"openapi": "3.1.0",
				"info": {
					"title": "petstore3.1"
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
			SetOAS3Servers(tc.servers, obj)
			assert.Equal(t, len(tc.servers), len(obj.Servers))
			assert.Equal(t, tc.servers[0], obj.Servers[0].URL)
			assert.Equal(t, tc.servers[1], obj.Servers[1].URL)
		})
	}
}

func TestParseOAS2(t *testing.T) {
	tests := []struct {
		name     string
		hasError bool
		spec     string
	}{
		{
			name:     "Should parse the OAS2 spec as json",
			hasError: false,
			spec:     petstore2Json,
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
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseOAS2([]byte(tc.spec))
			if tc.hasError == false {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
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
			obj, err := ParseOAS2([]byte(petstore2Json))
			assert.Nil(t, err)
			err = SetOAS2HostDetails(obj, tc.endpointURL)
			assert.Equal(t, obj.Host, tc.host)
			assert.Equal(t, obj.BasePath, tc.basePath)
		})
	}
}

// Test cases for the hybrid OpenAPI parsing approach
func TestGetOpenAPIVersion(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected string
		hasError bool
	}{
		{
			name: "OpenAPI 3.0 JSON",
			spec: `{
				"openapi": "3.0.0",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {}
			}`,
			expected: "3.0.0",
			hasError: false,
		},
		{
			name: "OpenAPI 3.1 JSON",
			spec: `{
				"openapi": "3.1.0",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {}
			}`,
			expected: "3.1.0",
			hasError: false,
		},
		{
			name: "Swagger 2.0 JSON",
			spec: `{
				"swagger": "2.0",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"paths": {}
			}`,
			expected: "2.0",
			hasError: false,
		},
		{
			name: "Invalid spec",
			spec: `{
				"invalid": "spec"
			}`,
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getOpenAPIVersion([]byte(tt.spec))
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, version)
			}
		})
	}
}

func TestIsOpenAPI31(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"3.0.0", false},
		{"3.0.1", false},
		{"3.0.2", false},
		{"3.1.0", true},
		{"3.1.1", true},
		{"3.2.0", true},
		{"3.3.0", true},
		{"2.0", false},
		{"4.0.0", false}, // hypothetical future major version
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isOpenAPI31(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseOAS31WithLibOpenAPI(t *testing.T) {
	// Test OpenAPI 3.1 spec parsing with pb33f/libopenapi
	oas31Spec := `{
		"openapi": "3.1.0",
		"info": {
			"title": "Test API 3.1",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"responses": {
						"200": {
							"description": "OK"
						}
					}
				}
			}
		}
	}`

	t.Run("Parse valid OpenAPI 3.1 spec", func(t *testing.T) {
		doc, err := parseOAS31WithLibOpenAPI([]byte(oas31Spec))
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "Test API 3.1", doc.Info.Title)
		assert.NotNil(t, doc.Paths)
	})

	t.Run("Parse invalid spec should fail", func(t *testing.T) {
		invalidSpec := `{"invalid": "spec"}`
		_, err := parseOAS31WithLibOpenAPI([]byte(invalidSpec))
		assert.Error(t, err)
	})
}
