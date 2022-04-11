package log

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"

	"github.com/sirupsen/logrus"
)

type httpTrace struct {
	reqID string
}

// NewRequestWithTraceContext - New request trace context
func NewRequestWithTraceContext(id string, req *http.Request) *http.Request {
	trace := &httpTrace{reqID: id}

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
	Tracef("[ID:%s] getting connection %s", t.reqID, hostPort)
}

func (t *httpTrace) logDNSStart(info httptrace.DNSStartInfo) {
	Tracef("[ID:%s] dns lookup start, host: %s", t.reqID, info.Host)
}

func (t *httpTrace) logDNSDone(info httptrace.DNSDoneInfo) {
	if info.Err != nil {
		Tracef("[ID:%s] dns lookup failure, error: %s", t.reqID, info.Err.Error())
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
	Tracef("[ID:%s] dns lookup completed, resolved IPs: %s error: %s", t.reqID, resolvedIPs, info.Err)
}

func (t *httpTrace) logConnectStart(network, addr string) {
	Tracef("[ID:%s] creating connection %s:%s", t.reqID, network, addr)
}

func (t *httpTrace) logConnectDone(network, addr string, err error) {
	if err != nil {
		Tracef("[ID:%s] connection creation failure %s:%s error %s", t.reqID, network, addr, err.Error())
		return
	}
	Tracef("[ID:%s] connection created %s:%s", t.reqID, network, addr)
}

func (t *httpTrace) logWroteHeaderField(key string, value []string) {
	if _, ok := networkTraceIgnoreHeaders[key]; !ok {
		Tracef("[ID:%s] writing header %s: %v", t.reqID, key, value)
	} else {
		Tracef("[ID:%s] writing header %s: ***", t.reqID, key)
	}
}

func (t *httpTrace) logGotConn(info httptrace.GotConnInfo) {
	Tracef("[ID:%s] connection established, local addr: %s:%s, remote addr: %s:%s", t.reqID,
		info.Conn.LocalAddr().Network(), info.Conn.RemoteAddr().String(),
		info.Conn.RemoteAddr().Network(), info.Conn.RemoteAddr().String(),
	)
}

func (t *httpTrace) logTLSHandshakeStart() {
	Tracef("[ID:%s] TLS handshake start", t.reqID)
}

func (t *httpTrace) logTLSHandshakeDone(state tls.ConnectionState, err error) {
	if err != nil {
		Tracef("[ID:%s] TLS handshake failure, error: %s", t.reqID, err.Error())
		return
	}
	Tracef("[ID:%s] TLS handshake completed, server name: %s protocol: %s", t.reqID,
		state.ServerName, state.NegotiatedProtocol,
	)
}

func (t *httpTrace) logGotFirstResponseByte() {
	Tracef("[ID:%s] reading response", t.reqID)
}

func (t *httpTrace) logWroteRequest(info httptrace.WroteRequestInfo) {
	if info.Err != nil {
		Tracef("[ID:%s] failed to write request, error: %s", t.reqID, info.Err.Error())
	}
	Tracef("[ID:%s] writing request completed", t.reqID)
}

// IsHTTPLogTraceEnabled -
func IsHTTPLogTraceEnabled() bool {
	return logHTTPTrace && log.GetLevel() == logrus.TraceLevel
}
