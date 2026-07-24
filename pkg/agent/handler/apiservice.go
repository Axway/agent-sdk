package handler

import (
	"context"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
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

func (h *apiSvcHandler) ShouldHandle(ctx context.Context, event *proto.Event) bool {
	if event.Payload.Metadata.Scope.Name != h.envName || event.Payload.Metadata.Scope.Kind != management.EnvironmentGVK().Kind {
		return false
	}
	return true
}

// HandleCache adds the API Service to the cache during discoveryCache's bulk rebuild.
func (h *apiSvcHandler) HandleCache(resource *apiv1.ResourceInstance) error {
	return h.agentCacheManager.AddAPIService(resource)
}

// GetAPIServerFields returns the fields needed to process the given event. A subresource update
// only needs a restricted fetch if the resource is already cached (looked up by name - the
// APIService cache is keyed by external API ID/name from x-agent-details, not metadata.id), so
// Handle can merge the updated subresource onto it; otherwise the full resource is needed to
// populate the cache from scratch, so no restriction is returned.
func (h *apiSvcHandler) GetAPIServerFields(ctx context.Context, event *proto.Event) []string {
	if event.Metadata.GetSubresource() == "" {
		return nil
	}
	if existing := h.agentCacheManager.GetAPIServiceWithName(event.Payload.Name); existing == nil {
		return nil
	}
	return []string{"name", "metadata.id", event.Metadata.GetSubresource()}
}

func (h *apiSvcHandler) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)

	log := getLoggerFromContext(ctx).
		WithComponent("apiServiceHandler").
		WithField("action", action).
		WithField("resource", resource.Name).
		WithField("resourceID", resource.Metadata.ID)

	if action == proto.Event_DELETED {
		// remove the service from the cache by name
		return h.agentCacheManager.DeleteAPIService(resource.Name)
	}

	if meta != nil && meta.Subresource != "" {
		existing := h.agentCacheManager.GetAPIServiceWithName(resource.Name)
		if existing == nil {
			// GetAPIServerFields didn't restrict fields in this case, so resource is already the
			// full fetch - cache it directly.
			if err := h.agentCacheManager.AddAPIService(resource); err != nil {
				log.WithError(err).Error("could not handle api service event")
			}
			return nil
		}
		if v := resource.GetSubResource(meta.Subresource); v != nil {
			existing.SetSubResource(meta.Subresource, v)
		}
		if err := h.agentCacheManager.AddAPIService(existing); err != nil {
			log.WithError(err).Error("could not handle api service subresource update")
		}
		return nil
	}

	id, err := util.GetAgentDetailsValue(resource, definitions.AttrExternalAPIID)
	if err != nil {
		log.WithError(err).Error("could not find the external API ID on the API Service")
	}
	log = log.WithField("apiID", id)

	defer log.Trace("finished processing request")

	log.Debug("adding or updating api service in cache")
	err = h.agentCacheManager.AddAPIService(resource)
	if err != nil {
		log.WithError(err).Error("could not handle api service event")
	}
	return err

}
