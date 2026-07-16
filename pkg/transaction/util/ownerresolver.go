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

func ResolveAPIOwner(apiExternalID string, cacheManager cache.Manager) *models.Owner {
	if cacheManager == nil {
		return &models.Owner{Type: "unknown"}
	}

	apiID := strings.TrimPrefix(apiExternalID, SummaryEventProxyIDPrefix)
	ri := cacheManager.GetAPIServiceWithAPIID(apiID)
	if ri == nil {
		logger.WithField("apiID", apiID).Trace("api service not found in cache, owner is unknown")
		return &models.Owner{Type: "unknown"}
	}

	return ResolveAPIOwnerFromInstance(ri)
}

func ResolveAPIOwnerFromInstance(ri *v1.ResourceInstance) *models.Owner {
	if ri == nil {
		return &models.Owner{Type: "unknown"}
	}

	svc := &management.APIService{}
	if err := svc.FromInstance(ri); err != nil {
		return &models.Owner{Type: "unknown"}
	}

	if svc.Owner == nil {
		logger.WithField("apiName", ri.Name).Trace("api service has no owner")
		return &models.Owner{Type: "none"}
	}

	if svc.Owner.Type == v1.TeamOwner {
		if svc.Owner.ID == "" {
			return &models.Owner{Type: "unknown"}
		}
		if svc.Owner.User != nil && svc.Owner.User.ID != "" {
			logger.WithField("apiName", ri.Name).WithField("userGUID", svc.Owner.User.ID).Trace("resolved api owner as user (x-private team)")
			return &models.Owner{Type: "user", TeamGUID: svc.Owner.ID, UserGUID: svc.Owner.User.ID}
		}
		logger.WithField("apiName", ri.Name).WithField("teamGUID", svc.Owner.ID).Trace("resolved api owner")
		return &models.Owner{Type: "team", TeamGUID: svc.Owner.ID}
	}

	return &models.Owner{Type: "unknown"}
}

func ResolveProductOwner(ref v1.EmbeddedReference) *models.Owner {
	if ref.Owner == nil {
		return &models.Owner{Type: "none"}
	}
	if ref.Owner.Type == v1.TeamOwner {
		if ref.Owner.ID == "" {
			return &models.Owner{Type: "unknown"}
		}
		if ref.Owner.User != nil && ref.Owner.User.ID != "" {
			return &models.Owner{Type: "user", TeamGUID: ref.Owner.ID, UserGUID: ref.Owner.User.ID}
		}
		return &models.Owner{Type: "team", TeamGUID: ref.Owner.ID}
	}
	return &models.Owner{Type: "unknown"}
}

func ResolveAppOwnerFromManagedApp(manApp *v1.ResourceInstance) *models.Owner {
	if manApp == nil {
		return &models.Owner{Type: "unknown"}
	}

	app := &management.ManagedApplication{}
	if err := app.FromInstance(manApp); err != nil {
		return &models.Owner{Type: "unknown"}
	}

	owner := app.Marketplace.Resource.Owner
	if owner == nil {
		logger.WithField("appName", manApp.Name).Trace("managed application has no owner")
		return &models.Owner{Type: "none"}
	}

	if owner.Type == v1.TeamOwner {
		if owner.ID == "" {
			return &models.Owner{Type: "unknown"}
		}
		if owner.User != nil && owner.User.ID != "" {
			logger.WithField("appName", manApp.Name).WithField("userGUID", owner.User.ID).Trace("resolved app owner as user (x-private team)")
			return &models.Owner{Type: "user", TeamGUID: owner.ID, UserGUID: owner.User.ID}
		}
		logger.WithField("appName", manApp.Name).WithField("teamGUID", owner.ID).Trace("resolved app owner from managed application")
		return &models.Owner{Type: "team", TeamGUID: owner.ID}
	}

	return &models.Owner{Type: "unknown"}
}
