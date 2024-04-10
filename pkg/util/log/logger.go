package log

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/sirupsen/logrus"
)

// FieldLogger Wraps the StdLogger, and provides logrus methods for logging with fields.
// Intended to mimic the logrus.FieldLogger interface, but with our own interface and implementation.
type FieldLogger interface {
	StdLogger
	WithField(key string, value interface{}) FieldLogger
	WithFields(fields logrus.Fields) FieldLogger
	WithError(err error) FieldLogger
	WithComponent(componentName string) FieldLogger
	WithPackage(packageName string) FieldLogger
}

// StdLogger interface for logging methods found in the go standard library logger, and logrus methods.
type StdLogger interface {
	Redactor
	logrus.StdLogger

	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Debugln(args ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})

	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	Traceln(args ...interface{})
}

// Redactor interface for redacting log messages
type Redactor interface {
	DebugRedacted(fields []string, args ...interface{})
	ErrorRedacted(fields []string, args ...interface{})
	InfoRedacted(fields []string, args ...interface{})
	TraceRedacted(fields []string, args ...interface{})
}

// NewFieldLogger returns a FieldLogger for standard logging, and logp logging.
func NewFieldLogger() FieldLogger {
	entry := logrus.NewEntry(log)
	return &logger{
		entry:  entry,
		noLogP: false,
	}
}

// NewFieldLogger returns a FieldLogger for standard logging, and logp logging.
func NewMetricFieldLogger() FieldLogger {
	entry := logrus.NewEntry(metric)
	return &logger{
		entry:  entry,
		noLogP: true,
	}
}

type logger struct {
	name   string
	entry  *logrus.Entry
	noLogP bool
}

// WithComponent adds a field to the log message
func (l *logger) WithComponent(value string) FieldLogger {
	return &logger{entry: l.entry.WithField("component", value), noLogP: l.noLogP}
}

// WithPackage adds a field to the log message
func (l *logger) WithPackage(value string) FieldLogger {
	return &logger{entry: l.entry.WithField("package", value), noLogP: l.noLogP}
}

// WithField adds a field to the log message
func (l *logger) WithField(key string, value interface{}) FieldLogger {
	return &logger{entry: l.entry.WithField(key, value), noLogP: l.noLogP}
}

// WithFields adds multiple fields to the log message
func (l *logger) WithFields(fields logrus.Fields) FieldLogger {
	return &logger{entry: l.entry.WithFields(fields), noLogP: l.noLogP}
}

// WithError adds an error field to the message
func (l *logger) WithError(err error) FieldLogger {
	return &logger{entry: l.entry.WithError(err), noLogP: l.noLogP}
}

// Debugf prints a formatted debug message
func (l *logger) Debugf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Named(debugSelector).Debugf(format, args...)
		return
	}
	l.entry.Debugf(format, args...)
}

// Infof prints a formatted info message
func (l *logger) Infof(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Infof(format, args...)
		return
	}
	l.entry.Infof(format, args...)
}

// Printf formats a message
func (l *logger) Printf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Infof(format, args...)
		return
	}
	l.entry.Printf(format, args...)
}

// Warnf prints a formatted warning message
func (l *logger) Warnf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Warnf(format, args...)
		return
	}
	l.entry.Warnf(format, args...)
}

// Tracef prints a formatted trace message
func (l *logger) Tracef(format string, args ...interface{}) {
	if l.isLogP() && GetLevel() == logrus.TraceLevel {
		lgp := l.logpWithEntries()
		lgp.Named(traceSelector).Debugf(format, args...)
		return
	}
	l.entry.Tracef(format, args...)
}

// Errorf prints a formatted error message
func (l *logger) Errorf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Errorw(format, args...)
		return
	}
	l.entry.Errorf(format, args...)
}

// Fatalf prints a formatted fatal message
func (l *logger) Fatalf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Fatalw(format, args...)
		return
	}
	l.entry.Fatalf(format, args...)
}

