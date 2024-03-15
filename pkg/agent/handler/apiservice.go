package handler

import (
	"context"
	"fmt"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type apiSvcHandler struct {
	agentCacheManager agentcache.Manager
	envName           string
}

// NewAPISvcHandler creates a Handler for API Services.
func NewAPISvcHandler(agentCacheManager agentcache.Manager, envName string) Handler {
	return &apiSvcHandler{
		agentCacheManager: agentCacheManager,
		envName:           envName,
	}
}

func (h *apiSvcHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.APIServiceGVK().Kind {
		return nil
	}

	if resource.Metadata.Scope.Name != h.envName {
		return nil
	}

	if action == proto.Event_CREATED || action == proto.Event_UPDATED {
		return h.agentCacheManager.AddAPIService(resource)
	}

	if action == proto.Event_DELETED {
		// external api id is not available on the resource for a delete event.
		// retrieve all keys and match the metadata id to see which resource needs to be deleted from the cache.
		keys := h.agentCacheManager.GetAPIServiceKeys()
		for _, k := range keys {
			svc := h.agentCacheManager.GetAPIServiceWithAPIID(k)

			if svc != nil && svc.Metadata.ID == resource.Metadata.ID {
				id, err := util.GetAgentDetailsValue(svc, definitions.AttrExternalAPIID)
				if err != nil {
					return fmt.Errorf(
						"%s not found on api service %s. %s", definitions.AttrExternalAPIID, resource.Name, err,
					)
				}
				h.agentCacheManager.DeleteAPIService(id)
				break
			}
		}
	}

	return nil
}
