package log

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/sirupsen/logrus"
)

// FieldLogger Wraps the StdLogger, and provides logrus methods for logging with fields
type FieldLogger interface {
	StdLogger
	WithField(key string, value interface{}) FieldLogger
	WithFields(fields logrus.Fields) FieldLogger
	WithError(err error) FieldLogger
}

// StdLogger interface for logging methods found in the go standard library logger.
type StdLogger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Trace(v ...interface{})
	Tracef(format string, v ...interface{})
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
}

// LogRedactor interface for redacting log messages
type LogRedactor interface {
	TraceRedacted(fields []string, v ...interface{})
	ErrorRedacted(fields []string, v ...interface{})
	InfoRedacted(fields []string, v ...interface{})
	DebugRedacted(fields []string, v ...interface{})
}

// NewFieldLogger returns a FieldLogger for standard logging, and logp logging.
func NewFieldLogger() FieldLogger {
	entry := logrus.NewEntry(log)
	return &logger{
		entry: entry,
	}
}

type logger struct {
	entry *logrus.Entry
}

// Debug prints a debug message
func (l *logger) Debug(v ...interface{}) {
	if l.isLogP() {
		logp.L().Debug(debugSelector, v)
		return
	}
	l.entry.Debug(v...)
}

// Debugf prints a formatted debug message
func (l *logger) Debugf(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Debug(debugSelector, format, v)
		return
	}
	l.entry.Debugf(format, v...)
}

// Error prints an error message
func (l *logger) Error(v ...interface{}) {
	if l.isLogP() {
		logp.L().Error(v)
		return
	}
	l.entry.Error(v...)
}

// Errorf prints a formatted error message
func (l *logger) Errorf(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Errorf(format, v)
		return
	}
	l.entry.Errorf(format, v...)
}

// Fatal prints a fatal error message
func (l *logger) Fatal(v ...interface{}) {
	if l.isLogP() {
		logp.L().Fatal(v...)
		return
	}
	l.entry.Fatal(v...)
}

// Fatalf prints a formatted fatal message
func (l *logger) Fatalf(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Fatalf(format, v...)
		return
	}
	l.entry.Fatalf(format, v...)
}

// Info prints an info message
func (l *logger) Info(v ...interface{}) {
	if l.isLogP() {
		logp.L().Info(v...)
		return
	}
	l.entry.Info(v...)
}

// Infof prints a formatted info message
func (l *logger) Infof(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Infof(format, v...)
		return
	}
	l.entry.Infof(format, v...)
}

// Panic prints a panic message
func (l *logger) Panic(v ...interface{}) {
	if l.isLogP() {
		logp.L().Panic(v...)
		return
	}
	l.entry.Panic(v...)
}

// Panicf prints a formatted panic message
func (l *logger) Panicf(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Panicf(format, v...)
		return
	}
	l.entry.Panicf(format, v...)
}

// Trace prints a trace message
func (l *logger) Trace(v ...interface{}) {
	if l.isLogP() {
		logp.L().Debug(traceSelector, fmt.Sprint(v...))
		return
	}
	l.entry.Trace(v...)
}

// Tracef prints a formatted trace message
func (l *logger) Tracef(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Debug(traceSelector, fmt.Sprint(v...))
		return
	}
	l.entry.Tracef(format, v...)
}

// Warn prints a warning message
func (l *logger) Warn(v ...interface{}) {
	if l.isLogP() {
		logp.L().Warn(v...)
		return
	}
	l.entry.Warn(v...)
}

// Warnf prints a formatted warning message
func (l *logger) Warnf(format string, v ...interface{}) {
	if l.isLogP() {
		logp.L().Warnf(format, v...)
		return
	}
	l.entry.Warnf(format, v...)
}

// WithField adds a field to the log message
func (l *logger) WithField(key string, value interface{}) FieldLogger {
	return &logger{entry: l.entry.WithField(key, value)}
}

// WithFields adds multiple fields to the log message
func (l *logger) WithFields(fields logrus.Fields) FieldLogger {
	return &logger{entry: l.entry.WithFields(fields)}
}

// WithError adds an error field to the message
func (l *logger) WithError(err error) FieldLogger {
	return &logger{entry: l.entry.WithError(err)}
}

func (l *logger) TraceRedacted(fields []string, v ...interface{}) {
	l.Trace(ObscureArguments(fields, v...))
}

func (l *logger) ErrorRedacted(fields []string, v ...interface{}) {
	l.Error(ObscureArguments(fields, v...))
}

func (l *logger) InfoRedacted(fields []string, v ...interface{}) {
	l.Info(ObscureArguments(fields, v...))
}

func (l *logger) DebugRedacted(fields []string, v ...interface{}) {
	l.Debug(ObscureArguments(fields, v...))
}

func (l *logger) isLogP() bool {
	return isLogP
}
