package log

import (
	"fmt"
	"os"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/sirupsen/logrus"
)

const (
	debugSelector = "apic-agents"
	traceSelector = "apic-agents-trace"
)

// Get returns the global logger
func Get() *logrus.Logger {
	return log
}

// GetMetricLogger returns the metric logger
func GetMetricLogger() *logrus.Logger {
	return metric
}

func GetPublishedTransactionLogger() *logrus.Logger {
	return usage
}

var networkTraceIgnoreHeaders = map[string]interface{}{
	"X-Axway-Tenant-Id": true,
	"Authorization":     true,
}

var isLogP bool
var logHTTPTrace bool

func init() {
	networkTrace := os.Getenv("LOG_HTTP_TRACE")
	logHTTPTrace = networkTrace == "true"
}

// SetIsLogP -
func SetIsLogP() {
	isLogP = true
}

// UnsetIsLogP -
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

// TraceRedacted Redacted log for traces
func TraceRedacted(redactedFields []string, args ...interface{}) {
	Trace(ObscureArguments(redactedFields, args...))
}

// ErrorRedacted Redacted log for errors
func ErrorRedacted(redactedFields []string, args ...interface{}) {
	Error(ObscureArguments(redactedFields, args...))
}

// InfoRedacted Redacted log for information
func InfoRedacted(redactedFields []string, args ...interface{}) {
	Info(ObscureArguments(redactedFields, args...))
}

// DebugRedacted Redacted log for debugging
func DebugRedacted(redactedFields []string, args ...interface{}) {
	Debug(ObscureArguments(redactedFields, args...))
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
