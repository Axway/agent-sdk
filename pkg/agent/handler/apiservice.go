package handler

import (
	"context"

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

	log := getLoggerFromContext(ctx).
		WithComponent("apiServiceHandler").
		WithField("action", action).
		WithField("apiName", resource.Name).
		WithField("apiID", resource.Metadata.ID)

	id, err := util.GetAgentDetailsValue(resource, definitions.AttrExternalAPIID)
	if err != nil {
		log.WithError(err).Error("could not find the external API ID on the API Service")
	}
	log = log.WithField(definitions.AttrExternalAPIID, id)

	defer log.Trace("finished processing request")

	if action != proto.Event_DELETED {
		log.Debug("adding or updating api service in cache")
		err := h.agentCacheManager.AddAPIService(resource)
		if err != nil {
			log.WithError(err).Error("could not handle api service event")
		}
		return err
	}

	// remove the service from the cache by name
	return h.agentCacheManager.DeleteAPIService(resource.Name)
}