// Panicf prints a formatted panic message
func (l *logger) Panicf(format string, args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Panicw(format, args...)
		return
	}
	l.entry.Panicf(format, args...)
}

// Debug prints a debug message
func (l *logger) Debug(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Named(debugSelector).Debug(args...)
		return
	}
	l.entry.Debug(args...)
}

// Info prints an info message
func (l *logger) Info(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Info(args...)
		return
	}
	l.entry.Info(args...)
}

// Print prints a message
func (l *logger) Print(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Info(args...)
		return
	}
	l.entry.Print(args...)
}

// Trace prints a trace message
func (l *logger) Trace(args ...interface{}) {
	if l.isLogP() && GetLevel() == logrus.TraceLevel {
		lgp := l.logpWithEntries()
		lgp.Named(traceSelector).Debug(args...)
		return
	}
	l.entry.Trace(args...)
}

// Warn prints a warning message
func (l *logger) Warn(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Warn(args...)
		return
	}
	l.entry.Warn(args...)
}

// Error prints an error message
func (l *logger) Error(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Error(args...)
		return
	}
	l.entry.Error(args...)
}

// Fatal prints a fatal error message
func (l *logger) Fatal(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Fatal(args...)
		return
	}
	l.entry.Fatal(args...)
}

// Panic prints a panic message
func (l *logger) Panic(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Panic(args...)
		return
	}
	l.entry.Panic(args...)
}

// Debugln prints a debug line
func (l *logger) Debugln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Named(debugSelector).Debug(args...)
		return
	}
	l.entry.Debugln(args...)
}

// Infoln prints an info line
func (l *logger) Infoln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Info(args...)
		return
	}
	l.entry.Infoln(args...)
}

// Println prints a line
func (l *logger) Println(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Info(args...)
		return
	}
	l.entry.Println(args...)
}

// Traceln prints a trace line
func (l *logger) Traceln(args ...interface{}) {
	if l.isLogP() && GetLevel() == logrus.TraceLevel {
		lgp := l.logpWithEntries()
		lgp.Named(traceSelector).Debug(args...)
		return
	}
	l.entry.Traceln(args...)
}

// Warnln prints a warn line
func (l *logger) Warnln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Warn(args...)
		return
	}
	l.entry.Warnln(args...)
}

// Errorln prints an error line
func (l *logger) Errorln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Error(args...)
		return
	}
	l.entry.Errorln(args...)
}

// Fatalln prints a fatal line
func (l *logger) Fatalln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Fatal(args...)
		return
	}
	l.entry.Fatalln(args...)
}

// Panicln prints a panic line
func (l *logger) Panicln(args ...interface{}) {
	if l.isLogP() {
		lgp := l.logpWithEntries()
		lgp.Panic(args...)
		return
	}
	l.entry.Panicln(args...)
}

// TraceRedacted redacts a trace message
func (l *logger) TraceRedacted(fields []string, args ...interface{}) {
	l.Trace(ObscureArguments(fields, args...))
}

// ErrorRedacted redacts an error message
func (l *logger) ErrorRedacted(fields []string, args ...interface{}) {
	l.Error(ObscureArguments(fields, args...))
}

// InfoRedacted redacts an info message
func (l *logger) InfoRedacted(fields []string, args ...interface{}) {
	l.Info(ObscureArguments(fields, args...))
}

// DebugRedacted redacts a debug message
func (l *logger) DebugRedacted(fields []string, args ...interface{}) {
	l.Debug(ObscureArguments(fields, args...))
}

func (l *logger) isLogP() bool {
	if l.noLogP {
		return false
	}
	return isLogP
}

func (l *logger) logpWithEntries() *logp.Logger {
	var entries []interface{}
	for k, val := range l.entry.Data {
		entries = append(entries, logp.String(k, fmt.Sprintf("%v", val)))
	}
	lgp := logp.L()
	for _, entry := range entries {
		lgp = lgp.With(entry)
	}
	return lgp
}
