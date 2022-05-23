package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	catalog "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1alpha1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

const envName = "mockEnv"

func TestDiscoveryCache_execute(t *testing.T) {
	tests := []struct {
		agentType         config.AgentType
		name              string
		mpEnabled         bool
		svcCount          int
		instCount         int
		accessReqDefCount int
		categoryCount     int
		crdCount          int
		aclCount          int
		managedAppCount   int
		accessReqCount    int
		credCount         int
	}{
		{
			name:              "should fetch resources for a discovery agent",
			agentType:         config.DiscoveryAgent,
			mpEnabled:         true,
			svcCount:          2,
			instCount:         2,
			categoryCount:     2,
			accessReqDefCount: 2,
			crdCount:          2,
			aclCount:          2,
			managedAppCount:   2,
			accessReqCount:    2,
			credCount:         2,
		},
		{
			name:              "should fetch resources for a discovery agent with marketplace disabled",
			agentType:         config.DiscoveryAgent,
			mpEnabled:         false,
			svcCount:          2,
			instCount:         2,
			categoryCount:     2,
			accessReqDefCount: 0,
			crdCount:          0,
			aclCount:          2,
			managedAppCount:   0,
			accessReqCount:    0,
			credCount:         0,
		},
		{
			name:              "should fetch resources for a traceability agent",
			agentType:         config.TraceabilityAgent,
			mpEnabled:         true,
			svcCount:          2,
			instCount:         2,
			categoryCount:     0,
			accessReqDefCount: 0,
			crdCount:          0,
			aclCount:          0,
			managedAppCount:   2,
			accessReqCount:    2,
			credCount:         0,
		},
		{
			name:              "should fetch resources for a traceability agent with marketplace disabled",
			agentType:         config.TraceabilityAgent,
			mpEnabled:         false,
			svcCount:          2,
			instCount:         2,
			categoryCount:     0,
			accessReqDefCount: 0,
			crdCount:          0,
			aclCount:          0,
			managedAppCount:   0,
			accessReqCount:    0,
			credCount:         0,
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
				instances:   newAPISvcInstance(scopeName),
				categories:  newCategories(),
				ards:        newARDs(scopeName),
				crds:        newCRDs(scopeName),
				acls:        newAccessControlLists(scopeName),
				managedApps: newManagedApps(scopeName),
				accessReqs:  newAccessReqs(scopeName),
				creds:       newCredentials(scopeName),
			}
			svcHandler := &mockHandler{
				kind: mv1.APIServiceGVK().Kind,
			}
			instanceHandler := &mockHandler{
				kind: mv1.APIServiceInstanceGVK().Kind,
			}
			accessReqDefHandler := &mockHandler{
				kind: mv1.AccessRequestDefinitionGVK().Kind,
			}
			categoryHandler := &mockHandler{
				kind: catalog.CategoryGVK().Kind,
			}
			crdHandler := &mockHandler{
				kind: mv1.CredentialRequestDefinitionGVK().Kind,
			}
			aclHandler := &mockHandler{
				kind: mv1.AccessControlListGVK().Kind,
			}
			managedAppHandler := &mockHandler{
				kind: mv1.ManagedApplicationGVK().Kind,
			}
			accessReqHandler := &mockHandler{
				kind: mv1.AccessRequestGVK().Kind,
			}
			credHandler := &mockHandler{
				kind: mv1.CredentialGVK().Kind,
			}

			handlers := []handler.Handler{
				svcHandler,
				instanceHandler,
				categoryHandler,
				accessReqDefHandler,
				crdHandler,
				aclHandler,
				managedAppHandler,
				accessReqHandler,
				credHandler,
			}
			dc := newDiscoveryCache(
				cfg,
				c,
				handlers,
				withMpEnabled(tc.mpEnabled),
				withAdditionalDiscoverFuncs(func() error {
					return nil
				}),
			)

			err := dc.execute()
			assert.Nil(t, err)
			assert.Equal(t, tc.svcCount, svcHandler.count)
			assert.Equal(t, tc.instCount, instanceHandler.count)
			assert.Equal(t, tc.categoryCount, categoryHandler.count)
			assert.Equal(t, tc.accessReqDefCount, accessReqDefHandler.count)
			assert.Equal(t, tc.crdCount, crdHandler.count)
			assert.Equal(t, tc.aclCount, aclHandler.count)
			assert.Equal(t, tc.managedAppCount, managedAppHandler.count)
			assert.Equal(t, tc.accessReqCount, accessReqHandler.count)
			assert.Equal(t, tc.credCount, credHandler.count)
		})
	}

}

func TestDiscoveryCacheWithAdditionalDiscoveryFuncs(t *testing.T) {
	cfg := createCentralCfg("apicentral.axway.com", envName)
	cfg.AgentType = config.DiscoveryAgent
	agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
	agent.cfg = cfg
	Initialize(cfg)
	scopeName := agent.cfg.GetEnvironmentName()
	svcs := newAPIServices(scopeName)
	c := &mockRIClient{
		svcs: svcs,
	}
	h1 := &mockHandler{}
	handlers := []handler.Handler{
		h1,
	}
	dc := newDiscoveryCache(cfg, c, handlers)

	err := dc.handleAPISvc()
	assert.Nil(t, err)
	assert.Equal(t, len(svcs), h1.count)
}

