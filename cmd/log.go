package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	lineFormat = "line"
	jsonFormat = "json"
	logPackage = "package"
)

var log logrus.FieldLogger = logrus.StandardLogger()

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

// SetupLogging - sets up logging for each used package
func SetupLogging(level string, format string) error {

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}

	formatter, err := getFormatter(format)

	if err != nil {
		return err
	}

	logger := logrus.New()

	logger.SetLevel(lvl)
	logger.SetFormatter(formatter)
	logger.SetOutput(os.Stdout)

	return nil

}
