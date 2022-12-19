package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

type ctxFields string

const (
	Kind               ctxFields = "kind"
	Name               ctxFields = "name"
	APIService         ctxFields = "apiService"
	APIServiceInstance ctxFields = "apiServiceInstance"
)

var allCtxFields = []ctxFields{Kind, Name}

// NewLoggerFromContext returns a FieldLogger for standard logging, and logp logging.
func NewLoggerFromContext(ctx context.Context) FieldLogger {
	entry := logrus.NewEntry(log)
	logger := &logger{
		entry: entry,
	}
	return UpdateLoggerWithContext(ctx, logger)
}

// UpdateLoggerWithContext returns a FieldLogger for standard logging, and logp logging.
func UpdateLoggerWithContext(ctx context.Context, logger FieldLogger) FieldLogger {
	for _, field := range allCtxFields {
		if v := ctx.Value(field); v != nil {
			logger.WithField(string(field), v)
		}
	}
	return logger
}
