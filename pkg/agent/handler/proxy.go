package handler

import (
	"context"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
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

// Kinds returns nil since the set of target Kinds changes at runtime via RegisterTargetHandler /
// UnregisterTargetHandler, so this Handler must be checked for every event.
func (h *StreamWatchProxyHandler) Kinds() []string {
	return nil
}

func (h *StreamWatchProxyHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if h.targetResourceHandlerMap == nil {
		return false
	}
	if len(h.targetResourceHandlerMap) == 0 {
		return false
	}

	return true
}

// Handle receives the type of the event (add, update, delete), event metadata and updated API Server resource
func (h *StreamWatchProxyHandler) Handle(ctx context.Context, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error {
	event := NewEventFromResource(GetActionFromContext(ctx), eventMetadata, resource)
	for _, handler := range h.targetResourceHandlerMap {
		if !handler.ShouldHandle(ctx, event) {
			continue
		}
		err := handler.Handle(ctx, eventMetadata, resource)
		if err != nil {
			return err
		}
	}
	return nil
}
