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

var jmsProperties = map[string]string{
	"jmsMessageID":   "messageid",
	"jmsDestination": "queue://test-queue",
	"jmsReplyTo":     "queue://reply-queue",
}

func TestDefaultRedaction(t *testing.T) {
	redactionCfg := DefaultConfig()

	SetupGlobalRedaction(redactionCfg)

	// URI redaction
	redactedPath, err := URIRedaction("https://apicentral.axway.com/test/the/path/redaction")
	assert.Nil(t, err)
	assert.NotNil(t, redactedPath)
	assert.Equal(t, "/{*}/{*}/{*}/{*}", redactedPath)

	// Only send path to URI redaction
	redactedPath, err = URIRedaction("/test/the/path/redaction")
	assert.Nil(t, err)
	assert.NotNil(t, redactedPath)
	assert.Equal(t, "/{*}/{*}/{*}/{*}", redactedPath)

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

	// JMS Property redaction
	redactedJMSProperties, err := JMSPropertiesRedaction(jmsProperties)
	assert.Nil(t, err)
	assert.NotNil(t, redactedJMSProperties)
	assert.Len(t, redactedJMSProperties, 0)
}

func TestBadSetupRedaction(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
	}{
		{
			name: "PathRegex",
			config: Config{
				Path: Path{
					Allowed: []Show{
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
				Args: Filter{
					Allowed: []Show{
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
				Args: Filter{
					Sanitize: []Sanitize{
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
				ResponseHeaders: Filter{
					Sanitize: []Sanitize{
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
				ResponseHeaders: Filter{
					Allowed: []Show{
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
				RequestHeaders: Filter{
					Sanitize: []Sanitize{
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
				RequestHeaders: Filter{
					Allowed: []Show{
						{
							KeyMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "JMSPropertiesSanitizeRegex",
			config: Config{
				RequestHeaders: Filter{
					Sanitize: []Sanitize{
						{
							KeyMatch:   "*test",
							ValueMatch: "*test",
						},
					},
				},
			},
		},
		{
			name: "JMSPropertiesRegex",
			config: Config{
				RequestHeaders: Filter{
					Allowed: []Show{
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
		pathConfig []Show
		input      string
		output     string
	}{
		{
			name: "SingleWord",
			pathConfig: []Show{
				{
					KeyMatch: "test",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "/test/{*}/{*}/{*}",
		},
		{
			name: "TwoWords",
			pathConfig: []Show{
				{
					KeyMatch: "test",
				},
				{
					KeyMatch: "redaction",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "/test/{*}/{*}/redaction",
		},
		{
			name: "Regex",
			pathConfig: []Show{
				{
					KeyMatch: ".*th.*",
				},
			},
			input:  "https://apicentral.axway.com/test/the/path/redaction",
			output: "/{*}/the/path/{*}",
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
		qpConfig Filter
		input    map[string][]string
		output   map[string][]string
	}{
		{
			name: "SingleParam",
			qpConfig: Filter{
				Allowed: []Show{
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
			qpConfig: Filter{
				Allowed: []Show{
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
			qpConfig: Filter{
				Allowed: []Show{
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
			qpConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "param1",
					},
					{
						KeyMatch: "param2",
					},
				},
				Sanitize: []Sanitize{
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
			qpConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "param1",
					},
				},
				Sanitize: []Sanitize{
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
		responseConfig Filter
		requestConfig  Filter
		inputResponse  map[string]string
		inputRequest   map[string]string
		outputResponse map[string]string
		outputRequest  map[string]string
	}{
		{
			name: "SingleParam",
			responseConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "x-value",
					},
				},
			},
			requestConfig: Filter{
				Allowed: []Show{
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
			responseConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "x-value",
					},
					{
						KeyMatch: "x-response",
					},
				},
			},
			requestConfig: Filter{
				Allowed: []Show{
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
			responseConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "^.*response",
					},
				},
			},
			requestConfig: Filter{
				Allowed: []Show{
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
			responseConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "^x-.*",
					},
				},
				Sanitize: []Sanitize{
					{
						KeyMatch:   "^x-value.*$",
						ValueMatch: "^tes",
					},
				},
			},
			requestConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "^x-amplify.*$",
					},
				},
				Sanitize: []Sanitize{
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
			responseConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "x-response",
					},
				},
				Sanitize: []Sanitize{
					{
						KeyMatch:   "^x-value",
						ValueMatch: "^tes",
					},
				},
			},
			requestConfig: Filter{
				Allowed: []Show{
					{
						KeyMatch: "^x-amplify-somethingelse",
					},
				},
				Sanitize: []Sanitize{
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
