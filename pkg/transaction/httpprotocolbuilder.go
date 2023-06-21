package transaction

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
)

// HTTPProtocolBuilder - Interface to build the HTTP protocol details for transaction log event
type HTTPProtocolBuilder interface {
	SetURI(uri string) HTTPProtocolBuilder
	SetVersion(version string) HTTPProtocolBuilder
	SetArgs(args string) HTTPProtocolBuilder
	SetArgsMap(args map[string][]string) HTTPProtocolBuilder
	AddArg(key string, value []string) HTTPProtocolBuilder
	SetMethod(method string) HTTPProtocolBuilder
	SetStatus(status int, statusText string) HTTPProtocolBuilder
	SetUserAgent(userAgent string) HTTPProtocolBuilder
	SetHost(host string) HTTPProtocolBuilder
	SetByteLength(byteReceived, byteSent int) HTTPProtocolBuilder
	SetRemoteAddress(remoteName string, remoteAddr string, remotePort int) HTTPProtocolBuilder
	SetLocalAddress(localAddr string, localPort int) HTTPProtocolBuilder
	SetSSLProperties(sslProtocol, sslServerName, sslSubject string) HTTPProtocolBuilder
	SetAuthSubjectID(authSubjectID string) HTTPProtocolBuilder
	SetHeaders(requestHeaders, responseHeaders string) HTTPProtocolBuilder
	SetRequestHeaders(requestHeaders map[string]string) HTTPProtocolBuilder
	SetResponseHeaders(responseHeaders map[string]string) HTTPProtocolBuilder
	AddRequestHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	AddResponseHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	SetIndexedHeaders(indexedRequestHeaders, indexedResponseHeaders string) HTTPProtocolBuilder
	SetIndexedRequestHeaders(indexedRequestHeaders map[string]string) HTTPProtocolBuilder
	SetIndexedResponseHeaders(indexedResponseHeaders map[string]string) HTTPProtocolBuilder
	AddIndexedRequestHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	AddIndexedResponseHeader(headerKey string, headerValue string) HTTPProtocolBuilder
	SetPayload(requestPayload, responsePayload string) HTTPProtocolBuilder
	SetWAFStatus(wasStatus int) HTTPProtocolBuilder
	SetRedactionConfig(config redaction.Redactions) HTTPProtocolBuilder

	Build() (TransportProtocol, error)
}

type httpProtocolBuilder struct {
	HTTPProtocolBuilder
	err                    error
	httpProtocol           *Protocol
	argsMap                map[string][]string
	requestHeaders         map[string]string
	responseHeaders        map[string]string
	indexedRequestHeaders  map[string]string
	indexedResponseHeaders map[string]string
	redactionConfig        redaction.Redactions
}

// NewHTTPProtocolBuilder - Creates a new http protocol builder
func NewHTTPProtocolBuilder() HTTPProtocolBuilder {
	builder := &httpProtocolBuilder{
		httpProtocol: &Protocol{
			Type:    "http",
			Version: "1.1",
		},
		argsMap:                make(map[string][]string),
		requestHeaders:         make(map[string]string),
		responseHeaders:        make(map[string]string),
		indexedRequestHeaders:  make(map[string]string),
		indexedResponseHeaders: make(map[string]string),
	}
	return builder
}

func (b *httpProtocolBuilder) SetURI(uri string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.uriRaw = uri
	return b
}

func (b *httpProtocolBuilder) SetVersion(version string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.Version = version
	return b
}

func (b *httpProtocolBuilder) SetArgs(args string) HTTPProtocolBuilder {
	if b.err != nil || args == "" {
		return b
	}
	var argMap map[string][]string
	b.err = json.Unmarshal([]byte(args), &argMap)
	if b.err == nil {
		b.argsMap = argMap
	}
	return b
}

func (b *httpProtocolBuilder) AddArg(key string, value []string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	if _, ok := b.argsMap[key]; ok {
		b.err = fmt.Errorf("arg with key %s has already been added", key)
	} else {
		b.argsMap[key] = value
	}
	return b
}

func (b *httpProtocolBuilder) SetArgsMap(args map[string][]string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.argsMap = args
	return b
}

func (b *httpProtocolBuilder) SetMethod(method string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.Method = method
	return b
}

func (b *httpProtocolBuilder) SetStatus(status int, statusText string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.Status = status
	b.httpProtocol.StatusText = statusText
	return b
}

func (b *httpProtocolBuilder) SetUserAgent(userAgent string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.UserAgent = userAgent
	return b
}

func (b *httpProtocolBuilder) SetHost(host string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.Host = host
	return b
}

func (b *httpProtocolBuilder) SetByteLength(byteReceived, byteSent int) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.BytesReceived = byteReceived
	b.httpProtocol.BytesSent = byteSent
	return b
}

