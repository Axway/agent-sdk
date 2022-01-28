package handler

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

const (
	apiService         = "APIService"
	apiServiceInstance = "APIServiceInstance"
	category           = "Category"
	discoveryAgent     = "DiscoveryAgent"
	traceabilityAgent  = "TraceabilityAgent"
	governanceAgent    = "GovernanceAgent"
)

// Handler interface used by the EventListener to process events.
type Handler interface {
	// Handle receives the type of the event (add, update, delete), event metadata and the API Server resource, if it exists.
	Handle(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error
}

type apiSvcHandler struct {
	agentCacheManager agentcache.Manager
}

// NewAPISvcHandler creates a Handler for API Services.
func NewAPISvcHandler(agentCacheManager agentcache.Manager) Handler {
	return &apiSvcHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *apiSvcHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != apiService {
		return nil
	}

	id, ok := resource.Attributes[definitions.AttrExternalAPIID]
	if !ok {
		return fmt.Errorf("%s not found on ResourceClient api service %s", definitions.AttrExternalAPIID, resource.Name)
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.agentCacheManager.AddAPIService(resource)
	}

	if action == proto.Event_DELETED {
		return h.agentCacheManager.DeleteAPIService(id)
	}

	return nil
}

type instanceHandler struct {
	agentCacheManager agentcache.Manager
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(agentCacheManager agentcache.Manager) Handler {
	return &instanceHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (h *instanceHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != apiServiceInstance {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		h.agentCacheManager.AddAPIServiceInstance(resource)
	}

	if action == proto.Event_DELETED {
		key := resource.Metadata.ID
		return h.agentCacheManager.DeleteAPIServiceInstance(key)
	}

	return nil
}

type categoryHandler struct {
	agentCacheManager agentcache.Manager
}

// NewCategoryHandler creates a Handler for Categories.
func NewCategoryHandler(agentCacheManager agentcache.Manager) Handler {
	return &categoryHandler{
		agentCacheManager: agentCacheManager,
	}
}

func (c *categoryHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if resource.Kind != category {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		c.agentCacheManager.AddCategory(resource)
	}

	if action == proto.Event_DELETED {
		return c.agentCacheManager.DeleteCategory(resource.Name)
	}

	return nil
}

type agentResourceHandler struct {
	agentResourceManager resource.Manager
}

// NewAgentResourceHandler - creates a Handler for Agent resources
func NewAgentResourceHandler(agentResourceManager resource.Manager) Handler {
	return &agentResourceHandler{
		agentResourceManager: agentResourceManager,
	}
}

func (h *agentResourceHandler) Handle(action proto.Event_Type, _ *proto.EventMeta, resource *v1.ResourceInstance) error {
	if h.agentResourceManager != nil && action == proto.Event_UPDATED {
		kind := resource.Kind
		switch kind {
		case discoveryAgent:
			fallthrough
		case traceabilityAgent:
			fallthrough
		case governanceAgent:
			h.agentResourceManager.SetAgentResource(resource)
		}
	}
	return nil
}

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
func (h *StreamWatchProxyHandler) Handle(action proto.Event_Type, eventMetadata *proto.EventMeta, resource *v1.ResourceInstance) error {
	if h.targetResourceHandlerMap != nil {
		for _, handler := range h.targetResourceHandlerMap {
			err := handler.Handle(action, eventMetadata, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
