package redaction

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var requestHeaders = map[string]string{
	"request":                 "value",
	"x-amplify-something":     "random",
	"x-amplify-somethingelse": "else",
}

var responseHeaders = map[string]string{
	"response":   "value",
	"x-response": "random",
	"x-value":    "test",
}

var queryParams = map[string][]string{
	"param1": {"date"},
	"param2": {"day", "time"},
}

func TestDefaultRedaction(t *testing.T) {
	redactionCfg := DefaultConfig()

	SetupGlobalRedaction(redactionCfg)

	// URI redaction
	redactedPath, err := URIRedaction("https://apicentral.axway.com/test/the/path/redaction")
	assert.Nil(t, err)
	assert.NotNil(t, redactedPath)
	assert.Equal(t, "https://apicentral.axway.com/{*}/{*}/{*}/{*}", redactedPath)

	// Query args redaction
	queryArgString := ""
	for key, val := range queryParams {
		if queryArgString != "" {
			queryArgString += "&"
		}
		queryArgString += fmt.Sprintf("%s=%s", key, strings.Join(val, ","))
	}
	redactedQueryParamsString, err := QueryArgsRedactionString(queryArgString)
	assert.Nil(t, err)
	assert.Empty(t, redactedQueryParamsString)
	var redactedQueryParams map[string][]string
	json.Unmarshal([]byte(redactedQueryParamsString), &redactedQueryParams)
	assert.Len(t, redactedQueryParams, 0)

	// Request Header redaction
	redactedRequestHeaders, err := RequestHeadersRedaction(requestHeaders)
	assert.Nil(t, err)
	assert.NotNil(t, redactedRequestHeaders)
	assert.Len(t, redactedRequestHeaders, 0)

	// Response Header redaction
	redactedResponseHeaders, err := ResponseHeadersRedaction(responseHeaders)
	assert.Nil(t, err)
	assert.NotNil(t, redactedResponseHeaders)
	assert.Len(t, redactedResponseHeaders, 0)
}