func (b *httpProtocolBuilder) SetRemoteAddress(remoteName string, remoteAddr string, remotePort int) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.RemoteName = remoteName
	b.httpProtocol.RemoteAddr = remoteAddr
	b.httpProtocol.RemotePort = remotePort
	return b
}

func (b *httpProtocolBuilder) SetLocalAddress(localAddr string, localPort int) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.LocalAddr = localAddr
	b.httpProtocol.LocalPort = localPort
	return b
}

func (b *httpProtocolBuilder) SetSSLProperties(sslProtocol string, sslServerName string, sslSubject string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.SslProtocol = sslProtocol
	b.httpProtocol.SslServerName = sslServerName
	b.httpProtocol.SslSubject = sslSubject
	return b
}

func (b *httpProtocolBuilder) SetAuthSubjectID(authSubjectID string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.AuthSubjectID = authSubjectID
	return b
}

func (b *httpProtocolBuilder) SetHeaders(requestHeadersString, responseHeadersString string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}

	if requestHeadersString != "" {
		var requestHeaders map[string]string
		b.err = json.Unmarshal([]byte(requestHeadersString), &requestHeaders)
		if b.err != nil {
			return b
		}
		b.requestHeaders = requestHeaders
	}

	if requestHeadersString != "" {
		var responseHeaders map[string]string
		b.err = json.Unmarshal([]byte(responseHeadersString), &responseHeaders)
		if b.err != nil {
			return b
		}
		b.responseHeaders = responseHeaders
	}
	return b
}

func (b *httpProtocolBuilder) SetRequestHeaders(requestHeaders map[string]string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.requestHeaders = requestHeaders
	return b
}

func (b *httpProtocolBuilder) AddRequestHeader(key string, value string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	if _, ok := b.requestHeaders[key]; ok {
		b.err = fmt.Errorf("response Header with key %s has already been added", key)
	} else {
		b.requestHeaders[key] = value
	}
	return b
}

func (b *httpProtocolBuilder) SetResponseHeaders(responseHeaders map[string]string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.responseHeaders = responseHeaders
	return b
}

func (b *httpProtocolBuilder) AddResponseHeader(key string, value string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	if _, ok := b.responseHeaders[key]; ok {
		b.err = fmt.Errorf("response Header with key %s has already been added", key)
	} else {
		b.responseHeaders[key] = value
	}
	return b
}

func (b *httpProtocolBuilder) SetIndexedHeaders(indexedRequestHeadersString, indexedResponseHeadersString string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}

	var indexedRequestHeaders map[string]string
	b.err = json.Unmarshal([]byte(indexedRequestHeadersString), &indexedRequestHeaders)
	if b.err != nil {
		return b
	}
	b.indexedRequestHeaders = indexedRequestHeaders

	var indexedResponseHeaders map[string]string
	b.err = json.Unmarshal([]byte(indexedResponseHeadersString), &indexedResponseHeaders)
	if b.err != nil {
		return b
	}
	b.indexedResponseHeaders = indexedResponseHeaders
	return b
}

func (b *httpProtocolBuilder) SetIndexedRequestHeaders(indexedRequestHeaders map[string]string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.indexedRequestHeaders = indexedRequestHeaders
	return b
}

func (b *httpProtocolBuilder) AddIndexedRequestHeader(key string, value string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	if _, ok := b.indexedRequestHeaders[key]; ok {
		b.err = fmt.Errorf("indexed Response Header with key %s has already been added", key)
	} else {
		b.indexedRequestHeaders[key] = value
	}
	return b
}

func (b *httpProtocolBuilder) SetIndexedResponseHeaders(indexedResponseHeaders map[string]string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.indexedResponseHeaders = indexedResponseHeaders
	return b
}

func (b *httpProtocolBuilder) AddIndexedResponseHeader(key string, value string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	if _, ok := b.indexedResponseHeaders[key]; ok {
		b.err = fmt.Errorf("indexed Response Header with key %s has already been added", key)
	} else {
		b.indexedResponseHeaders[key] = value
	}
	return b
}

func (b *httpProtocolBuilder) SetPayload(requestPayload, responsePayload string) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.RequestPayload = requestPayload
	b.httpProtocol.ResponsePayload = responsePayload
	return b
}

func (b *httpProtocolBuilder) SetWAFStatus(wasStatus int) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.httpProtocol.WafStatus = wasStatus
	return b
}

func (b *httpProtocolBuilder) SetRedactionConfig(config redaction.Redactions) HTTPProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.redactionConfig = config
	return b
}

