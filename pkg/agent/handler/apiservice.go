package handler

import (
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

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
	if resource.Kind != mv1.APIServiceGVK().Kind {
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
