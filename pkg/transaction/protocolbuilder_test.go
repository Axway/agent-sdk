package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createHTTPProtocol(uri, method, reqHeaders, resHeaders string, status, reqLen, resLen int) (TransportProtocol, error) {
	return NewHTTPProtocolBuilder().
		SetURI(uri).
		SetVersion("1.1").
		SetArgs("args").
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
		SetIndexedHeaders("indexedRequestHeaders", "indexedResponseHeaders").
		SetPayload("requestPayload", "responsePayload").
		SetWAFStatus(1).
		Build()
}

func TestHTTPProtocolBuilder(t *testing.T) {
	httpProtocol, err := createHTTPProtocol("/testuri", "GET", "reqHeader", "resHeader", 200, 10, 10)
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)

	httpProtocolBuilder := NewHTTPProtocolBuilder()

	httpProtocol, err = httpProtocolBuilder.Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "URI property not set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "Method property not set in HTTP protocol details", err.Error())
	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "Host property not set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(20, "OK").
		Build()
	assert.Nil(t, httpProtocol)
	assert.NotNil(t, err)
	assert.Equal(t, "Invalid status code set in HTTP protocol details", err.Error())

	httpProtocol, err = httpProtocolBuilder.
		SetURI("/test").
		SetMethod("GET").
		SetHost("host").
		SetStatus(200, "OK").
		Build()
	assert.Nil(t, err)
	assert.NotNil(t, httpProtocol)
}

func createJMSProtocol(msgID, correlationID, jmsType, url, destination, replyTo, status string, mode, priority, exp, timestamp int) (TransportProtocol, error) {
	return NewJMSProtocolBuilder().
		SetMessageID(msgID).
		SetCorrelationID(correlationID).
		SetAuthSubjectID("authSubject").
		SetDestination(destination).
		SetProviderURL(url).
		SetDeliveryMode(mode).
		SetPriority(priority).
		SetReplyTo(replyTo).
		SetRedelivered(0).
		SetTimestamp(timestamp).
		SetExpiration(exp).
		SetJMSType(jmsType).
		SetStatus(status).
		SetStatusText("OK").
		Build()
}
func TestJMSProtocolBuilder(t *testing.T) {
	timeStamp := int(time.Now().Unix())
	jmsProtocol, err := createJMSProtocol("m1", "c1", "jms", "jms://test", "dest", "source", "Success", 1, 1, 2, timeStamp)
	assert.Nil(t, err)
	assert.NotNil(t, jmsProtocol)
}
