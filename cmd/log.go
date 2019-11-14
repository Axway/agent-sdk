package cmd

import (
	"errors"
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
func SetupLogging(level string, format string) {

	lvl, err := log.ParseLevel(level)
	if err == nil {
		log.Debugf("Unknown logLevel: %v. Defaulting to Info", level)
	} else {
		log.SetLevel(lvl)
	}

	formatter, err := getFormatter(format)
	if err != nil {
		log.Debugf("Unknown logFormat: %v. Defaulting to JSON", format)
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: time.RFC3339})
	} else {
		log.SetFormatter(formatter)
	}

	log.SetOutput(os.Stdout)
}
