package transaction

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/stretchr/testify/assert"
)

func createHTTPProtocol(uri, method, reqHeaders, resHeaders string, status, reqLen, resLen int, redactionConfig redaction.Redactions) (TransportProtocol, error) {
	redaction.SetupGlobalRedaction(redaction.Config{})
	return NewHTTPProtocolBuilder().
		SetURI(uri).
		SetVersion("1.1").
		SetArgs(`{"param1": ["date"], "param2": ["day, time"]}`).
		SetMethod(method).
		SetStatus(status, "statusTxt").
		SetUserAgent("userAgent").
		SetHost("host").
		SetByteLength(reqLen, resLen).
		SetRemoteAddress("remoteName", "remoteAddr", 2222).
		SetLocalAddress("localAddr", 1111).
		SetAuthSubjectID("authsubject").
		SetSSLProperties("TLS1.1", "sslServer", "sslSubject").
		SetHeaders(reqHeaders, resHeaders).
		SetIndexedHeaders(`{"indexedrequest": "value", "x-amplify-indexed": "random", "x-amplify-indexedagain": "else"}`,
			`{"indexedresponse": "value", "x-indexedresponse": "random", "x-indexed": "test"}`).
		SetPayload("requestPayload", "responsePayload").
		SetWAFStatus(1).
		SetRedactionConfig(redactionConfig).
		Build()
}

func TestHTTPProtocolBuilder(t *testing.T) {
	config := redaction.Config{
		Path: redaction.Path{
			Allowed: []redaction.Show{},
		},
		Args: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		RequestHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		ResponseHeaders: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
		MaskingCharacters: "{*}",
		JMSProperties: redaction.Filter{
			Allowed:  []redaction.Show{},
			Sanitize: []redaction.Sanitize{},
		},
	}

	redactionConfig, _ := config.SetupRedactions()

	httpProtocol, err := createHTTPProtocol("/testuri", "GET", `{"request": "value", "x-amplify-something": "random", "x-amplify-somethingelse": "else"}`,
		`{"response": "value", "x-response": "random", "x-value": "test"}`, 200, 10, 10, redactionConfig)
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocolBuilder := NewHTTPProtocolBuilder()

	httpProtocol, err = httpProtocolBuilder.Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "Raw Uri property not set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "method property not set in HTTP protocol details", err.Error())
	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "host property not set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(20, "OK").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid status code set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		AddArg("newarg", []string{"one", "two"}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		SetArgsMap(map[string][]string{"test": {"one", "two"}}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		SetRequestHeaders(map[string]string{"reqHead": "one"}).
		SetResponseHeaders(map[string]string{"rspHead": "two"}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		AddRequestHeader("key", "two").
		AddResponseHeader("key", "two").
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		SetIndexedRequestHeaders(map[string]string{"test": "one"}).
		SetIndexedResponseHeaders(map[string]string{"test": "two"}).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		AddIndexedRequestHeader("key", "one").
		AddIndexedResponseHeader("key", "two").
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		SetArgsMap(map[string][]string{"test": {"one", "two"}}).
		SetRedactionConfig(redactionConfig).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)
}

type redactionTest struct {
	uriRedactionCalled             bool
	pathRedactionCalled            bool
	queryArgsRedactionCalled       bool
	queryArgsRedactionStringCalled bool
	requestHeadersRedactionCalled  bool
	responseHeadersRedactionCalled bool
	jmsPropertiesRedactionCalled   bool
}

func (r *redactionTest) URIRedaction(uri string) (string, error) {
	r.uriRedactionCalled = true
	return "test", nil
}

func (r *redactionTest) PathRedaction(path string) string {
	r.pathRedactionCalled = true
	return "test"
}

func (r *redactionTest) QueryArgsRedaction(queryArgs map[string][]string) (map[string][]string, error) {
	r.queryArgsRedactionCalled = true
	return queryArgs, nil
}

func (r *redactionTest) QueryArgsRedactionString(queryArgs string) (string, error) {
	r.queryArgsRedactionStringCalled = true
	return queryArgs, nil
}

func (r *redactionTest) RequestHeadersRedaction(requestHeaders map[string]string) (map[string]string, error) {
	r.requestHeadersRedactionCalled = true
	return requestHeaders, nil
}

func (r *redactionTest) ResponseHeadersRedaction(responseHeaders map[string]string) (map[string]string, error) {
	r.responseHeadersRedactionCalled = true
	return responseHeaders, nil
}

func (r *redactionTest) JMSPropertiesRedaction(jmsProperties map[string]string) (map[string]string, error) {
	r.jmsPropertiesRedactionCalled = true
	return jmsProperties, nil
}
func TestRedactionOverride(t *testing.T) {

	redactionConfig := &redactionTest{}

	httpProtocolBuilder := NewHTTPProtocolBuilder()

	httpProtocol, err := httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		SetArgsMap(map[string][]string{"test": {"one", "two"}}).
		AddRequestHeader("key", "one").
		AddResponseHeader("key", "two").
		SetRedactionConfig(redactionConfig).
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)
	assert.True(t, redactionConfig.uriRedactionCalled)
	assert.False(t, redactionConfig.pathRedactionCalled)
	assert.True(t, redactionConfig.queryArgsRedactionCalled)
	assert.False(t, redactionConfig.queryArgsRedactionStringCalled)
	assert.True(t, redactionConfig.requestHeadersRedactionCalled)
	assert.True(t, redactionConfig.responseHeadersRedactionCalled)
	assert.False(t, redactionConfig.jmsPropertiesRedactionCalled)
}
