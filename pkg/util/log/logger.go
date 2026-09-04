package log

import (
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

// NewFieldLogger returns a FieldLogger for standard logging.
func NewFieldLogger() FieldLogger {
	entry := logrus.NewEntry(log)
	return &logger{entry: entry}
}

// NewMetricFieldLogger returns a FieldLogger for metric logging.
func NewMetricFieldLogger() FieldLogger {
	entry := logrus.NewEntry(metric)
	return &logger{entry: entry}
}

// NewUsageFieldLogger returns a FieldLogger for usage logging.
func NewUsageFieldLogger() FieldLogger {
	entry := logrus.NewEntry(usage)
	return &logger{entry: entry}
}

// NewFieldLoggerEntry returns a FieldLogger wrapping the given logrus.Logger.
func NewFieldLoggerEntry(l *logrus.Logger) FieldLogger {
	entry := logrus.NewEntry(l)
	return &logger{entry: entry}
}

type logger struct {
	entry *logrus.Entry
}

// WithComponent adds a field to the log message
func (l *logger) WithComponent(value string) FieldLogger {
	return &logger{entry: l.entry.WithField("component", value)}
}

// WithPackage adds a field to the log message
func (l *logger) WithPackage(value string) FieldLogger {
	return &logger{entry: l.entry.WithField("package", value)}
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

// Debugf prints a formatted debug message
func (l *logger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof prints a formatted info message
func (l *logger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Printf formats a message
func (l *logger) Printf(format string, args ...interface{}) {
	l.entry.Printf(format, args...)
}

// Warnf prints a formatted warning message
func (l *logger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Tracef prints a formatted trace message
func (l *logger) Tracef(format string, args ...interface{}) {
	l.entry.Tracef(format, args...)
}

// Errorf prints a formatted error message
func (l *logger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatalf prints a formatted fatal message
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// Panicf prints a formatted panic message
func (l *logger) Panicf(format string, args ...interface{}) {
	l.entry.Panicf(format, args...)
}

// Debug prints a debug message
func (l *logger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info prints an info message
func (l *logger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Print prints a message
func (l *logger) Print(args ...interface{}) {
	l.entry.Print(args...)
}

// Trace prints a trace message
func (l *logger) Trace(args ...interface{}) {
	l.entry.Trace(args...)
}

// Warn prints a warning message
func (l *logger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error prints an error message
func (l *logger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Fatal prints a fatal error message
func (l *logger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Panic prints a panic message
func (l *logger) Panic(args ...interface{}) {
	l.entry.Panic(args...)
}

// Debugln prints a debug line
func (l *logger) Debugln(args ...interface{}) {
	l.entry.Debugln(args...)
}

// Infoln prints an info line
func (l *logger) Infoln(args ...interface{}) {
	l.entry.Infoln(args...)
}

// Println prints a line
func (l *logger) Println(args ...interface{}) {
	l.entry.Println(args...)
}

// Traceln prints a trace line
func (l *logger) Traceln(args ...interface{}) {
	l.entry.Traceln(args...)
}

// Warnln prints a warn line
func (l *logger) Warnln(args ...interface{}) {
	l.entry.Warnln(args...)
}

// Errorln prints an error line
func (l *logger) Errorln(args ...interface{}) {
	l.entry.Errorln(args...)
}

// Fatalln prints a fatal line
func (l *logger) Fatalln(args ...interface{}) {
	l.entry.Fatalln(args...)
}

// Panicln prints a panic line
func (l *logger) Panicln(args ...interface{}) {
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