func TestDiscoveryCache_handleMarketplaceResources(t *testing.T) {
	tests := []struct {
		name      string
		agentType config.AgentType
		credCount int
	}{
		{
			name:      "should fetch credentials for a discovery agent",
			credCount: 2,
			agentType: config.DiscoveryAgent,
		},
		{
			name:      "should not fetch credentials for a traceability agent",
			credCount: 0,
			agentType: config.TraceabilityAgent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := createCentralCfg("apicentral.axway.com", envName)
			agent.cacheManager = agentcache.NewAgentCacheManager(agent.cfg, false)
			agent.cfg = cfg
			Initialize(cfg)
			cfg.AgentType = tc.agentType

			scopeName := agent.cfg.GetEnvironmentName()
			managedApps := newManagedApps(scopeName)
			accessReqs := newAccessReqs(scopeName)
			creds := newCredentials(scopeName)
			c := &mockRIClient{
				creds:       creds,
				managedApps: managedApps,
				accessReqs:  accessReqs,
			}

			h1 := &mockHandler{
				kind: mv1.ManagedApplicationGVK().Kind,
			}
			h2 := &mockHandler{
				kind: mv1.AccessRequestGVK().Kind,
			}
			h3 := &mockHandler{
				kind: mv1.CredentialGVK().Kind,
			}
			handlers := []handler.Handler{h1, h2, h3}
			dc := newDiscoveryCache(cfg, c, handlers, withMpEnabled(true))

			err := dc.handleMarketplaceResources()
			assert.Nil(t, err)
			assert.Equal(t, len(managedApps), h1.count)
			assert.Equal(t, len(accessReqs), h2.count)
			assert.Equal(t, tc.credCount, h3.count)
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
	svc1, _ := mv1.NewAPIService("svc1", scope).AsInstance()
	svc2, _ := mv1.NewAPIService("svc2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		svc1, svc2,
	}
}

func newAPISvcInstance(scope string) []*apiv1.ResourceInstance {
	inst1, _ := mv1.NewAPIServiceInstance("inst1", scope).AsInstance()
	inst2, _ := mv1.NewAPIServiceInstance("inst2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		inst1, inst2,
	}
}

func newCategories() []*apiv1.ResourceInstance {
	cat1, _ := catalog.NewCategory("cat1").AsInstance()
	cat2, _ := catalog.NewCategory("cat2").AsInstance()
	return []*apiv1.ResourceInstance{
		cat1, cat2,
	}
}

func newARDs(scope string) []*apiv1.ResourceInstance {
	ard1, _ := mv1.NewAccessRequestDefinition("ard1", scope).AsInstance()
	ard2, _ := mv1.NewAccessRequestDefinition("ard2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		ard1, ard2,
	}
}

func newManagedApps(scope string) []*apiv1.ResourceInstance {
	app1, _ := mv1.NewManagedApplication("app1", scope).AsInstance()
	app2, _ := mv1.NewManagedApplication("app2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		app1, app2,
	}
}

func newCRDs(scope string) []*apiv1.ResourceInstance {
	crd1, _ := mv1.NewCredentialRequestDefinition("crd1", scope).AsInstance()
	crd2, _ := mv1.NewCredentialRequestDefinition("crd2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		crd1, crd2,
	}
}

func newAccessReqs(scope string) []*apiv1.ResourceInstance {
	ar1, _ := mv1.NewAccessRequest("ar1", scope).AsInstance()
	ar2, _ := mv1.NewAccessRequest("ar2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		ar1, ar2,
	}
}

func newCredentials(scope string) []*apiv1.ResourceInstance {
	cred1, _ := mv1.NewCredential("cred1", scope).AsInstance()
	cred2, _ := mv1.NewCredential("cred2", scope).AsInstance()
	return []*apiv1.ResourceInstance{
		cred1, cred2,
	}
}

func newAccessControlLists(scope string) []*apiv1.ResourceInstance {
	acl1, _ := mv1.NewAccessControlList("acl1", mv1.EnvironmentGVK().Kind, scope)
	inst1, _ := acl1.AsInstance()
	acl2, _ := mv1.NewAccessControlList("acl2", mv1.EnvironmentGVK().Kind, scope)
	inst2, _ := acl2.AsInstance()
	return []*apiv1.ResourceInstance{
		inst1, inst2,
	}
}

type mockRIClient struct {
	svcs        []*apiv1.ResourceInstance
	instances   []*apiv1.ResourceInstance
	categories  []*apiv1.ResourceInstance
	ards        []*apiv1.ResourceInstance
	crds        []*apiv1.ResourceInstance
	managedApps []*apiv1.ResourceInstance
	accessReqs  []*apiv1.ResourceInstance
	creds       []*apiv1.ResourceInstance
	acls        []*apiv1.ResourceInstance
	err         error
}

func (m mockRIClient) GetAPIV1ResourceInstancesWithPageSize(_ map[string]string, URL string, _ int) ([]*apiv1.ResourceInstance, error) {
	fmt.Println(URL)
	if strings.Contains(URL, "apiserviceinstances") {
		return m.instances, m.err
	} else if strings.Contains(URL, "apiservices") {
		return m.svcs, m.err
	} else if strings.Contains(URL, "categories") {
		return m.categories, m.err
	} else if strings.Contains(URL, "accessrequestdefinitions") {
		return m.ards, m.err
	} else if strings.Contains(URL, "credentialrequestdefinitions") {
		return m.crds, m.err
	} else if strings.Contains(URL, "managedapplications") {
		return m.managedApps, m.err
	} else if strings.Contains(URL, "accessrequests") {
		return m.accessReqs, m.err
	} else if strings.Contains(URL, "credentials") {
		return m.creds, m.err
	} else if strings.Contains(URL, "accesscontrollists") {
		return m.acls, m.err
	}
	return make([]*apiv1.ResourceInstance, 0), m.err
}

type mig struct {
	called bool
}

func (m *mig) Migrate(ri *apiv1.ResourceInstance) (*apiv1.ResourceInstance, error) {
	m.called = true
	return ri, nil
}
