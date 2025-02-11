package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

const envName = "mockEnv"

func TestDiscoveryCache_execute(t *testing.T) {
	tests := []struct {
		agentType       config.AgentType
		name            string
		svcCount        int
		managedAppCount int
		accessReqCount  int
		credCount       int
		withMigration   bool
		wt              *management.WatchTopic
	}{
		{
			name:            "should fetch resources based on the watch topic",
			agentType:       config.DiscoveryAgent,
			svcCount:        2,
			managedAppCount: 2,
			accessReqCount:  2,
			credCount:       2,
			wt:              mpWatchTopic,
		},
		{
			name:            "should fetch resources and perform a migration",
			agentType:       config.DiscoveryAgent,
			svcCount:        2,
			managedAppCount: 2,
			accessReqCount:  2,
			credCount:       2,
			withMigration:   true,
			wt:              mpWatchTopic,
		},
		{
			name:            "should fetch resources based on the watch topic with marketplace disabled",
			agentType:       config.TraceabilityAgent,
			svcCount:        2,
			managedAppCount: 0,
			accessReqCount:  0,
			credCount:       0,
			wt:              watchTopicNoMP,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := createCentralCfg("apicentral.axway.com", envName)
			cfg.AgentType = tc.agentType
			agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
			agent.cfg = cfg
			Initialize(cfg)
			scopeName := agent.cfg.GetEnvironmentName()
			c := &mockRIClient{
				svcs:        newAPIServices(scopeName),
				managedApps: newManagedApps(scopeName),
				manAppProfs: newManaAppProfs(scopeName),
				accessReqs:  newAccessReqs(scopeName),
				creds:       newCredentials(scopeName),
			}
			svcHandler := &mockHandler{
				kind: management.APIServiceGVK().Kind,
			}
			managedAppHandler := &mockHandler{
				kind: management.ManagedApplicationGVK().Kind,
			}
			managedAppProfHandler := &mockHandler{
				kind: management.ManagedApplicationProfileGVK().Kind,
			}
			accessReqHandler := &mockHandler{
				kind: management.AccessRequestGVK().Kind,
			}
			credHandler := &mockHandler{
				kind: management.CredentialGVK().Kind,
			}

			handlers := []handler.Handler{
				svcHandler,
				managedAppHandler,
				managedAppProfHandler,
				accessReqHandler,
				credHandler,
			}

			opts := []discoveryOpt{
				withAdditionalDiscoverFuncs(func() error {
					return nil
				}),
			}

			migration := &mockMigrator{mutex: sync.Mutex{}}
			if tc.withMigration {
				opts = append(opts, withMigration(migration))
			}

			dc := newDiscoveryCache(
				cfg,
				c,
				handlers,
				tc.wt,
				opts...,
			)

			err := dc.execute()
			assert.Nil(t, err)
			assert.Equal(t, tc.svcCount, svcHandler.count)
			assert.Equal(t, tc.managedAppCount, managedAppHandler.count)
			assert.Equal(t, tc.accessReqCount, accessReqHandler.count)
			assert.Equal(t, tc.credCount, credHandler.count)
			if tc.withMigration {
				assert.True(t, migration.called)
			} else {
				assert.False(t, migration.called)
			}
		})
	}

}

type mockHandler struct {
	count int
	err   error
	kind  string
}

func (m *mockHandler) Handle(_ context.Context, _ *proto.EventMeta, ri *apiv1.ResourceInstance) error {
	if m.kind != "" && ri.Kind != m.kind {
		return nil
	}
	m.count = m.count + 1
	return m.err
}

func newAPIServices(scope string) []*apiv1.ResourceInstance {
	svc1, _ := management.NewAPIService("svc1", scope).AsInstance()
	svc2, _ := management.NewAPIService("svc2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		svc1, svc2,
	}
}

func newManagedApps(scope string) []*apiv1.ResourceInstance {
	app1, _ := management.NewManagedApplication("app1", scope).AsInstance()
	app2, _ := management.NewManagedApplication("app2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		app1, app2,
	}
}

func newManaAppProfs(scope string) []*apiv1.ResourceInstance {
	map1, _ := management.NewManagedApplicationProfile("map1", scope).AsInstance()
	map2, _ := management.NewManagedApplicationProfile("map2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		map1, map2,
	}
}

func newAccessReqs(scope string) []*apiv1.ResourceInstance {
	ar1, _ := management.NewAccessRequest("ar1", scope).AsInstance()
	ar2, _ := management.NewAccessRequest("ar2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		ar1, ar2,
	}
}

func newCredentials(scope string) []*apiv1.ResourceInstance {
	cred1, _ := management.NewCredential("cred1", scope).AsInstance()
	cred2, _ := management.NewCredential("cred2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		cred1, cred2,
	}
}

type mockRIClient struct {
	svcs        []*apiv1.ResourceInstance
	managedApps []*apiv1.ResourceInstance
	manAppProfs []*apiv1.ResourceInstance
	accessReqs  []*apiv1.ResourceInstance
	creds       []*apiv1.ResourceInstance
	err         error
}

func (m mockRIClient) GetAPIV1ResourceInstances(_ map[string]string, URL string) ([]*apiv1.ResourceInstance, error) {
	fmt.Println(URL)
	if strings.Contains(URL, "apiservices") {
		return m.svcs, m.err
	} else if strings.Contains(URL, "managedapplications") {
		return m.managedApps, m.err
	} else if strings.Contains(URL, "managedapplicationprofiles") {
		return m.manAppProfs, m.err
	} else if strings.Contains(URL, "accessrequests") {
		return m.accessReqs, m.err
	} else if strings.Contains(URL, "credentials") {
		return m.creds, m.err
	}
	return make([]*apiv1.ResourceInstance, 0), m.err
}

type mockMigrator struct {
	called bool
	mutex  sync.Mutex
}

func (m *mockMigrator) Migrate(_ context.Context, ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.called = true
	return ri, nil
}

var mpWatchTopic = &management.WatchTopic{
	ResourceMeta: apiv1.ResourceMeta{},
	Owner:        nil,
	Spec: management.WatchTopicSpec{
		Description: "",
		Filters: []management.WatchTopicSpecFilters{
			{
				Group: "management",
				Kind:  management.APIServiceGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
			{
				Group: "management",
				Kind:  management.ManagedApplicationGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
			{
				Group: "management",
				Kind:  management.ManagedApplicationProfileGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
			{
				Group: "management",
				Kind:  management.AccessRequestGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
			{
				Group: "management",
				Kind:  management.CredentialGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
		},
	},
}

var watchTopicNoMP = &management.WatchTopic{
	ResourceMeta: apiv1.ResourceMeta{},
	Owner:        nil,
	Spec: management.WatchTopicSpec{
		Description: "",
		Filters: []management.WatchTopicSpecFilters{
			{
				Group: "management",
				Kind:  management.APIServiceGVK().Kind,
				Name:  "*",
				Scope: &management.WatchTopicSpecScope{
					Kind: "Environment",
					Name: envName,
				},
			},
		},
	},
}
