package handler

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
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
	// Handle receives the type of the event (add, update, delete), and the ResourceClient on API Server, if it exists.
	Handle(action proto.Event_Type, resource *v1.ResourceInstance) error
}

type apiSvcHandler struct {
	apis cache.Cache
}

// NewAPISvcHandler creates a Handler for API Services.
func NewAPISvcHandler(cache cache.Cache) Handler {
	return &apiSvcHandler{
		apis: cache,
	}
}

func (h *apiSvcHandler) Handle(action proto.Event_Type, resource *v1.ResourceInstance) error {
	if resource.Kind != apiService {
		return nil
	}

	id, ok := resource.Attributes[apic.AttrExternalAPIID]
	if !ok {
		return fmt.Errorf("%s not found on ResourceClient api service %s", apic.AttrExternalAPIID, resource.Name)
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		externalAPIName := resource.Attributes[apic.AttrExternalAPIName]
		primaryKey, ok := resource.Attributes[apic.AttrExternalAPIPrimaryKey]
		if !ok {
			return h.apis.SetWithSecondaryKey(id, externalAPIName, resource)
		}

		return h.apis.SetWithSecondaryKey(primaryKey, externalAPIName, resource)
	}

	if action == proto.Event_DELETED {
		return h.apis.Delete(id)
	}

	return nil
}

type instanceHandler struct {
	instances cache.Cache
}

// NewInstanceHandler creates a Handler for API Service Instances.
func NewInstanceHandler(cache cache.Cache) Handler {
	return &instanceHandler{
		instances: cache,
	}
}

func (h *instanceHandler) Handle(action proto.Event_Type, resource *v1.ResourceInstance) error {
	if resource.Kind != apiServiceInstance {
		return nil
	}

	key := resource.Metadata.ID
	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return h.instances.Set(key, resource)
	}

	if action == proto.Event_DELETED {
		return h.instances.Delete(key)
	}

	return nil
}

type categoryHandler struct {
	categories cache.Cache
}

// NewCategoryHandler creates a Handler for Categories.
func NewCategoryHandler(cache cache.Cache) Handler {
	return &categoryHandler{
		categories: cache,
	}
}

func (c *categoryHandler) Handle(action proto.Event_Type, resource *v1.ResourceInstance) error {
	if resource.Kind != category {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return c.categories.SetWithSecondaryKey(resource.Name, resource.Title, resource)
	}

	if action == proto.Event_DELETED {
		return c.categories.Delete(resource.Name)
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

func (h *agentResourceHandler) Handle(action proto.Event_Type, resource *v1.ResourceInstance) error {
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
	// UnRegisterTargetHandler removes the specified handler
	UnregisterTargetHandler(name string)
}

// StreamWatchProxyHandler - proxy handler for stream watch
type StreamWatchProxyHandler struct {
	targetResourceHandlerMap map[string]Handler
}

// NewProxyHandler - creates a Handler to proxy target resource handler
func NewStreamWatchProxyHandler() *StreamWatchProxyHandler {
	return &StreamWatchProxyHandler{
		targetResourceHandlerMap: make(map[string]Handler),
	}
}

func (h *StreamWatchProxyHandler) RegisterTargetHandler(name string, resourceHandler Handler) {
	h.targetResourceHandlerMap[name] = resourceHandler
}

func (h *StreamWatchProxyHandler) UnregisterTargetHandler(name string) {
	delete(h.targetResourceHandlerMap, name)
}

func (h *StreamWatchProxyHandler) Handle(action proto.Event_Type, resource *v1.ResourceInstance) error {
	if h.targetResourceHandlerMap != nil {
		for _, handler := range h.targetResourceHandlerMap {
			err := handler.Handle(action, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
