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

const (
	testOwnerTeamGUID1 = "team-guid-1"
	testOwnerTeamGUID2 = "team-guid-2"
	testOwnerUserGUID  = "user-guid-1"
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
		expected *models.Owner
	}{
		"nil cache manager returns unknown": {
			apiID:    "api-1",
			cache:    nil,
			expected: &models.Owner{Type: "unknown"},
		},
		"cache miss returns unknown": {
			apiID:    "not-in-cache",
			cache:    newCacheWithAPIService("api-1", &v1.Owner{Type: v1.TeamOwner, ID: "team-1"}),
			expected: &models.Owner{Type: "unknown"},
		},
		"api service with nil owner": {
			apiID:    "api-2",
			cache:    newCacheWithAPIService("api-2", nil),
			expected: &models.Owner{Type: "none"},
		},
		"api service team owner with GUID": {
			apiID:    "api-3",
			cache:    newCacheWithAPIService("api-3", &v1.Owner{Type: v1.TeamOwner, ID: "team-guid-3"}),
			expected: &models.Owner{Type: "team", TeamGUID: "team-guid-3"},
		},
		"api service team owner with empty GUID": {
			apiID:    "api-4",
			cache:    newCacheWithAPIService("api-4", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			expected: &models.Owner{Type: "unknown"},
		},
		"prefix stripped before lookup": {
			apiID:    SummaryEventProxyIDPrefix + "api-5",
			cache:    newCacheWithAPIService("api-5", &v1.Owner{Type: v1.TeamOwner, ID: "team-5"}),
			expected: &models.Owner{Type: "team", TeamGUID: "team-5"},
		},
		"api service x-private owner": {
			apiID:    "api-6",
			cache:    newCacheWithAPIService("api-6", &v1.Owner{Type: v1.TeamOwner, ID: "team-guid-6", User: &v1.OwnerUser{ID: testOwnerUserGUID}}),
			expected: &models.Owner{Type: "user", TeamGUID: "team-guid-6", UserGUID: testOwnerUserGUID},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveAPIOwner(tc.apiID, tc.cache))
		})
	}
}

func TestResolveAPIOwnerFromInstance(t *testing.T) {
	tests := map[string]struct {
		ri       *v1.ResourceInstance
		expected *models.Owner
	}{
		"nil resource instance returns unknown": {
			ri:       nil,
			expected: &models.Owner{Type: "unknown"},
		},
		"api service with nil owner returns none": {
			ri:       apiServiceRI("api-1", nil),
			expected: &models.Owner{Type: "none"},
		},
		"api service team owner with GUID": {
			ri:       apiServiceRI("api-2", &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID2}),
			expected: &models.Owner{Type: "team", TeamGUID: testOwnerTeamGUID2},
		},
		"api service team owner with empty GUID": {
			ri:       apiServiceRI("api-3", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			expected: &models.Owner{Type: "unknown"},
		},
		"api service x-private owner": {
			ri:       apiServiceRI("api-4", &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID1, User: &v1.OwnerUser{ID: testOwnerUserGUID}}),
			expected: &models.Owner{Type: "user", TeamGUID: testOwnerTeamGUID1, UserGUID: testOwnerUserGUID},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveAPIOwnerFromInstance(tc.ri))
		})
	}
}

func TestResolveProductOwner(t *testing.T) {
	tests := map[string]struct {
		ref      v1.EmbeddedReference
		expected *models.Owner
	}{
		"empty embedded reference": {
			ref:      v1.EmbeddedReference{},
			expected: &models.Owner{Type: "none"},
		},
		"product ref team owner with GUID": {
			ref:      v1.EmbeddedReference{Owner: &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID1}},
			expected: &models.Owner{Type: "team", TeamGUID: testOwnerTeamGUID1},
		},
		"product ref team owner with empty GUID": {
			ref:      v1.EmbeddedReference{Owner: &v1.Owner{Type: v1.TeamOwner, ID: ""}},
			expected: &models.Owner{Type: "unknown"},
		},
		"reference with name but no owner returns none": {
			ref:      v1.EmbeddedReference{Kind: "PublishedProduct", Name: "product-1"},
			expected: &models.Owner{Type: "none"},
		},
		"product ref x-private owner": {
			ref:      v1.EmbeddedReference{Owner: &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID1, User: &v1.OwnerUser{ID: testOwnerUserGUID}}},
			expected: &models.Owner{Type: "user", TeamGUID: testOwnerTeamGUID1, UserGUID: testOwnerUserGUID},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveProductOwner(tc.ref))
		})
	}
}

func TestResolveAppOwnerFromManagedApp(t *testing.T) {
	tests := map[string]struct {
		manApp   *v1.ResourceInstance
		expected *models.Owner
	}{
		"nil resource instance returns unknown": {
			manApp:   nil,
			expected: &models.Owner{Type: "unknown"},
		},
		"managed app with nil owner": {
			manApp:   managedAppRI("app-1", nil),
			expected: &models.Owner{Type: "none"},
		},
		"managed app team owner with GUID": {
			manApp:   managedAppRI("app-2", &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID2}),
			expected: &models.Owner{Type: "team", TeamGUID: testOwnerTeamGUID2},
		},
		"managed app team owner with empty GUID": {
			manApp:   managedAppRI("app-3", &v1.Owner{Type: v1.TeamOwner, ID: ""}),
			expected: &models.Owner{Type: "unknown"},
		},
		"managed app x-private owner": {
			manApp:   managedAppRI("app-4", &v1.Owner{Type: v1.TeamOwner, ID: testOwnerTeamGUID2, User: &v1.OwnerUser{ID: testOwnerUserGUID}}),
			expected: &models.Owner{Type: "user", TeamGUID: testOwnerTeamGUID2, UserGUID: testOwnerUserGUID},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ResolveAppOwnerFromManagedApp(tc.manApp))
		})
	}
}
