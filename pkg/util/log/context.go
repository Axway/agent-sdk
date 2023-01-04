package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

type ContextField string

const (
	KindCtx ContextField = "kind"
	NameCtx ContextField = "name"
)

var allCtxFields = map[ContextField]struct{}{KindCtx: {}, NameCtx: {}}

func RegisterContextField(ctxFields ...ContextField) {
	for _, ctxField := range ctxFields {
		allCtxFields[ctxField] = struct{}{}
	}
}

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
	for field := range allCtxFields {
		if v := ctx.Value(field); v != nil {
			logger.WithField(string(field), v)
		}
	}
	return logger
}
