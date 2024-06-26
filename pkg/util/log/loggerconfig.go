package log

import (
	"flag"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
)

// GlobalLoggerConfig - is the default config of the logger
var GlobalLoggerConfig LoggerConfig

func init() {
	GlobalLoggerConfig = LoggerConfig{
		output: STDOUT,
		path:   ".",
		cfg: rotatefilehook.RotateFileConfig{
			Level: logrus.InfoLevel,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: time.RFC3339,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyMsg: "message",
				},
			},
		},
		metricCfg: rotatefilehook.RotateFileConfig{
			Level: logrus.InfoLevel,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: time.RFC3339,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyMsg: "message",
				},
			},
		},
		usageCfg: rotatefilehook.RotateFileConfig{
			Level: logrus.InfoLevel,
			Formatter: &logrus.JSONFormatter{
				TimestampFormat: time.RFC3339,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyMsg: "message",
				},
			},
		},
		initialized: false,
	}
}

// LoggerConfig - is a builder used to setup the logging for an agent
type LoggerConfig struct {
	err           error
	output        LoggingOutput
	path          string
	cfg           rotatefilehook.RotateFileConfig
	metricCfg     rotatefilehook.RotateFileConfig
	usageCfg      rotatefilehook.RotateFileConfig
	initialized   bool
	metricEnabled bool
	usageEnabled  bool
}

// Apply - applies the config changes to the logger
func (b *LoggerConfig) Apply() error {
	if b.err != nil {
		return b.err
	}

	// validate metric fields, if enabled
	if b.metricEnabled {
		if err := b.validateSize("log.metricfile.rotateeverybytes", b.metricCfg.MaxSize); err != nil {
			return err
		}
		if err := b.validate0orGreater("log.metricfile.keepfiles", b.metricCfg.MaxBackups); err != nil {
			return err
		}
		if err := b.validate0orGreater("log.metricfile.cleanbackupsevery", b.metricCfg.MaxAge); err != nil {
			return err
		}
		b.metricCfg.MaxSize = ConvertMaxSize(b.metricCfg.MaxSize)
	}

	if b.usageEnabled {
		if err := b.validateSize("log.usagefile.rotateeverybytes", b.usageCfg.MaxSize); err != nil {
			return err
		}
		if err := b.validate0orGreater("log.usagefile.keepfiles", b.usageCfg.MaxBackups); err != nil {
			return err
		}
		if err := b.validate0orGreater("log.usagefile.cleanbackupsevery", b.usageCfg.MaxAge); err != nil {
			return err
		}
		b.usageCfg.MaxSize = ConvertMaxSize(b.usageCfg.MaxSize)
	}

	// update the log logger
	log.SetFormatter(b.cfg.Formatter)
	log.SetLevel(b.cfg.Level)

	// update the logrus logger
	logrus.SetFormatter(b.cfg.Formatter)
	logrus.SetLevel(b.cfg.Level)

	// Set the stdout output for the log and logrus
	if b.output == STDOUT || b.output == Both {
		writer := io.Writer(os.Stdout)
		log.SetOutput(writer)
		logrus.SetOutput(writer)
	}

	if !b.initialized || isLogP {
		// Add the rotate file hook for log and logrus
		if b.output == File || b.output == Both {
			if b.path != "" {
				b.cfg.Filename = path.Join(b.path, b.cfg.Filename)
			}
			rotateFileHook, _ := rotatefilehook.NewRotateFileHook(b.cfg)
			log.AddHook(rotateFileHook)
			logrus.StandardLogger().AddHook(rotateFileHook)
		}

		isTest := flag.Lookup("test.v") != nil

		// skip metric log setup in unit tests
		if !isTest && b.metricEnabled {
			b.metricCfg.Filename = path.Join(b.path, "audit", b.metricCfg.Filename)
			rotateMetricHook, _ := rotatefilehook.NewRotateFileHook(b.metricCfg)
			metric.AddHook(rotateMetricHook)
			metric.SetOutput(io.Discard) // discard logging to stderr
		}

		if !isTest && b.usageEnabled {
			b.usageCfg.Filename = path.Join(b.path, "audit", b.usageCfg.Filename)
			rotateUsageHook, _ := rotatefilehook.NewRotateFileHook(b.usageCfg)
			usage.AddHook(rotateUsageHook)
			usage.SetOutput(io.Discard) // discard logging to stderr
		}

		// Set to initialized if this is not a test
		b.initialized = !isTest
	}

	return nil
}

// Level - sets the logger level
func (b *LoggerConfig) Level(level string) *LoggerConfig {
	if b.err == nil {
		lvl, err := logrus.ParseLevel(level)
		if err != nil {
			b.err = ErrInvalidLogConfig.FormatError("log.level", "trace, debug, info, warn, error")
		}
		b.cfg.Level = lvl
	}
	return b
}

// GetLevel - returns current log level
func (b *LoggerConfig) GetLevel() string {
	return b.cfg.Level.String()
}

