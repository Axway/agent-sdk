package log

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	lineFormat = "line"
	jsonFormat = "json"
	logPackage = "package"
)

// Create a new instance of the logger. You can have any number of instances.
var log = logrus.New()

// var contextLogger = log.WithField("package", "apic")

// Error -
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf -
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Debug -
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf -
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Info -
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof -
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Warn -
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf -
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// SetLevel -
func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

func getFormatter(format string) (logrus.Formatter, error) {
	switch format {
	case lineFormat:
		return &logrus.TextFormatter{TimestampFormat: time.RFC3339}, nil
	case jsonFormat:
		return &logrus.JSONFormatter{TimestampFormat: time.RFC3339}, nil
	default:
		return nil, errors.New("[discovery_agent] invalid log format")
	}
}

// SetupLogging - sets up logging
func SetupLogging(agentName, level, format, outputType, logPath string) {
	setupLogLevel(level)
	setupLogFormatter(format)
	setLogOutput(agentName, outputType, logPath)
	// log = logrus.FieldLogger = logrus.WithField("package", "apic")
	// contextLogger = log //logrus.WithField("package", "apic")

	// // SetLog sets the logger for the package.
	// func SetLog(newLog logrus.FieldLogger) {
	// 	log = newLog
	// 	return
	// }

}

// setupLogLevel - Sets up the log level
func setupLogLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		log.Debugf("Unknown logLevel: %v. Defaulting to Info", level)
	} else {
		log.SetLevel(lvl)
		logrus.SetLevel(lvl)
	}
}

// setupLogFormatter - Sets up the log format
func setupLogFormatter(format string) {
	formatter, err := getFormatter(format)
	if err != nil {
		log.Debugf("Unknown logFormat: %v. Defaulting to JSON", format)
		log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
		logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	} else {
		log.SetFormatter(formatter)
		logrus.SetFormatter(formatter)
	}
}

// setLogOutput - Sets up the log output (stdout, file or both)
func setLogOutput(agentName, outputType, logPath string) {
	logWriter := io.Writer(os.Stdout)
	if outputType == "file" {
		logWriter = createFileWriter(logPath, agentName)
	} else if outputType == "both" {
		fileWriter := createFileWriter(logPath, agentName)
		logWriter = io.MultiWriter(os.Stdout, fileWriter)
	}
	log.SetOutput(logWriter)
	logrus.SetOutput(logWriter)
}

func createFileWriter(logPath, agentName string) io.Writer {
	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		panic(err.Error())
	}
	logFile, err := os.OpenFile(logPath+"/"+agentName+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		panic(err.Error())
	}
	return logFile
}
