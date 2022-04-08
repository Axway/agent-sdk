package log

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGlobalLoggerConfig(t *testing.T) {
	assert.Equal(t, STDOUT, GlobalLoggerConfig.output, "Expected default output to be STDOUT")
	assert.Equal(t, ".", GlobalLoggerConfig.path, "Expected default path to be current directory '.'")
	assert.Equal(t, logrus.InfoLevel, GlobalLoggerConfig.cfg.Level, "Expected default level to be info")
	assert.IsType(t, &logrus.JSONFormatter{}, GlobalLoggerConfig.cfg.Formatter, "Expected default formatter to be of JSON type")
}

func TestLoggerConfig(t *testing.T) {
	lc := LoggerConfig{}

	// Level
	err := lc.Level("debug1").Apply()
	assert.NotNil(t, err, "Expected an error for an invalid level type")
	lc.err = nil

	err = lc.Level("debug").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// Format
	err = lc.Format("fake").Apply()
	assert.NotNil(t, err, "Expected an error for an invalid format type")
	lc.err = nil

	err = lc.Format("line").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	err = lc.Format("json").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// Output
	err = lc.Output("fake").Apply()
	assert.NotNil(t, err, "Expected an error for an invalid output type")
	lc.err = nil

	err = lc.Output("Both").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// Filename
	err = lc.Filename("test").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// Path
	err = lc.Path("test").Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// MaxSize
	err = lc.MaxSize(0).Apply()
	assert.NotNil(t, err, "Expected an error for an invalid max size value")
	lc.err = nil

	err = lc.MaxSize(1048576).Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// MaxBackups
	err = lc.MaxBackups(-100).Apply()
	assert.NotNil(t, err, "Expected an error for an invalid max backups value")
	lc.err = nil

	err = lc.MaxBackups(100).Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil

	// MaxAge
	err = lc.MaxAge(-100).Apply()
	assert.NotNil(t, err, "Expected an error for an invalid max age value")
	lc.err = nil

	err = lc.MaxAge(100).Apply()
	assert.Nil(t, err, "Did not expect an error")
	lc.err = nil
}

func Test(t *testing.T) {
	log.SetLevel(logrus.DebugLevel)
	logger := NewFieldLogger()
	logger.WithField("abc", struct{ asdf string }{asdf: "asdf"}).Debug("debugging")
}
