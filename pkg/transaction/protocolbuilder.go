package transaction

import "errors"

// HTTPProtocolBuilder - Interface to build the HTTP protocol details for transaction log event
type HTTPProtocolBuilder interface {
	SetURI(uri string) HTTPProtocolBuilder
	SetVersion(version string) HTTPProtocolBuilder
	SetArgs(args string) HTTPProtocolBuilder
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
	SetIndexedHeaders(indexedRequestHeaders, indexedResponseHeaders string) HTTPProtocolBuilder
	SetPayload(requestPayload, responsePayload string) HTTPProtocolBuilder
	SetWAFStatus(wasStatus int) HTTPProtocolBuilder

	Build() (TransportProtocol, error)
}

type httpProtocolBuilder struct {
	HTTPProtocolBuilder
	err          error
	httpProtocol *Protocol
}

// JMSProtocolBuilder - Interface to build the JMS protocol details for transaction log event
type JMSProtocolBuilder interface {
	SetMessageID(messageID string) JMSProtocolBuilder
	SetCorrelationID(correlationID string) JMSProtocolBuilder
	SetAuthSubjectID(authSubjectID string) JMSProtocolBuilder
	SetDestination(destination string) JMSProtocolBuilder
	SetProviderURL(providerURL string) JMSProtocolBuilder
	SetDeliveryMode(deliveryMode int) JMSProtocolBuilder
	SetPriority(priority int) JMSProtocolBuilder
	SetReplyTo(replyTo string) JMSProtocolBuilder
	SetRedelivered(redelivered int) JMSProtocolBuilder
	SetTimestamp(timestamp int) JMSProtocolBuilder
	SetExpiration(expiration int) JMSProtocolBuilder
	SetJMSType(jmsType string) JMSProtocolBuilder
	SetStatus(status string) JMSProtocolBuilder
	SetStatusText(statusText string) JMSProtocolBuilder

	Build() (TransportProtocol, error)
}

type jmsProtocolBuilder struct {
	JMSProtocolBuilder
	jmsProtocol *JMSProtocol
}

// NewHTTPProtocolBuilder - Creates a new http protocol builder
func NewHTTPProtocolBuilder() HTTPProtocolBuilder {
	builder := &httpProtocolBuilder{
		httpProtocol: &Protocol{
			Type:    "http",
			Version: "1.1",
		},
	}
	return builder
}

// NewJMSProtocolBuilder - Creates a new JMS protocol builder
func NewJMSProtocolBuilder() JMSProtocolBuilder {
	builder := &jmsProtocolBuilder{
		jmsProtocol: &JMSProtocol{
			Type: "jms",
		},
	}
	return builder
}

func (b *httpProtocolBuilder) SetURI(uri string) HTTPProtocolBuilder {
	b.httpProtocol.URI = uri
	return b
}

func (b *httpProtocolBuilder) SetVersion(version string) HTTPProtocolBuilder {
	b.httpProtocol.Version = version
	return b
}

func (b *httpProtocolBuilder) SetArgs(args string) HTTPProtocolBuilder {
	b.httpProtocol.Args = args
	return b
}

func (b *httpProtocolBuilder) SetMethod(method string) HTTPProtocolBuilder {
	b.httpProtocol.Method = method
	return b
}

func (b *httpProtocolBuilder) SetStatus(status int, statusText string) HTTPProtocolBuilder {
	b.httpProtocol.Status = status
	b.httpProtocol.StatusText = statusText
	return b
}

func (b *httpProtocolBuilder) SetUserAgent(userAgent string) HTTPProtocolBuilder {
	b.httpProtocol.UserAgent = userAgent
	return b
}

func (b *httpProtocolBuilder) SetHost(host string) HTTPProtocolBuilder {
	b.httpProtocol.Host = host
	return b
}

func (b *httpProtocolBuilder) SetByteLength(byteReceived, byteSent int) HTTPProtocolBuilder {
	b.httpProtocol.BytesReceived = byteReceived
	b.httpProtocol.BytesSent = byteSent
	return b
}

func (b *httpProtocolBuilder) SetRemoteAddress(remoteName string, remoteAddr string, remotePort int) HTTPProtocolBuilder {
	b.httpProtocol.RemoteName = remoteName
	b.httpProtocol.RemoteAddr = remoteAddr
	b.httpProtocol.RemotePort = remotePort
	return b
}

func (b *httpProtocolBuilder) SetLocalAddress(localAddr string, localPort int) HTTPProtocolBuilder {
	b.httpProtocol.LocalAddr = localAddr
	b.httpProtocol.LocalPort = localPort
	return b
}

