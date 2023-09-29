package config

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDefaultLogConfig(t *testing.T) {
	rootCmd := &cobra.Command{}
	props := properties.NewProperties(rootCmd)

	// Add default properties
	logName := "discovery_agent.log"
	AddLogConfigProperties(props, logName)

	// Validate the default
	_, err := ParseAndSetupLogConfig(props)
	assert.Nil(t, err, "Expected no error with default values")
	assert.Equal(t, "info", props.StringPropertyValue(pathLogLevel), "Unexpected default log level")
	assert.Equal(t, "json", props.StringPropertyValue(pathLogFormat), "Unexpected default log format")
	assert.Equal(t, "stdout", props.StringPropertyValue(pathLogOutput), "Unexpected default log output")
	assert.Equal(t, "", props.StringPropertyValue(pathLogMaskedValues), "Unexpected default masked values")
	assert.Equal(t, logName, props.StringPropertyValue(pathLogFileName), "Unexpected default file name")
	assert.Equal(t, "logs", props.StringPropertyValue(pathLogFilePath), "Unexpected default log path")
	assert.Equal(t, 10485760, props.IntPropertyValue(pathLogFileMaxSize), "Unexpected default max size")
	assert.Equal(t, 0, props.IntPropertyValue(pathLogFileMaxAge), "Unexpected default max age")
	assert.Equal(t, 7, props.IntPropertyValue(pathLogFileMaxBackups), "Unexpected default max backups")
}

func TestLogConfigValidations(t *testing.T) {
	// Bad Level
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props := properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug1", "")
	_, err := ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad log level")

	// Bad Format
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line1", "")
	_, err = ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad log format")

	// Bad Output
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line", "")
	props.AddStringProperty(pathLogOutput, "unknown", "")
	_, err = ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad log output")

	// Bad Max Size
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line", "")
	props.AddStringProperty(pathLogOutput, "file", "")
	props.AddStringProperty(pathLogFileName, "filename", "")
	props.AddStringProperty(pathLogFilePath, "path", "")
	props.AddIntProperty(pathLogFileMaxSize, 0, "")
	_, err = ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad max size")

	// Bad Max Backups
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line", "")
	props.AddStringProperty(pathLogOutput, "file", "")
	props.AddStringProperty(pathLogFileName, "filename", "")
	props.AddStringProperty(pathLogFilePath, "path", "")
	props.AddIntProperty(pathLogFileMaxSize, 1048576, "")
	props.AddIntProperty(pathLogFileMaxBackups, -1, "")
	_, err = ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad max backups")

	// Bad Max Age
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line", "")
	props.AddStringProperty(pathLogOutput, "file", "")
	props.AddStringProperty(pathLogFileName, "filename", "")
	props.AddStringProperty(pathLogFilePath, "path", "")
	props.AddIntProperty(pathLogFileMaxSize, 1048576, "")
	props.AddIntProperty(pathLogFileMaxBackups, 1, "")
	props.AddIntProperty(pathLogFileMaxAge, -1, "")
	_, err = ParseAndSetupLogConfig(props)
	assert.NotNil(t, err, "Expected error with bad max age")

	// All Good
	log.GlobalLoggerConfig = log.LoggerConfig{}
	props = properties.NewProperties(&cobra.Command{})
	props.AddStringProperty(pathLogLevel, "debug", "")
	props.AddStringProperty(pathLogFormat, "line", "")
	props.AddStringProperty(pathLogOutput, "file", "")
	props.AddStringProperty(pathLogFileName, "filename", "")
	props.AddStringProperty(pathLogFilePath, "path", "")
	props.AddIntProperty(pathLogFileMaxSize, 1048576, "")
	props.AddIntProperty(pathLogFileMaxBackups, 1, "")
	props.AddIntProperty(pathLogFileMaxAge, 1, "")
	_, err = ParseAndSetupLogConfig(props)
	assert.Nil(t, err, "Expected no errors")
}
