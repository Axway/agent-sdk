package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
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

func (h *apiSvcHandler) Kinds() []string {
	return []string{management.APIServiceGVK().Kind}
}

func (h *apiSvcHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Kind != management.APIServiceGVK().Kind {
		return false
	}

	if event.Payload.Metadata.Scope.Name != h.envName || event.Payload.Metadata.Scope.Kind != management.EnvironmentGVK().Kind {
		return false
	}

	if action := GetActionFromContext(ctx); action != proto.Event_DELETED {
		return true
	}

	existing, _ := h.agentCacheManager.GetAPIServiceCache().Get(event.Payload.Name)
	if existing == nil {
		existing, _ = h.agentCacheManager.GetAPIServiceCache().GetBySecondaryKey(event.Payload.Name)
	}
	existingSvc, ok := existing.(*apiv1.ResourceInstance)
	if !ok {
		log.Trace("invalid resource type in cache, skipping delete")
		return false
	}

	if existingSvc.Metadata.ID != event.Payload.Metadata.Id {
		log.Trace("resource id mismatch, skipping delete")
		return false
	}
	return true
}

func (h *apiSvcHandler) Handle(ctx context.Context, _ *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	log := getLoggerFromContext(ctx).
		WithComponent("apiServiceHandler").
		WithField("action", action).
		WithField("resource", resource.Name).
		WithField("resourceID", resource.Metadata.ID)

	id, err := util.GetAgentDetailsValue(resource, definitions.AttrExternalAPIID)
	if err != nil {
		log.WithError(err).Error("could not find the external API ID on the API Service")
	}
	log = log.WithField("apiID", id)

	defer log.Trace("finished processing request")

	if action == proto.Event_DELETED {
		// remove the service from the cache by name
		return h.agentCacheManager.DeleteAPIService(resource.Name)
	}

	log.Debug("adding or updating api service in cache")
	err = h.agentCacheManager.AddAPIService(resource)
	if err != nil {
		log.WithError(err).Error("could not handle api service event")
	}
	return err

}
