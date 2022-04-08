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
}

// Redactor interface for redacting log messages
type Redactor interface {
	TraceRedacted(fields []string, args ...interface{})
	ErrorRedacted(fields []string, args ...interface{})
	InfoRedacted(fields []string, args ...interface{})
	DebugRedacted(fields []string, args ...interface{})
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
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debugf(format, args)
		return
	}
	l.entry.Debugf(format, args...)
}

// Infof prints a formatted info message
func (l *logger) Infof(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Infof(format, args...)
		return
	}
	l.entry.Infof(format, args...)
}

// Printf formats a message
func (l *logger) Printf(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debugf(format, args)
		return
	}
	l.entry.Printf(format, args...)
}

// Warnf prints a formatted warning message
func (l *logger) Warnf(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Warnf(format, args...)
		return
	}
	l.entry.Warnf(format, args...)
}

// Tracef prints a formatted trace message
func (l *logger) Tracef(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debugf(format, args...)
		return
	}
	l.entry.Tracef(format, args...)
}

// Errorf prints a formatted error message
func (l *logger) Errorf(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Errorf(format, args)
		return
	}
	l.entry.Errorf(format, args...)
}

// Fatalf prints a formatted fatal message
func (l *logger) Fatalf(format string, args ...interface{}) {
	if l.isLogP() {
		logp.L().Fatalf(format, args...)
		return
	}
	l.entry.Fatalf(format, args...)
}

// Panicf prints a formatted panic message
func (l *logger) Panicf(format string, args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Panicf(format, args...)
		return
	}
	l.entry.Panicf(format, args...)
}

// Debug prints a debug message
func (l *logger) Debug(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debug(args...)
		return
	}
	l.entry.Debug(args...)
}

// Info prints an info message
func (l *logger) Info(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Info(args...)
		return
	}
	l.entry.Info(args...)
}

// Print prints a message
func (l *logger) Print(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Info(args...)
		return
	}
	l.entry.Print(args...)
}

// Trace prints a trace message
func (l *logger) Trace(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debug(args...)
		return
	}
	l.entry.Trace(args...)
}

// Warn prints a warning message
func (l *logger) Warn(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Warn(args...)
		return
	}
	l.entry.Warn(args...)
}

// Error prints an error message
func (l *logger) Error(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Error(args...)
		return
	}
	l.entry.Error(args...)
}

// Fatal prints a fatal error message
func (l *logger) Fatal(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Fatal(args...)
		return
	}
	l.entry.Fatal(args...)
}

// Panic prints a panic message
func (l *logger) Panic(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Panic(args...)
		return
	}
	l.entry.Panic(args...)
}

// Debugln prints a debug line
func (l *logger) Debugln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debug(args...)
		return
	}
	l.entry.Debugln(args...)
}

// Infoln prints an info line
func (l *logger) Infoln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Info(args...)
		return
	}
	l.entry.Infoln(args...)
}

// Println prints a line
func (l *logger) Println(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Info(args...)
		return
	}
	l.entry.Println(args...)
}

// Traceln prints a trace line
func (l *logger) Traceln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Debug(args...)
		return
	}
	l.entry.Trace(args...)
}

// Warnln prints a warn line
func (l *logger) Warnln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Warn(args...)
		return
	}
	l.entry.Warnln(args...)
}

// Errorln prints an error line
func (l *logger) Errorln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Error(args...)
		return
	}
	l.entry.Errorln(args...)
}

// Fatalln prints a fatal line
func (l *logger) Fatalln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Fatal(args...)
		return
	}
	l.entry.Fatalln(args...)
}

// Panicln prints a panic line
func (l *logger) Panicln(args ...interface{}) {
	if l.isLogP() {
		args = append(args, l.formatEntries()...)
		logp.L().Panic(args...)
		return
	}
	l.entry.Panic(args...)
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
	return true
}

func (l *logger) formatEntries() []interface{} {
	var args []interface{}
	for k, val := range l.entry.Data {
		s := fmt.Sprintf("%v=%v", k, val)
		args = append(args, s)
	}
	return args
}
