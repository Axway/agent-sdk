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

// kindsProvider is implemented by target handlers that can report the resource Kind(s) they
// handle. Handler itself doesn't declare Kinds(), since most Handlers are registered directly by
// Kind and don't need to report it; target handlers registered here that don't implement it
// (e.g. externally registered custom handlers) simply don't contribute to the union below.
type kindsProvider interface {
	Kinds() []string
}

// Kinds returns the union of Kinds of the currently registered target handlers. It is read once,
// by newHandlers() at agent construction time, so target handlers must be registered (e.g. via
// RegisterProvisioner) before that point for their Kinds to be picked up.
func (h *StreamWatchProxyHandler) Kinds() []string {
	kindSet := map[string]struct{}{}
	for _, resourceHandler := range h.targetResourceHandlerMap {
		kp, ok := resourceHandler.(kindsProvider)
		if !ok {
			continue
		}
		for _, kind := range kp.Kinds() {
			kindSet[kind] = struct{}{}
		}
	}

	kinds := make([]string, 0, len(kindSet))
	for kind := range kindSet {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (h *StreamWatchProxyHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	for _, handler := range h.targetResourceHandlerMap {
		if handler.ShouldHandle(ctx, event) {
			return true
		}
	}

	return false
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
