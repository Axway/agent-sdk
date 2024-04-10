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
		initialized: false,
	}
}

// LoggerConfig - is a builder used to setup the logging for an agent
type LoggerConfig struct {
	err         error
	output      LoggingOutput
	path        string
	cfg         rotatefilehook.RotateFileConfig
	metricCfg   rotatefilehook.RotateFileConfig
	initialized bool
}

// Apply - applies the config changes to the logger
func (b *LoggerConfig) Apply() error {
	if b.err != nil {
		return b.err
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
		if !isTest {
			b.metricCfg.Filename = path.Join(b.path, "metrics", b.metricCfg.Filename)
			rotateMetricHook, _ := rotatefilehook.NewRotateFileHook(b.metricCfg)
			metric.AddHook(rotateMetricHook)
			metric.SetOutput(io.Discard) // discard logging to stderr
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
		b.cfg.MaxSize = int(float64(maxSize) / 1024 / 1024)
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
		b.err = b.validateSize("log.metricfile.rotateeverybytes", maxSize)
		b.metricCfg.MaxSize = maxSize
	}
	return b
}

// MaxMetricBackups -
func (b *LoggerConfig) MaxMetricBackups(maxBackups int) *LoggerConfig {
	if b.err == nil {
		b.err = b.validate0orGreater("log.metricfile.keepfiles", maxBackups)
		b.metricCfg.MaxBackups = maxBackups
	}
	return b
}

// MaxAge -
func (b *LoggerConfig) MaxMetricAge(maxAge int) *LoggerConfig {
	if b.err == nil {
		b.err = b.validate0orGreater("log.metricfile.cleanbackupsevery", maxAge)
		b.metricCfg.MaxAge = maxAge
	}
	return b
}
