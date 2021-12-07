package log

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"os"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/sirupsen/logrus"
)

const (
	debugSelector = "apic-agents"
	traceSelector = "apic-agents-trace"
)

var networkTraceIgnoreHeaders = map[string]interface{}{
	"X-Axway-Tenant-Id": true,
	"Authorization":     true,
}
var isLogP bool
var logHTTPTrace bool

func init() {
	networkTrace := os.Getenv("LOG_HTTP_TRACE")
	logHTTPTrace = (networkTrace == "true")
}

//SetIsLogP -
func SetIsLogP() {
	isLogP = true
}

//UnsetIsLogP -
func UnsetIsLogP() {
	isLogP = false
}

// Trace -
func Trace(args ...interface{}) {
	if isLogP {
		// forward trace logs to logp debug with the trace selector
		if log.Level == logrus.TraceLevel {
			logp.Debug(traceSelector, fmt.Sprint(args...))
		}
	} else {
		log.Trace(args...)
	}
}

// Tracef -
func Tracef(format string, args ...interface{}) {
	if isLogP {
		// forward trace logs to logp debug with the trace selector
		if log.Level == logrus.TraceLevel {
			logp.Debug(traceSelector, format, args...)
		}
	} else {
		log.Tracef(format, args...)
	}
}

// Error -
func Error(args ...interface{}) {
	if isLogP {
		logp.Err(fmt.Sprint(args...))
	} else {
		log.Error(args...)
	}
}

// Errorf -
func Errorf(format string, args ...interface{}) {
	if isLogP {
		logp.Err(format, args...)
	} else {
		log.Errorf(format, args...)
	}
}

// Debug -
func Debug(args ...interface{}) {
	if isLogP {
		logp.Debug(debugSelector, fmt.Sprint(args...))
	} else {
		log.Debug(args...)
	}
}

// Debugf -
func Debugf(format string, args ...interface{}) {
	if isLogP {
		logp.Debug(debugSelector, format, args...)
	} else {
		log.Debugf(format, args...)
	}
}

// Info -
func Info(args ...interface{}) {
	if isLogP {
		logp.Info(fmt.Sprint(args...))
	} else {
		log.Info(args...)
	}
}

// Infof -
func Infof(format string, args ...interface{}) {
	if isLogP {
		logp.Info(format, args...)
	} else {
		log.Infof(format, args...)
	}
}

// Warn -
func Warn(args ...interface{}) {
	if isLogP {
		logp.Warn(fmt.Sprint(args...))
	} else {
		log.Warn(args...)
	}
}

// Warnf -
func Warnf(format string, args ...interface{}) {
	if isLogP {
		logp.Warn(format, args...)
	} else {
		log.Warnf(format, args...)
	}
}

// SetLevel -
func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

// GetLevel -
func GetLevel() logrus.Level {
	return log.GetLevel()
}

// DeprecationWarningReplace - log a deprecation warning with the old and replaced usage
func DeprecationWarningReplace(old string, new string) {
	Warnf("%s is deprecated, please start using %s", old, new)
}

// DeprecationWarningDoc - log a deprecation warning with the old and replaced usage
func DeprecationWarningDoc(old string, docRef string) {
	Warnf("%s is deprecated, please refer to docs.axway.com regarding %s", old, docRef)
}

//////////////////////////////
// HTTP client trace logging
//////////////////////////////

// IsHTTPLogTraceEnabled - ...
func IsHTTPLogTraceEnabled() bool {
	return logHTTPTrace && log.GetLevel() == logrus.TraceLevel
}

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
		info.Conn.RemoteAddr().Network(), info.Conn.RemoteAddr().String())
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
		state.ServerName, state.NegotiatedProtocol)
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