func (b *httpProtocolBuilder) SetSSLProperties(sslProtocol string, sslServerName string, sslSubject string) HTTPProtocolBuilder {
	b.httpProtocol.SslProtocol = sslProtocol
	b.httpProtocol.SslServerName = sslServerName
	b.httpProtocol.SslSubject = sslSubject
	return b
}

func (b *httpProtocolBuilder) SetAuthSubjectID(authSubjectID string) HTTPProtocolBuilder {
	b.httpProtocol.AuthSubjectID = authSubjectID
	return b
}

func (b *httpProtocolBuilder) SetHeaders(requestHeaders, responseHeaders string) HTTPProtocolBuilder {
	b.httpProtocol.RequestHeaders = requestHeaders
	b.httpProtocol.ResponseHeaders = responseHeaders
	return b
}

func (b *httpProtocolBuilder) SetIndexedHeaders(indexedRequestHeaders, indexedResponseHeaders string) HTTPProtocolBuilder {
	b.httpProtocol.IndexedRequestHeaders = indexedRequestHeaders
	b.httpProtocol.IndexedResponseHeaders = indexedResponseHeaders
	return b
}

func (b *httpProtocolBuilder) SetPayload(requestPayload, responsePayload string) HTTPProtocolBuilder {
	b.httpProtocol.RequestPayload = requestPayload
	b.httpProtocol.ResponsePayload = responsePayload
	return b
}

func (b *httpProtocolBuilder) SetWAFStatus(wasStatus int) HTTPProtocolBuilder {
	b.httpProtocol.WafStatus = wasStatus
	return b
}

func (b *httpProtocolBuilder) Build() (TransportProtocol, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.httpProtocol.URI == "" {
		return nil, errors.New("URI property not set in HTTP protocol details")
	}

	if b.httpProtocol.Method == "" {
		return nil, errors.New("Method property not set in HTTP protocol details")
	}

	if b.httpProtocol.Host == "" {
		return nil, errors.New("Host property not set in HTTP protocol details")
	}

	if b.httpProtocol.Status < 100 || b.httpProtocol.Status > 600 {
		return nil, errors.New("Invalid status code set in HTTP protocol details")
	}
	return b.httpProtocol, nil
}

func (b *jmsProtocolBuilder) SetMessageID(messageID string) JMSProtocolBuilder {
	b.jmsProtocol.JMSMessageID = messageID
	return b
}

func (b *jmsProtocolBuilder) SetCorrelationID(correlationID string) JMSProtocolBuilder {
	b.jmsProtocol.JMSCorrelationID = correlationID
	return b
}

func (b *jmsProtocolBuilder) SetAuthSubjectID(authSubjectID string) JMSProtocolBuilder {
	b.jmsProtocol.AuthSubjectID = authSubjectID
	return b
}

func (b *jmsProtocolBuilder) SetDestination(destination string) JMSProtocolBuilder {
	b.jmsProtocol.JMSDestination = destination
	return b
}

func (b *jmsProtocolBuilder) SetProviderURL(providerURL string) JMSProtocolBuilder {
	b.jmsProtocol.JMSProviderURL = providerURL
	return b
}

func (b *jmsProtocolBuilder) SetDeliveryMode(deliveryMode int) JMSProtocolBuilder {
	b.jmsProtocol.JMSDeliveryMode = deliveryMode
	return b
}

func (b *jmsProtocolBuilder) SetPriority(priority int) JMSProtocolBuilder {
	b.jmsProtocol.JMSPriority = priority
	return b
}

func (b *jmsProtocolBuilder) SetReplyTo(replyTo string) JMSProtocolBuilder {
	b.jmsProtocol.JMSReplyTo = replyTo
	return b
}

func (b *jmsProtocolBuilder) SetRedelivered(redelivered int) JMSProtocolBuilder {
	b.jmsProtocol.JMSRedelivered = redelivered
	return b
}

func (b *jmsProtocolBuilder) SetTimestamp(timestamp int) JMSProtocolBuilder {
	b.jmsProtocol.JMSTimestamp = timestamp
	return b
}

func (b *jmsProtocolBuilder) SetExpiration(expiration int) JMSProtocolBuilder {
	b.jmsProtocol.JMSExpiration = expiration
	return b
}

func (b *jmsProtocolBuilder) SetJMSType(jmsType string) JMSProtocolBuilder {
	b.jmsProtocol.JMSType = jmsType
	return b
}

func (b *jmsProtocolBuilder) SetStatus(status string) JMSProtocolBuilder {
	b.jmsProtocol.JMSStatus = status
	return b
}

func (b *jmsProtocolBuilder) SetStatusText(statusText string) JMSProtocolBuilder {
	b.jmsProtocol.JMSStatusText = statusText
	return b
}

func (b *jmsProtocolBuilder) Build() (TransportProtocol, error) {
	return b.jmsProtocol, nil
}
