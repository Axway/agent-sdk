package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/sirupsen/logrus"
)

// LogConfig - Interface for logging config
type LogConfig interface {
	SetLevel(level string)
	GetMetricConfig() LogFileConfiguration
	GetUsageConfig() LogFileConfiguration
}

// LogConfiguration -
type LogConfiguration struct {
	LogConfig
	Level        string               `config:"level"`
	Format       string               `config:"format"`
	Output       string               `config:"output"`
	File         LogFileConfiguration `config:"file"`
	MetricFile   LogFileConfiguration `config:"metricfile"`
	UsageFile    LogFileConfiguration `config:"usagefile"`
	MaskedValues string               `config:"maskedvalues"`
}

func (l *LogConfiguration) setupLogger(agentType AgentType) error {
	return log.GlobalLoggerConfig.Level(l.Level).
		Format(l.Format).
		Output(l.Output).
		Filename(l.File.Name).
		Path(l.File.Path).
		MaxSize(l.File.MaxSize).
		MaxBackups(l.File.MaxBackups).
		MaxAge(l.File.MaxAge).
		Metrics(agentType != DiscoveryAgent && l.MetricFile.Enabled).
		MetricFilename(l.MetricFile.Name).
		MaxMetricSize(l.MetricFile.MaxSize).
		MaxMetricBackups(l.MetricFile.MaxBackups).
		MaxMetricAge(l.MetricFile.MaxAge).
		Usage(agentType != DiscoveryAgent && l.UsageFile.Enabled).
		UsageFilename(l.UsageFile.Name).
		MaxUsageSize(l.UsageFile.MaxSize).
		MaxUsageBackups(l.UsageFile.MaxBackups).
		MaxUsageAge(l.UsageFile.MaxAge).
		Apply()
}

func (l *LogConfiguration) GetMetricConfig() LogFileConfiguration {
	return l.MetricFile
}

func (l *LogConfiguration) GetUsagefileConfig() LogFileConfiguration {
	return l.UsageFile
}

// LogFileConfiguration - setup the logging configuration for file output
type LogFileConfiguration struct {
	Enabled    bool   `config:"enable"`
	Name       string `config:"name"`
	Path       string `config:"path"`
	MaxSize    int    `config:"rotateeverybytes"`
	MaxAge     int    `config:"cleanbackups"`
	MaxBackups int    `config:"keepfiles"`
}

const (
	pathLogLevel                 = "log.level"
	pathLogFormat                = "log.format"
	pathLogOutput                = "log.output"
	pathLogMaskedValues          = "log.maskedValues"
	pathLogFileName              = "log.file.name"
	pathLogFilePath              = "log.file.path"
	pathLogFileMaxSize           = "log.file.rotateeverybytes"
	pathLogFileMaxAge            = "log.file.cleanbackups"
	pathLogFileMaxBackups        = "log.file.keepfiles"
	pathLogMetricsFileEnabled    = "log.metricfile.enabled"
	pathLogMetricsFileName       = "log.metricfile.name"
	pathLogMetricsFileMaxSize    = "log.metricfile.rotateeverybytes"
	pathLogMetricsFileMaxAge     = "log.metricfile.cleanbackups"
	pathLogMetricsFileMaxBackups = "log.metricfile.keepfiles"
	pathLogUsageFileEnabled      = "log.usagefile.enabled"
	pathLogUsageFileName         = "log.usagefile.name"
	pathLogUsageFileMaxSize      = "log.usagefile.rotateeverybytes"
	pathLogUsageFileMaxAge       = "log.usagefile.cleanbackupsevery"
	pathLogUsageFileMaxBackups   = "log.usagefile.keepfiles"
)

// AddLogConfigProperties - Adds the command properties needed for Log Config
func AddLogConfigProperties(props properties.Properties, defaultFileName string) {
	props.AddStringProperty(pathLogLevel, "info", "Log level (trace, debug, info, warn, error)")
	props.AddStringProperty(pathLogFormat, "json", "Log format (json, line)")
	props.AddStringProperty(pathLogOutput, "stdout", "Log output type (stdout, file, both)")
	props.AddStringProperty(pathLogMaskedValues, "", "List of key words in the config to be masked (e.g. pwd, password, secret, key")

	// Log file options
	props.AddStringProperty(pathLogFileName, defaultFileName, "Name of the log files")
	props.AddStringProperty(pathLogFilePath, "logs", "Log file path if output type is file or both")
	props.AddIntProperty(pathLogFileMaxSize, 10485760, "The maximum size of a log file, in bytes  (default: 10485760 - 10 MB)")
	props.AddIntProperty(pathLogFileMaxAge, 0, "The maximum number of days, 24 hour periods, to keep the log file backups")
	props.AddIntProperty(pathLogFileMaxBackups, 7, "The maximum number of backups to keep of log files (default: 7)")
}

// AddMetricLogConfigProperties - Adds the command properties needed for Log Config
func AddMetricLogConfigProperties(props properties.Properties, agentType AgentType) {
	if agentType == DiscoveryAgent {
		return
	}
	// Metric log file options
	props.AddBoolProperty(pathLogMetricsFileEnabled, true, "Set to false to disable metrics file logging")
	props.AddStringProperty(pathLogMetricsFileName, "metrics.log", "Name of the metric log files")
	props.AddIntProperty(pathLogMetricsFileMaxSize, 10485760, "The maximum size of a metrics log file, in bytes (default: 10485760 - 10 MB)")
	props.AddIntProperty(pathLogMetricsFileMaxAge, 0, "The maximum number of days, 24 hour periods, to keep the metrics log file backups")
	props.AddIntProperty(pathLogMetricsFileMaxBackups, 0, "The maximum number of backups to keep of metrics log files (default: unlimited)")
}

