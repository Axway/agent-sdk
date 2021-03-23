package log

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/sirupsen/logrus"
)

const (
	debugSelector = "apic-agents"
	traceSelector = "apic-agents-trace"
)

var isLogP bool

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
