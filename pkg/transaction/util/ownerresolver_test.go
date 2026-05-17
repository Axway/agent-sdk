package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func apiServiceRI(apiID string, owner *v1.Owner) *v1.ResourceInstance {
	svc := management.NewAPIService("svc-"+apiID, "env1")
	svc.SubResources = map[string]interface{}{
		"x-agent-details": map[string]interface{}{
			"externalAPIID": apiID,
		},
	}
	svc.Owner = owner
	ri, _ := svc.AsInstance()
	return ri
}

func managedAppRI(name string, owner *v1.Owner) *v1.ResourceInstance {
	app := management.NewManagedApplication(name, "env1")
	app.Marketplace = management.ManagedApplicationMarketplace{
		Name: "mp1",
		Resource: management.ManagedApplicationMarketplaceResource{
			Owner: owner,
		},
	}
	ri, _ := app.AsInstance()
	return ri
}

func newCacheWithAPIService(apiID string, owner *v1.Owner) agentcache.Manager {
	m := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	m.AddAPIService(apiServiceRI(apiID, owner))
	return m
}

func TestResolveAPIOwner(t *testing.T) {
	tests := map[string]struct {
		apiID    string
		cache    agentcache.Manager
		expected *models.OwnerBlock
	}{
		"nil cache manager returns unknown": {
			apiID:    "api-1",
			cache:    nil,
			expected: &models.OwnerBlock{Type: "unknown"},
		},
		"cache miss returns unknown": {
			apiID:    "not-in-cache",
			cache:    newCacheWithAPIService("api-1", &v1.Owner{Type: v1.TeamOwner, ID: "team-1"}),
			expected: &models.OwnerBlock{Type: "unknown"},
		},
		"nil owner returns none": {
			apiID:    "api-2",
			cache:    newCacheWithAPIService("api-2", nil),
			expected: &models.OwnerBlock{Type: "none"},
		},
		"team owner with GUID returns team block": {
			apiID:    "api-3",
			cache:    newCacheWithAPIService("api-3", &v1.Owner{Type: v1.TeamOwner, ID: "team-guid-3"}),
			expected: &models.OwnerBlock{Type: "team", TeamGUID: "team-guid-3"},
		},
		"team owner with empty GUID returns unknown": {
			apiID:    "api-4",
			cache:    newCacheWithAPIService("api-4", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			expected: &models.OwnerBlock{Type: "unknown"},
		},
		"prefix stripped before lookup": {
			apiID:    SummaryEventProxyIDPrefix + "api-5",
			cache:    newCacheWithAPIService("api-5", &v1.Owner{Type: v1.TeamOwner, ID: "team-5"}),
			expected: &models.OwnerBlock{Type: "team", TeamGUID: "team-5"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveAPIOwner(tc.apiID, tc.cache))
		})
	}
}

func TestResolveAppOwner(t *testing.T) {
	tests := map[string]struct {
		manApp   *v1.ResourceInstance
		expected *models.OwnerBlock
	}{
		"nil resource instance returns unknown": {
			manApp:   nil,
			expected: &models.OwnerBlock{Type: "unknown"},
		},
		"nil owner returns none": {
			manApp:   managedAppRI("app-1", nil),
			expected: &models.OwnerBlock{Type: "none"},
		},
		"team owner with GUID returns team block": {
			manApp:   managedAppRI("app-2", &v1.Owner{Type: v1.TeamOwner, ID: "team-guid-2"}),
			expected: &models.OwnerBlock{Type: "team", TeamGUID: "team-guid-2"},
		},
		"team owner with empty GUID returns unknown": {
			manApp:   managedAppRI("app-3", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			expected: &models.OwnerBlock{Type: "unknown"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveAppOwner(tc.manApp))
		})
	}
}
