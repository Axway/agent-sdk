package log

import "github.com/sirupsen/logrus"

// Create a new instance of the logger
var log = logrus.New()

// Create a new instance of the metric logger
var metric = logrus.New()

var usage = logrus.New()

// LoggingOutput - Defines how the logger will log its output
type LoggingOutput int

const (
	// STDOUT - logs to the standard output of the agent process
	STDOUT LoggingOutput = iota
	// File - logs to a file, configure file settings for more options
	File
	// Both - logs to stdout and file, see the file confugration settings
	Both
)

// StringLoggingOutputMap - maps the string value representation of an output type to it's LoggingFormat value
var stringLoggingOutputMap = map[string]LoggingOutput{
	"stdout": STDOUT,
	"file":   File,
	"both":   Both,
}

// LoggingFormat - Defines the format of the logging output
type LoggingFormat int

const (
	// Line - logs individual lines, preceded by the timestamp and level
	Line LoggingFormat = iota + 1
	// JSON - logs in JSON format with the timestamp, level, and message all being separate fields
	JSON
)

// loggingFormatStringMap - maps the LoggingFormat type to it's string representation
var loggingFormatStringMap = map[LoggingFormat]string{
	Line: "line",
	JSON: "json",
}