func TestBadSetupRedaction(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
	}{
		{
			name: "PathRegex",
			config: Config{
				Path: path{
					Allowed: []show{
						{
							KeyMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "QueryArgsAllowRegex",
			config: Config{
				Args: filter{
					Allowed: []show{
						{
							KeyMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "QueryArgsSanitizeRegex",
			config: Config{
				Args: filter{
					Sanitize: []sanitize{
						{
							KeyMatch:   "test",
							ValueMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "ResponseHeadersSanitizeRegex",
			config: Config{
				ResponseHeaders: filter{
					Sanitize: []sanitize{
						{
							KeyMatch:   "*test",
							ValueMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "ResponseHeadersRegex",
			config: Config{
				ResponseHeaders: filter{
					Allowed: []show{
						{
							KeyMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "RequesteHeadersSanitizeRegex",
			config: Config{
				RequestHeaders: filter{
					Sanitize: []sanitize{
						{
							KeyMatch:   "*test",
							ValueMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "RequestHeadersRegex",
			config: Config{
				RequestHeaders: filter{
					Allowed: []show{
						{
							KeyMatch: "*test",
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := SetupGlobalRedaction(test.config)
			assert.NotNil(t, err)
		})
	}

}

func TestURIRedaction(t *testing.T) {
	testCases := []struct {
		name       string
		pathConfig []show
		input      string
		output     string
	}{
		{
			name: "SingleWord",
			pathConfig: []show{
				{
					KeyMatch: "test",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "https://apicentral.axway.com/test/{*}/{*}/{*}",
		},
		{
			name: "TwoWords",
			pathConfig: []show{
				{
					KeyMatch: "test",
				},
				{
					KeyMatch: "redaction",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "https://apicentral.axway.com/test/{*}/{*}/redaction",
		},
		{
			name: "Regex",
			pathConfig: []show{
				{
					KeyMatch: ".*th.*",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "https://apicentral.axway.com/{*}/the/path/{*}",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			defConfig := DefaultConfig()
			defConfig.Path.Allowed = test.pathConfig // update to the test config

			err := SetupGlobalRedaction(defConfig)
			assert.Nil(t, err)

			// URI redaction
			redactedPath, err := URIRedaction(test.input)
			assert.Nil(t, err)
			assert.NotNil(t, redactedPath)
			assert.Equal(t, test.output, redactedPath)
		})
	}
}

func TestQueryParamsRedaction(t *testing.T) {
	testCases := []struct {
		name     string
		qpConfig filter
		input    map[string][]string
		output   map[string][]string
	}{
		{
			name: "SingleParam",
			qpConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "param1",
					},
				},
			},
			input: queryParams,
			output: map[string][]string{
				"param1": {"date"},
			},
		},
		{
			name: "TwoParas",
			qpConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "param1",
					},
					{
						KeyMatch: "param2",
					},
				},
			},
			input: queryParams,
			output: map[string][]string{
				"param1": {"date"},
				"param2": {"day", "time"},
			},
		},
		{
			name: "AllowRegex",
			qpConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "param\\d",
					},
				},
			},
			input: queryParams,
			output: map[string][]string{
				"param1": {"date"},
				"param2": {"day", "time"},
			},
		},
		{
			name: "Sanitize1",
			qpConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "param1",
					},
					{
						KeyMatch: "param2",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "param2",
						ValueMatch: "time",
					},
				},
			},
			input: queryParams,
			output: map[string][]string{
				"param1": {"date"},
				"param2": {"day", "{*}"},
			},
		},
		{
			name: "SanitizeButNotAllowed",
			qpConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "param1",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "param2",
						ValueMatch: "time",
					},
				},
			},
			input: queryParams,
			output: map[string][]string{
				"param1": {"date"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			defConfig := DefaultConfig()
			defConfig.Args = test.qpConfig // update to the test config

			err := SetupGlobalRedaction(defConfig)
			assert.Nil(t, err)

			// QueryParams redaction
			redactedQueryParams, err := QueryArgsRedaction(queryParams)
			assert.Nil(t, err)
			assert.NotNil(t, redactedQueryParams)
			assert.Equal(t, test.output, redactedQueryParams)
		})
	}
}

func TestHeadersRedaction(t *testing.T) {
	testCases := []struct {
		name           string
		responseConfig filter
		requestConfig  filter
		inputResponse  map[string]string
		inputRequest   map[string]string
		outputResponse map[string]string
		outputRequest  map[string]string
	}{
		{
			name: "SingleParam",
			responseConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "x-value",
					},
				},
			},
			requestConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "request",
					},
				},
			},
			inputResponse: responseHeaders,
			inputRequest:  requestHeaders,
			outputResponse: map[string]string{
				"x-value": "test",
			},
			outputRequest: map[string]string{
				"request": "value",
			},
		},
		{
			name: "TwoParams",
			responseConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "x-value",
					},
					{
						KeyMatch: "x-response",
					},
				},
			},
			requestConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "request",
					},
					{
						KeyMatch: "x-amplify-somethingelse",
					},
				},
			},
			inputResponse: responseHeaders,
			inputRequest:  requestHeaders,
			outputResponse: map[string]string{
				"x-value":    "test",
				"x-response": "random",
			},
			outputRequest: map[string]string{
				"request":                 "value",
				"x-amplify-somethingelse": "else",
			},
		},
		{
			name: "Regex",
			responseConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "^.*response",
					},
				},
			},
			requestConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "^x-amplify.*$",
					},
				},
			},
			inputResponse: responseHeaders,
			inputRequest:  requestHeaders,
			outputResponse: map[string]string{
				"response":   "value",
				"x-response": "random",
			},
			outputRequest: map[string]string{
				"x-amplify-something":     "random",
				"x-amplify-somethingelse": "else",
			},
		},
		{
			name: "Sanitize",
			responseConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "^x-.*",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "^x-value.*$",
						ValueMatch: "^tes",
					},
				},
			},
			requestConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "^x-amplify.*$",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "^x-amplify.*$",
						ValueMatch: "^ran",
					},
				},
			},
			inputResponse: responseHeaders,
			inputRequest:  requestHeaders,
			outputResponse: map[string]string{
				"x-response": "random",
				"x-value":    "{*}t",
			},
			outputRequest: map[string]string{
				"x-amplify-something":     "{*}dom",
				"x-amplify-somethingelse": "else",
			},
		},
		{
			name: "SanitizeNoShow",
			responseConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "x-response",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "^x-value",
						ValueMatch: "^tes",
					},
				},
			},
			requestConfig: filter{
				Allowed: []show{
					{
						KeyMatch: "^x-amplify-somethingelse",
					},
				},
				Sanitize: []sanitize{
					{
						KeyMatch:   "^x-amplify-something",
						ValueMatch: "^ran",
					},
				},
			},
			inputResponse: responseHeaders,
			inputRequest:  requestHeaders,
			outputResponse: map[string]string{
				"x-response": "random",
			},
			outputRequest: map[string]string{
				"x-amplify-somethingelse": "else",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			defConfig := DefaultConfig()
			defConfig.RequestHeaders = test.requestConfig   // update to the test config
			defConfig.ResponseHeaders = test.responseConfig // update to the test config

			err := SetupGlobalRedaction(defConfig)
			assert.Nil(t, err)

			// Request Header redaction
			redactedRequestHeaders, err := RequestHeadersRedaction(test.inputRequest)
			assert.Nil(t, err)
			assert.NotNil(t, redactedRequestHeaders)
			assert.Equal(t, test.outputRequest, redactedRequestHeaders)

			// Response Header redaction
			redactedResponseHeaders, err := ResponseHeadersRedaction(test.inputResponse)
			assert.Nil(t, err)
			assert.NotNil(t, redactedResponseHeaders)
			assert.Equal(t, test.outputResponse, redactedResponseHeaders)
		})
	}
}
