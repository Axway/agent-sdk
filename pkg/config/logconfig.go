package config

import (
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/cmd/properties"
	"git.ecd.axway.org/apigov/apic_agents_sdk/pkg/util/log"
)

// LogConfig - Interface for logging config
type LogConfig interface {
	SetLevel(level string)
}

// LogConfiguration -
type LogConfiguration struct {
	LogConfig
	Level        string               `config:"level"`
	Format       string               `config:"format"`
	Output       string               `config:"output"`
	File         LogFileConfiguration `config:"file"`
	MaskedValues string               `config:"maskedvalues"`
}

func (l *LogConfiguration) setupLogger() error {
	return log.GlobalLoggerConfig.Level(l.Level).
		Format(l.Format).
		Output(l.Output).
		Filename(l.File.Name).
		Path(l.File.Path).
		MaxSize(l.File.MaxSize).
		MaxBackups(l.File.MaxBackups).
		MaxAge(l.File.MaxAge).
		Apply()
}

// LogFileConfiguration - setup the logging configuration for file output
type LogFileConfiguration struct {
	Name       string `config:"name"`
	Path       string `config:"path"`
	MaxSize    int    `config:"rotateeverymegabytes"`
	MaxAge     int    `config:"cleanbackups"`
	MaxBackups int    `config:"keepfiles"`
}

const (
	pathLogLevel          = "log.level"
	pathLogFormat         = "log.format"
	pathLogOutput         = "log.output"
	pathLogMaskedValues   = "log.maskedValues"
	pathLogFileName       = "log.file.name"
	pathLogFilePath       = "log.file.path"
	pathLogFileMaxSize    = "log.file.rotateeverymegabytes"
	pathLogFileMaxAge     = "log.file.cleanbackups"
	pathLogFileMaxBackups = "log.file.keepfiles"
)

// AddLogConfigProperties - Adds the command properties needed for Log Config
func AddLogConfigProperties(props properties.Properties, defaultFileName string) {
	props.AddStringProperty(pathLogLevel, "info", "Log level (debug, info, warn, error)")
	props.AddStringProperty(pathLogFormat, "json", "Log format (json, line)")
	props.AddStringProperty(pathLogOutput, "stdout", "Log output type (stdout, file, both)")
	props.AddStringProperty(pathLogMaskedValues, "", "List of key words in the config to be masked (e.g. pwd, password, secret, key")

	// Log file options
	props.AddStringProperty(pathLogFileName, defaultFileName, "Name of the log files")
	props.AddStringProperty(pathLogFilePath, "logs", "Log file path if output type is file or both")
	props.AddIntProperty(pathLogFileMaxSize, 100, "The maximum size of a log file, in megabytes  (default: 100)")
	props.AddIntProperty(pathLogFileMaxAge, 0, "The maximum number of days, 24 hour periods, to keep the log file backps")
	props.AddIntProperty(pathLogFileMaxBackups, 7, "The maximum number of backups to keep of log files (default: 7)")
}

// ParseAndSetupLogConfig - Parses the Log Config and setups the logger
func ParseAndSetupLogConfig(props properties.Properties) (LogConfig, error) {
	cfg := &LogConfiguration{
		Level:        props.StringPropertyValue(pathLogLevel),
		Format:       props.StringPropertyValue(pathLogFormat),
		Output:       props.StringPropertyValue(pathLogOutput),
		MaskedValues: props.StringPropertyValue(pathLogMaskedValues),
		File: LogFileConfiguration{
			Name:       props.StringPropertyValue(pathLogFileName),
			Path:       props.StringPropertyValue(pathLogFilePath),
			MaxSize:    props.IntPropertyValue(pathLogFileMaxSize),
			MaxBackups: props.IntPropertyValue(pathLogFileMaxBackups),
			MaxAge:     props.IntPropertyValue(pathLogFileMaxAge),
		},
	}

	// Only attempt to mask values if the key maskValues AND key words for maskValues exist
	if cfg.MaskedValues != "" {
		props.MaskValues(cfg.MaskedValues)
	}

	return cfg, cfg.setupLogger()
}