func AddUsageConfigProperties(props properties.Properties, agentType AgentType) {
	if agentType == DiscoveryAgent {
		return
	}

	props.AddBoolProperty(pathLogUsageFileEnabled, true, "Set to false to disable usage file logging")
	props.AddStringProperty(pathLogUsageFileName, "usage.log", "Name of the usage log files")
	props.AddIntProperty(pathLogUsageFileMaxSize, 10485760, "The maximum size of a usage log file, in bytes (default: 10485760 - 10 MB)")
	props.AddIntProperty(pathLogUsageFileMaxAge, 365, "The maximum number of days, 24 hour periods, to keep the usage log file backups")
	props.AddIntProperty(pathLogUsageFileMaxBackups, 0, "The maximum number of backups to keep of metrics log files (default: unlimited)")
}

// ParseAndSetupLogConfig - Parses the Log Config and setups the logger
func ParseAndSetupLogConfig(props properties.Properties, agentType AgentType) (LogConfig, error) {
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

	if agentType == TraceabilityAgent || agentType == ComplianceAgent {
		cfg.MetricFile = LogFileConfiguration{
			Enabled:    props.BoolPropertyValue(pathLogMetricsFileEnabled),
			Name:       props.StringPropertyValue(pathLogMetricsFileName),
			Path:       filepath.Join(cfg.File.Path, "metrics"),
			MaxSize:    props.IntPropertyValue(pathLogMetricsFileMaxSize),
			MaxBackups: props.IntPropertyValue(pathLogMetricsFileMaxBackups),
			MaxAge:     props.IntPropertyValue(pathLogMetricsFileMaxAge),
		}
		cfg.UsageFile = LogFileConfiguration{
			Enabled:    props.BoolPropertyValue(pathLogUsageFileEnabled),
			Name:       props.StringPropertyValue(pathLogUsageFileName),
			Path:       filepath.Join(cfg.File.Path, "usage"),
			MaxSize:    props.IntPropertyValue(pathLogUsageFileMaxSize),
			MaxBackups: props.IntPropertyValue(pathLogUsageFileMaxBackups),
			MaxAge:     props.IntPropertyValue(pathLogUsageFileMaxAge),
		}
	}

	// Only attempt to mask values if the key maskValues AND key words for maskValues exist
	if cfg.MaskedValues != "" {
		props.MaskValues(cfg.MaskedValues)
	}

	return cfg, cfg.setupLogger(agentType)
}

const (
	logLevelYAMLPath       = "logging.level"
	logJSONYAMLPath        = "logging.json"
	logSTDERRYAMLPath      = "logging.to_stderr"
	logFileYAMLPath        = "logging.to_files"
	logFilePermissionsPath = "logging.files.permissions"
)

// LogConfigOverrides - override the filebeat config options
func LogConfigOverrides() []cfgfile.ConditionalOverride {
	overrides := make([]cfgfile.ConditionalOverride, 0)
	overrides = setLogLevel(overrides)
	return overrideLogLevel(overrides)
}

func setLogLevel(overrides []cfgfile.ConditionalOverride) []cfgfile.ConditionalOverride {
	// Set level to info
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogLevel), 0)
			level, err := logrus.ParseLevel(output)
			if err == nil && level == logrus.InfoLevel {
				return true
			}
			return false
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logLevelYAMLPath: "info",
		}),
	})

	// Set level to warn
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogLevel), 0)
			level, err := logrus.ParseLevel(output)
			if err == nil && level == logrus.WarnLevel {
				return true
			}
			return false
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logLevelYAMLPath: "warn",
		}),
	})

	// Set level to error
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogLevel), 0)
			level, err := logrus.ParseLevel(output)
			if err == nil && level == logrus.ErrorLevel {
				return true
			}
			return false
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logLevelYAMLPath: "error",
		}),
	})

	return overrides
}

func overrideLogLevel(overrides []cfgfile.ConditionalOverride) []cfgfile.ConditionalOverride {
	// Override the level to debug, if trace or debug
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogLevel), 0)
			level, err := logrus.ParseLevel(output)
			if err == nil && (level == logrus.TraceLevel || level == logrus.DebugLevel) {
				return true
			}
			return false
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logLevelYAMLPath: "debug",
		}),
	})

	// Override the log output format
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogFormat), 0)
			return strings.ToLower(output) == "json"
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logJSONYAMLPath: true,
		}),
	})

	// Override the log output stream
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogOutput), 0)
			return strings.ToLower(output) == "stdout"
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logSTDERRYAMLPath: true,
			logFileYAMLPath:   false,
		}),
	})

	// Override the log output to file
	overrides = append(overrides, cfgfile.ConditionalOverride{
		Check: func(cfg *common.Config) bool {
			aliasKeyPrefix := properties.GetAliasKeyPrefix()
			output, _ := cfg.String(fmt.Sprintf("%s.%s", aliasKeyPrefix, pathLogOutput), 0)
			if strings.ToLower(output) == "file" || strings.ToLower(output) == "both" {
				if strings.ToLower(output) == "both" {
					log.Warn("Traceability agent can only log to one output type, setting to file output")
				}
				return true
			}
			return false
		},
		Config: common.MustNewConfigFrom(map[string]interface{}{
			logFileYAMLPath:        true,
			logSTDERRYAMLPath:      false,
			logFilePermissionsPath: "0600",
		}),
	})

	return overrides
}
