package cmd

import (
	"errors"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	lineFormat = "line"
	jsonFormat = "json"
	logPackage = "package"
)

func getFormatter(format string) (log.Formatter, error) {
	switch format {
	case lineFormat:
		return &log.TextFormatter{TimestampFormat: time.RFC3339}, nil
	case jsonFormat:
		return &log.JSONFormatter{TimestampFormat: time.RFC3339}, nil
	default:
		return nil, errors.New("[discovery_agent] invalid log format")
	}
}

// SetupLogging - sets up logging
func SetupLogging(agentName, level, format, outputType, logPath string) {
	setupLogLevel(level)
	setupLogFormatter(format)
	setLogOutput(agentName, outputType, logPath)
}

// setupLogLevel - Sets up the log level
func setupLogLevel(level string) {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		log.Debugf("Unknown logLevel: %v. Defaulting to Info", level)
	} else {
		log.SetLevel(lvl)
	}
}

// setupLogFormatter - Sets up the log format
func setupLogFormatter(format string) {
	formatter, err := getFormatter(format)
	if err != nil {
		log.Debugf("Unknown logFormat: %v. Defaulting to JSON", format)
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: time.RFC3339})
	} else {
		log.SetFormatter(formatter)
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
