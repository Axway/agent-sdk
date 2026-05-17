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

// ResolveAPIOwner returns an OwnerBlock for the API identified by apiExternalID.
// It strips the SummaryEventProxyIDPrefix before the cache lookup.
// Returns type "team" with TeamGUID when the APIService has an owner,
// "none" when the owner field is nil, and "unknown" on cache miss or empty GUID.
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

// ResolveAppOwner returns an OwnerBlock for the managed application resource instance.
// Returns type "team" with TeamGUID, "none" when owner is nil,
// or "unknown" when the GUID is missing or the input is nil.
func ResolveAppOwner(manApp *v1.ResourceInstance) *models.OwnerBlock {
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

	// Only TeamOwner is currently defined in v1.OwnerType; user ownership is not yet supported.
	if owner.Type == v1.TeamOwner {
		if owner.ID == "" {
			return &models.OwnerBlock{Type: "unknown"}
		}
		logger.WithField("appName", manApp.Name).WithField("teamGUID", owner.ID).Trace("resolved app owner")
		return &models.OwnerBlock{Type: "team", TeamGUID: owner.ID}
	}

	return &models.OwnerBlock{Type: "unknown"}
}
