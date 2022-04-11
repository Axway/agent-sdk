package handler

import (
	"context"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	corelog "github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/sirupsen/logrus"
)

// ProxyHandler interface to represent the proxy resource handler.
type ProxyHandler interface {
	// RegisterTargetHandler adds the target handler
	RegisterTargetHandler(name string, resourceHandler Handler)
	// UnregisterTargetHandler removes the specified handler
	UnregisterTargetHandler(name string)
}

// StreamWatchProxyHandler - proxy handler for stream watch
type StreamWatchProxyHandler struct {
	targetResourceHandlerMap map[string]Handler
}

// NewStreamWatchProxyHandler - creates a Handler to proxy target resource handler
func NewStreamWatchProxyHandler() *StreamWatchProxyHandler {
	return &StreamWatchProxyHandler{
		targetResourceHandlerMap: make(map[string]Handler),
	}
}

// RegisterTargetHandler adds the target handler
func (h *StreamWatchProxyHandler) RegisterTargetHandler(name string, resourceHandler Handler) {
	h.targetResourceHandlerMap[name] = resourceHandler
}

// UnregisterTargetHandler removes the specified handler
func (h *StreamWatchProxyHandler) UnregisterTargetHandler(name string) {
	delete(h.targetResourceHandlerMap, name)
}

// Handle receives the type of the event (add, update, delete), event metadata and updated API Server resource
func (h *StreamWatchProxyHandler) Handle(ctx context.Context, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error {
	if h.targetResourceHandlerMap != nil {
		for _, handler := range h.targetResourceHandlerMap {
			err := handler.Handle(ctx, eventMetadata, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// NewEventContext - create a context for the new event
func NewEventContext(action proto.Event_Type, eventMetadata *proto.EventMeta, kind, name string) context.Context {
	logger := fieldLogger.WithFields(
		logrus.Fields{
			actionField: action.String(),
			typeField:   kind,
			nameField:   name,
		},
	)
	if eventMetadata != nil {
		logger = logger.
			WithField(sequenceIDField, eventMetadata.SequenceID)
	}
	return setActionInContext(setLoggerInContext(context.Background(), logger), action)
}

func setLoggerInContext(ctx context.Context, logger corelog.FieldLogger) context.Context {
	return context.WithValue(ctx, ctxLogger, logger)
}

func setActionInContext(ctx context.Context, action proto.Event_Type) context.Context {
	return context.WithValue(ctx, ctxAction, action)
}

// GetLoggerFromContext- returns the field logger that is part of the context
func GetLoggerFromContext(ctx context.Context) corelog.FieldLogger {
	return ctx.Value(ctxLogger).(corelog.FieldLogger)
}

func getActionFromContext(ctx context.Context) proto.Event_Type {
	return ctx.Value(ctxAction).(proto.Event_Type)
}
