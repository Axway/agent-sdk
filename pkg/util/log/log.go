package log

import (
	"github.com/sirupsen/logrus"
)

// Error -
func Error(args ...interface{}) {
	log.Error(args...)
}

// Errorf -
func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// Debug -
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Debugf -
func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Info -
func Info(args ...interface{}) {
	log.Info(args...)
}

// Infof -
func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Warn -
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Warnf -
func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

// SetLevel -
func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

// GetLevel -
func GetLevel() logrus.Level {
	return log.GetLevel()
}
