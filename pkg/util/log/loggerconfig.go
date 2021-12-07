package log

import (
	"flag"
	"io"
	"io/ioutil"
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
			Level:     logrus.InfoLevel,
			Formatter: &logrus.JSONFormatter{TimestampFormat: time.RFC3339},
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
	log.SetOutput(ioutil.Discard)

	// update the logrus logger
	logrus.SetFormatter(b.cfg.Formatter)
	logrus.SetLevel(b.cfg.Level)
	logrus.SetOutput(ioutil.Discard)

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
		// Set to initialized if this is not a test
		b.initialized = flag.Lookup("test.v") == nil
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

// Format - sets the logger formatt
func (b *LoggerConfig) Format(format string) *LoggerConfig {
	if b.err == nil {
		switch strings.ToLower(format) {
		case loggingFormatStringMap[Line]:
			b.cfg.Formatter = &logrus.TextFormatter{TimestampFormat: time.RFC3339}
		case loggingFormatStringMap[JSON]:
			b.cfg.Formatter = &logrus.JSONFormatter{TimestampFormat: time.RFC3339}
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

// MaxSize -
func (b *LoggerConfig) MaxSize(maxSize int) *LoggerConfig {
	if b.err == nil {
		if maxSize < 1048576 {
			b.err = ErrInvalidLogConfig.FormatError("log.file.rotateeverybytes", "minimum of 1048576")
		}
		b.cfg.MaxSize = int(float64(maxSize) / 1024 / 1024)
	}
	return b
}

// MaxBackups -
func (b *LoggerConfig) MaxBackups(maxBackups int) *LoggerConfig {
	if b.err == nil {
		if maxBackups < 0 {
			b.err = ErrInvalidLogConfig.FormatError("log.file.keepfiles", "0 or greater")
		}
		b.cfg.MaxBackups = maxBackups
	}
	return b
}

// MaxAge -
func (b *LoggerConfig) MaxAge(maxAge int) *LoggerConfig {
	if b.err == nil {
		if maxAge < 0 {
			b.err = ErrInvalidLogConfig.FormatError("log.file.cleanbackupsevery", "0 or greater")
		}
		b.cfg.MaxAge = maxAge
	}
	return b
}
