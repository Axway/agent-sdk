package handler

// ProxyHandler interface to represent the proxy resource handler.
type ProxyHandler interface {
	// RegisterTargetHandler adds the target handler
	RegisterTargetHandler(name string, resourceHandler Handler)
	// UnregisterTargetHandler removes the specified handler
	UnregisterTargetHandler(name string)
}

// StreamWatchProxyHandler - proxy handler for stream watch
type StreamWatchProxyHandler struct {
	targetResourceHandlerMap map[string][]Handler
}

// NewStreamWatchProxyHandler - creates a Handler to proxy target resource handler
func NewStreamWatchProxyHandler() *StreamWatchProxyHandler {
	return &StreamWatchProxyHandler{
		targetResourceHandlerMap: make(map[string][]Handler),
	}
}

// RegisterTargetHandler adds the target handler
func (h *StreamWatchProxyHandler) RegisterTargetHandler(name string, resourceHandler Handler) {
	handlers, ok := h.targetResourceHandlerMap[name]
	if !ok {
		handlers = make([]Handler, 0)
	}
	handlers = append(handlers, resourceHandler)

	h.targetResourceHandlerMap[name] = handlers
}

func (h *StreamWatchProxyHandler) GetHandlers() map[string][]Handler {
	return h.targetResourceHandlerMap
}

// UnregisterTargetHandler removes the specified handler
func (h *StreamWatchProxyHandler) UnregisterTargetHandler(name string) {
	delete(h.targetResourceHandlerMap, name)
}
