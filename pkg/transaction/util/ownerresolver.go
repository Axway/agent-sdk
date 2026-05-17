package util

import (
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var logger = log.NewFieldLogger().
	WithPackage("sdk.transaction.util").
	WithComponent("ownerresolver")

func ResolveAPIOwner(apiExternalID string, cacheManager cache.Manager) *models.OwnerBlock {
	if cacheManager == nil {
		return &models.OwnerBlock{Type: "unknown"}
	}

	apiID := strings.TrimPrefix(apiExternalID, SummaryEventProxyIDPrefix)
	ri := cacheManager.GetAPIServiceWithAPIID(apiID)
	if ri == nil {
		logger.WithField("apiID", apiID).Trace("api service not found in cache, owner is unknown")
		return &models.OwnerBlock{Type: "unknown"}
	}

	svc := &management.APIService{}
	if err := svc.FromInstance(ri); err != nil {
		return &models.OwnerBlock{Type: "unknown"}
	}

	if svc.Owner == nil {
		logger.WithField("apiID", apiID).Trace("api service has no owner")
		return &models.OwnerBlock{Type: "none"}
	}

	if svc.Owner.Type == v1.TeamOwner {
		if svc.Owner.ID == "" {
			return &models.OwnerBlock{Type: "unknown"}
		}
		logger.WithField("apiID", apiID).WithField("teamGUID", svc.Owner.ID).Trace("resolved api owner")
		return &models.OwnerBlock{Type: "team", TeamGUID: svc.Owner.ID}
	}

	return &models.OwnerBlock{Type: "unknown"}
}

func ResolveAppOwner(accessRequest *management.AccessRequest) *models.OwnerBlock {
	if accessRequest == nil {
		return &models.OwnerBlock{Type: "unknown"}
	}

	owner := accessRequest.Owner
	if owner == nil {
		logger.WithField("accessRequestName", accessRequest.Name).Trace("access request has no owner")
		return &models.OwnerBlock{Type: "none"}
	}

	if owner.Type == v1.TeamOwner {
		if owner.ID == "" {
			return &models.OwnerBlock{Type: "unknown"}
		}
		logger.WithField("accessRequestName", accessRequest.Name).WithField("teamGUID", owner.ID).Trace("resolved app owner from access request")
		return &models.OwnerBlock{Type: "team", TeamGUID: owner.ID}
	}

	return &models.OwnerBlock{Type: "unknown"}
}

func ResolveAppOwnerFromManagedApp(manApp *v1.ResourceInstance) *models.OwnerBlock {
	if manApp == nil {
		return &models.OwnerBlock{Type: "unknown"}
	}

	app := &management.ManagedApplication{}
	if err := app.FromInstance(manApp); err != nil {
		return &models.OwnerBlock{Type: "unknown"}
	}

	owner := app.Marketplace.Resource.Owner
	if owner == nil {
		logger.WithField("appName", manApp.Name).Trace("managed application has no owner")
		return &models.OwnerBlock{Type: "none"}
	}

	if owner.Type == v1.TeamOwner {
		if owner.ID == "" {
			return &models.OwnerBlock{Type: "unknown"}
		}
		logger.WithField("appName", manApp.Name).WithField("teamGUID", owner.ID).Trace("resolved app owner from managed application")
		return &models.OwnerBlock{Type: "team", TeamGUID: owner.ID}
	}

	return &models.OwnerBlock{Type: "unknown"}
}
