package log

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"

	"github.com/sirupsen/logrus"
)

type httpTrace struct {
	reqID  string
	logger FieldLogger
}

// NewRequestWithTraceContext - New request trace context
func NewRequestWithTraceContext(id string, req *http.Request) *http.Request {
	logger := NewFieldLogger()
	trace := &httpTrace{reqID: id, logger: logger.WithField("component", "httpTrace")}

	clientTrace := &httptrace.ClientTrace{
		GetConn:              trace.logConnection,
		DNSStart:             trace.logDNSStart,
		DNSDone:              trace.logDNSDone,
		ConnectStart:         trace.logConnectStart,
		ConnectDone:          trace.logConnectDone,
		GotConn:              trace.logGotConn,
		WroteHeaderField:     trace.logWroteHeaderField,
		WroteRequest:         trace.logWroteRequest,
		GotFirstResponseByte: trace.logGotFirstResponseByte,
		TLSHandshakeStart:    trace.logTLSHandshakeStart,
		TLSHandshakeDone:     trace.logTLSHandshakeDone,
	}
	clientTraceCtx := httptrace.WithClientTrace(req.Context(), clientTrace)
	return req.WithContext(clientTraceCtx)
}

func (t *httpTrace) logConnection(hostPort string) {
	t.logger.
		WithField("ID", t.reqID).
		WithField("port", hostPort).
		Trace("getting connection")
}

func (t *httpTrace) logDNSStart(info httptrace.DNSStartInfo) {
	t.logger.
		WithField("ID", t.reqID).
		WithField("host", info.Host).
		Trace("dns lookup start")
}

func (t *httpTrace) logDNSDone(info httptrace.DNSDoneInfo) {
	if info.Err != nil {
		t.logger.
			WithField("ID", t.reqID).
			WithError(info.Err).
			Trace("dns lookup failure")
		return
	}
	resolvedIPs := ""
	for _, ip := range info.Addrs {
		if resolvedIPs != "" {
			resolvedIPs += ","
		}
		resolvedIPs += ip.String()
	}
	if resolvedIPs == "" {
		resolvedIPs = "none"
	}
	t.logger.
		WithField("ID", t.reqID).
		WithField("ips", resolvedIPs).
		Trace("dns lookup completed, resolved IPs")
}

func (t *httpTrace) logConnectStart(network, addr string) {
	t.logger.
		WithField("ID", t.reqID).
		WithField("network", network).
		WithField("addr", addr).
		Trace("creating connection")
}

func (t *httpTrace) logConnectDone(network, addr string, err error) {
	if err != nil {
		t.logger.
			WithField("ID", t.reqID).
			WithField("network", network).
			WithField("addr", addr).
			WithError(err).
			Trace("connection creation failure")
		return
	}
	t.logger.
		WithField("ID", t.reqID).
		WithField("network", network).
		WithField("addr", addr).
		Trace("connection created")
}

func (t *httpTrace) logWroteHeaderField(key string, value []string) {
	if _, ok := networkTraceIgnoreHeaders[key]; !ok {
		t.logger.
			WithField("ID", t.reqID).
			WithField("key", key).
			WithField("value", value).
			Trace("writing header")
	} else {
		t.logger.
			WithField("ID", t.reqID).
			WithField("key", key).
			WithField("value", "***").
			Trace("writing header")
	}
}

func (t *httpTrace) logGotConn(info httptrace.GotConnInfo) {
	t.logger.
		WithField("ID", t.reqID).
		WithField("local addr", fmt.Sprintf("%s:%s", info.Conn.LocalAddr().Network(), info.Conn.RemoteAddr().String())).
		WithField("remote addr", fmt.Sprintf("%s:%s", info.Conn.RemoteAddr().Network(), info.Conn.RemoteAddr().String())).
		Trace("connection established")
}

func (t *httpTrace) logTLSHandshakeStart() {
	t.logger.
		WithField("ID", t.reqID).
		Trace("TLS handshake start")
}

func (t *httpTrace) logTLSHandshakeDone(state tls.ConnectionState, err error) {
	if err != nil {
		t.logger.
			WithError(err).
			Trace("TLS handshake failure")
		return
	}
	t.logger.
		WithField("ID", t.reqID).
		WithField("protocol", state.NegotiatedProtocol).
		WithField("server name", state.ServerName).
		Trace("TLS handshake completed")
}

func (t *httpTrace) logGotFirstResponseByte() {
	t.logger.
		WithField("ID", t.reqID).
		Trace("reading response")
}

func (t *httpTrace) logWroteRequest(info httptrace.WroteRequestInfo) {
	if info.Err != nil {
		t.logger.
			WithField("ID", t.reqID).
			WithError(info.Err).
			Trace("failed to write request")
	}
	t.logger.
		WithField("ID", t.reqID).
		Trace("writing request completed")
}

// IsHTTPLogTraceEnabled -
func IsHTTPLogTraceEnabled() bool {
	return logHTTPTrace && log.GetLevel() == logrus.TraceLevel
}