func (b *httpProtocolBuilder) Build() (TransportProtocol, error) {
	if b.err != nil {
		return nil, b.err
	}
	// Complete the redactions
	b.queryArgsRedaction()
	if b.err != nil {
		return nil, b.err
	}
	b.headersRedaction()
	if b.err != nil {
		return nil, b.err
	}
	//set redacted URI
	if b.httpProtocol.uriRaw == "" {
		return nil, errors.New("Raw Uri property not set in HTTP protocol details")
	}
	if b.redactionConfig == nil {
		b.httpProtocol.URI, _ = redaction.URIRedaction(b.httpProtocol.uriRaw)
	} else {
		b.httpProtocol.URI, _ = b.redactionConfig.URIRedaction(b.httpProtocol.uriRaw)
	}

	// b.indexedHeadersRedaction()  // Indexed headers are not currently used in central

	if b.httpProtocol.RequestHeaders == "" || b.httpProtocol.ResponseHeaders == "" {
		return nil, errors.New("request or Response Headers not set in HTTP protocol details")
	}

	if b.httpProtocol.URI == "" {
		return nil, errors.New("uri property not set in HTTP protocol details")
	}

	if b.httpProtocol.Method == "" {
		return nil, errors.New("method property not set in HTTP protocol details")
	}

	if b.httpProtocol.Host == "" {
		return nil, errors.New("host property not set in HTTP protocol details")
	}

	if b.httpProtocol.Status < 100 || b.httpProtocol.Status > 600 {
		return nil, errors.New("invalid status code set in HTTP protocol details")
	}
	return b.httpProtocol, nil
}

func (b *httpProtocolBuilder) queryArgsRedaction() {
	// skip if there is already an error
	if b.err != nil {
		return
	}
	var redactedArgs map[string][]string
	var err error
	if len(b.argsMap) > 0 {
		if b.redactionConfig == nil {
			redactedArgs, err = redaction.QueryArgsRedaction(b.argsMap)
		} else {
			redactedArgs, err = b.redactionConfig.QueryArgsRedaction(b.argsMap)
		}
		if err != nil {
			b.err = ErrInRedactions.FormatError("QueryArgs", err)
			return
		}
		argBytes, err := json.Marshal(redactedArgs)
		if err != nil {
			b.err = err
			return
		}
		b.httpProtocol.Args = string(argBytes)
	}
}

func (b *httpProtocolBuilder) headersRedaction() {
	// skip if there is already an error
	if b.err != nil {
		return
	}

	b.httpProtocol.RequestHeaders, b.httpProtocol.ResponseHeaders, b.err =
		headersRedaction(b.requestHeaders, b.responseHeaders, b.redactionConfig)
}

func (b *httpProtocolBuilder) indexedHeadersRedaction() {
	// skip if there is already an error
	if b.err != nil {
		return
	}

	b.httpProtocol.IndexedRequestHeaders, b.httpProtocol.IndexedResponseHeaders, b.err =
		headersRedaction(b.indexedRequestHeaders, b.indexedResponseHeaders, b.redactionConfig)
}

func headersRedaction(requestHeaders, responseHeaders map[string]string, redactionConfig redaction.Redactions) (string, string, error) {
	const emptyHeaders = "{}"
	reqHeadersBytes := []byte(emptyHeaders)
	resHeadersBytes := []byte(emptyHeaders)

	if len(requestHeaders) > 0 {
		var redactedHeaders map[string]string
		var err error
		if redactionConfig == nil {
			redactedHeaders, err = redaction.RequestHeadersRedaction(requestHeaders)
		} else {
			redactedHeaders, err = redactionConfig.RequestHeadersRedaction(redactedHeaders)
		}
		if err != nil {
			return emptyHeaders, emptyHeaders, ErrInRedactions.FormatError("RequestHeaders", err)
		}
		reqHeadersBytes, err = json.Marshal(redactedHeaders)
		if err != nil {
			return emptyHeaders, emptyHeaders, ErrInRedactions.FormatError("RequestHeaders", err)
		}
	}

	if len(responseHeaders) > 0 {
		var redactedHeaders map[string]string
		var err error
		if redactionConfig == nil {
			redactedHeaders, err = redaction.ResponseHeadersRedaction(responseHeaders)
		} else {
			redactedHeaders, err = redactionConfig.ResponseHeadersRedaction(responseHeaders)
		}
		if err != nil {
			return emptyHeaders, emptyHeaders, ErrInRedactions.FormatError("ResponseHeaders", err)
		}
		resHeadersBytes, err = json.Marshal(redactedHeaders)
		if err != nil {
			return emptyHeaders, emptyHeaders, ErrInRedactions.FormatError("ResponseHeaders", err)
		}
	}

	return string(reqHeadersBytes), string(resHeadersBytes), nil
}