// Format - sets the logger formatt
func (b *LoggerConfig) Format(format string) *LoggerConfig {
	if b.err == nil {
		switch strings.ToLower(format) {
		case loggingFormatStringMap[Line]:
			b.cfg.Formatter = &logrus.TextFormatter{
				TimestampFormat:  time.RFC3339,
				FullTimestamp:    true,
				PadLevelText:     true,
				QuoteEmptyFields: true,
				DisableColors:    true,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyMsg: "message",
				},
			}
		case loggingFormatStringMap[JSON]:
			b.cfg.Formatter = &logrus.JSONFormatter{
				TimestampFormat: time.RFC3339,
				FieldMap: logrus.FieldMap{
					logrus.FieldKeyMsg: "message",
				},
			}
		default:
			b.err = ErrInvalidLogConfig.FormatError("log.format", "json, line")
		}
	}
	return b
}

// Output - sets how the logs will be tracked
func (b *LoggerConfig) Output(output string) *LoggerConfig {
	if b.err == nil {
		if _, ok := stringLoggingOutputMap[strings.ToLower(output)]; !ok {
			b.err = ErrInvalidLogConfig.FormatError("log.output", "stdout, file, both")
		}
		b.output = stringLoggingOutputMap[output]
	}
	return b
}

// Filename -
func (b *LoggerConfig) Filename(filename string) *LoggerConfig {
	if b.err == nil {
		b.cfg.Filename = filename
	}
	return b
}

// Path -
func (b *LoggerConfig) Path(path string) *LoggerConfig {
	if b.err == nil {
		b.path = path
	}
	return b
}

func (b *LoggerConfig) Metrics(enabled bool) *LoggerConfig {
	if b.err == nil {
		b.metricEnabled = enabled
	}
	return b
}

func (b *LoggerConfig) Usage(enabled bool) *LoggerConfig {
	if b.err == nil {
		b.usageEnabled = enabled
	}
	return b
}

func (b *LoggerConfig) validateSize(path string, maxSize int) error {
	if maxSize < 1048576 {
		return ErrInvalidLogConfig.FormatError(path, "minimum of 1048576")
	}
	return nil
}

func (b *LoggerConfig) validate0orGreater(path string, maxBackups int) error {
	if maxBackups < 0 {
		return ErrInvalidLogConfig.FormatError(path, "0 or greater")
	}
	return nil
}

// MaxSize -
func (b *LoggerConfig) MaxSize(maxSize int) *LoggerConfig {
	if b.err == nil {
		b.err = b.validateSize("log.file.rotateeverybytes", maxSize)
		b.cfg.MaxSize = ConvertMaxSize(maxSize)
	}
	return b
}

// MaxBackups -
func (b *LoggerConfig) MaxBackups(maxBackups int) *LoggerConfig {
	if b.err == nil {
		b.err = b.validate0orGreater("log.file.keepfiles", maxBackups)
		b.cfg.MaxBackups = maxBackups
	}
	return b
}

// MaxAge -
func (b *LoggerConfig) MaxAge(maxAge int) *LoggerConfig {
	if b.err == nil {
		b.err = b.validate0orGreater("log.file.cleanbackupsevery", maxAge)
		b.cfg.MaxAge = maxAge
	}
	return b
}

// Filename -
func (b *LoggerConfig) MetricFilename(filename string) *LoggerConfig {
	if b.err == nil {
		b.metricCfg.Filename = filename
	}
	return b
}

// MaxMetricSize -
func (b *LoggerConfig) MaxMetricSize(maxSize int) *LoggerConfig {
	if b.err == nil {
		b.metricCfg.MaxSize = maxSize
	}
	return b
}

// MaxMetricBackups -
func (b *LoggerConfig) MaxMetricBackups(maxBackups int) *LoggerConfig {
	if b.err == nil {
		b.metricCfg.MaxBackups = maxBackups
	}
	return b
}

// MaxAge -
func (b *LoggerConfig) MaxMetricAge(maxAge int) *LoggerConfig {
	if b.err == nil {
		b.metricCfg.MaxAge = maxAge
	}
	return b
}

func (b *LoggerConfig) UsageFilename(filename string) *LoggerConfig {
	if b.err == nil {
		b.usageCfg.Filename = filename
	}
	return b
}

func (b *LoggerConfig) MaxUsageSize(maxSize int) *LoggerConfig {
	if b.err == nil {
		b.usageCfg.MaxSize = maxSize
	}
	return b
}

func (b *LoggerConfig) MaxUsageBackups(maxBackups int) *LoggerConfig {
	if b.err == nil {
		b.usageCfg.MaxBackups = maxBackups
	}
	return b
}

func (b *LoggerConfig) MaxUsageAge(maxAge int) *LoggerConfig {
	if b.err == nil {
		b.usageCfg.MaxAge = maxAge
	}
	return b
}

// ConvertMaxSize - takes max size in bytes and returns in megabytes for the rotate file hook
func ConvertMaxSize(maxSize int) int {
	return int(maxSize / 1024 / 1024)
}
