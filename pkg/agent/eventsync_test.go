package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/handler"
	"github.com/Axway/agent-sdk/pkg/agent/resource"
	"github.com/Axway/agent-sdk/pkg/api"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestEventSync(t *testing.T) {
	tests := map[string]struct {
		cfgFn         func(cfg *config.CentralConfiguration)
		agentFeatsCfg *config.AgentFeaturesConfiguration
	}{
		"poll mode": {},
		"stream mode": {
			cfgFn: func(cfg *config.CentralConfiguration) {
				cfg.GRPCCfg = config.GRPCConfig{Enabled: true, Insecure: true}
			},
			agentFeatsCfg: &config.AgentFeaturesConfiguration{
				ConnectToCentral:     true,
				ProcessSystemSignals: true,
				VersionChecker:       false,
				PersistCache:         true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := createCentralCfg("https://abc.com", "mockenv")
			if tc.cfgFn != nil {
				tc.cfgFn(cfg)
			}
			err := Initialize(cfg)
			if tc.agentFeatsCfg != nil {
				agent.agentFeaturesCfg = tc.agentFeatsCfg
			}

			cfg.AgentName = "Test-DA"
			agentRes := createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "")

			mc := &mock.Client{
				ExecuteAPIMock: func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
					if method == api.PUT {
						return buffer, nil
					}
					return json.Marshal(agentRes)
				},
				GetResourceMock: func(url string) (*apiv1.ResourceInstance, error) {
					if strings.Contains(url, "/discoveryagents") {
						return agentRes, nil
					}
					wt := management.NewWatchTopic("mock-wt")
					ri, err := wt.AsInstance()
					return ri, err
				},
			}

			m, _ := resource.NewAgentResourceManager(cfg, mc, nil)
			agent.agentResourceManager = m
			InitializeForTest(mc)
			assert.Nil(t, err)

			es, err := newEventSync()
			assert.Nil(t, err)
			assert.NotNil(t, es.watchTopic)
			assert.NotNil(t, es.discoveryCache)
			assert.NotNil(t, es.sequence)
			assert.NotNil(t, es.harvester)

			es.harvester = &mockHarvester{}
			err = es.SyncCache()
			assert.Nil(t, err)
		})
	}
}

type mockHarvester struct {
	err error
}

func (m mockHarvester) EventCatchUp(ctx context.Context, link string, events chan *proto.Event) error {
	return m.err
}

func (m mockHarvester) ReceiveSyncEvents(ctx context.Context, topicSelfLink string, sequenceID int64, eventCh chan *proto.Event) (int64, error) {
	return 1, m.err
}

func TestInitCache_targetedRebuild(t *testing.T) {
	cfg := createCentralCfg("https://abc.com", "mockenv")
	err := Initialize(cfg)
	assert.Nil(t, err)
	cfg.AgentName = "Test-DA"
	agentRes := createDiscoveryAgentRes("111", "Test-DA", "test-dataplane", "")

	mc := &mock.Client{
		ExecuteAPIMock: func(method, url string, queryParam map[string]string, buffer []byte) ([]byte, error) {
			if method == api.PUT {
				return buffer, nil
			}
			return json.Marshal(agentRes)
		},
		GetResourceMock: func(url string) (*apiv1.ResourceInstance, error) {
			if strings.Contains(url, "/discoveryagents") {
				return agentRes, nil
			}
			wt := management.NewWatchTopic("mock-wt")
			ri, err := wt.AsInstance()
			return ri, err
		},
	}

	m, _ := resource.NewAgentResourceManager(cfg, mc, nil)
	agent.agentResourceManager = m
	InitializeForTest(mc)

	scopeName := cfg.Environment

	apiSvcFilter := management.WatchTopicSpecFilters{
		Group: management.APIServiceGVK().Group,
		Kind:  management.APIServiceGVK().Kind,
		Name:  "*",
		Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
	}
	instFilter := management.WatchTopicSpecFilters{
		Group: management.APIServiceInstanceGVK().Group,
		Kind:  management.APIServiceInstanceGVK().Kind,
		Name:  "*",
		Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
	}

	tests := map[string]struct {
		setup         func()
		wtFilters     []management.WatchTopicSpecFilters
		failedFilters []management.WatchTopicSpecFilters
		makeClient    func() resourceClient
		expectedCount int
		verify        func(t *testing.T)
	}{
		"targeted rebuild flushes only failed kind and rebuilds it": {
			setup: func() {
				inst1, _ := management.NewAPIServiceInstance("inst1", scopeName).AsInstance()
				agent.cacheManager.AddAPIServiceInstance(inst1)
			},
			wtFilters:     []management.WatchTopicSpecFilters{apiSvcFilter, instFilter},
			failedFilters: []management.WatchTopicSpecFilters{apiSvcFilter},
			makeClient: func() resourceClient {
				return &mockRIClient{svcs: newAPIServices(scopeName)}
			},
			expectedCount: 2,
			verify: func(t *testing.T) {
				// Instance cache was NOT flushed — the instance we added before should still be there
				cachedInst, instErr := agent.cacheManager.GetAPIServiceInstanceByName("inst1")
				assert.Nil(t, instErr)
				assert.NotNil(t, cachedInst)
			},
		},
		"targeted rebuild failure falls back to full rebuild": {
			wtFilters:     []management.WatchTopicSpecFilters{apiSvcFilter},
			failedFilters: []management.WatchTopicSpecFilters{apiSvcFilter},
			makeClient: func() resourceClient {
				callCount := 0
				return &failThenSucceedClient{
					failCount: 1,
					callCount: &callCount,
					svcs:      newAPIServices(scopeName),
				}
			},
			expectedCount: 2,
		},
		"no filters - full rebuild": {
			wtFilters: []management.WatchTopicSpecFilters{apiSvcFilter},
			makeClient: func() resourceClient {
				return &mockRIClient{svcs: newAPIServices(scopeName)}
			},
			expectedCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			agent.cacheManager = agentcache.NewAgentCacheManager(cfg, false)
			if tc.setup != nil {
				tc.setup()
			}

			svcHandler := &mockHandler{kind: management.APIServiceGVK().Kind}
			wt := &management.WatchTopic{Spec: management.WatchTopicSpec{Filters: tc.wtFilters}}
			dc := newDiscoveryCache(cfg, tc.makeClient(), []handler.Handler{svcHandler}, wt)

			es := &EventSync{
				watchTopic:     wt,
				harvester:      &mockHarvester{},
				discoveryCache: dc,
				sequence:       &mockSequence{},
			}

			err := es.initCache(tc.failedFilters...)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedCount, svcHandler.count)

			if tc.verify != nil {
				tc.verify(t)
			}
		})
	}
}

// failThenSucceedClient fails the first N calls, then returns normal results.
type failThenSucceedClient struct {
	failCount int
	callCount *int
	svcs      []*apiv1.ResourceInstance
}

func (f *failThenSucceedClient) GetAPIV1ResourceInstances(_ map[string]string, URL string) ([]*apiv1.ResourceInstance, error) {
	*f.callCount++
	if *f.callCount <= f.failCount {
		return nil, fmt.Errorf("simulated fetch error")
	}
	if strings.Contains(URL, "apiservices") {
		return f.svcs, nil
	}
	return []*apiv1.ResourceInstance{}, nil
}

type mockSequence struct {
	id int64
}

func (m *mockSequence) SetSequence(id int64) {
	m.id = id
}

func (m *mockSequence) GetSequence() int64 {
	return m.id
}

// mockListenerPauser records how many times Pause and Resume are called.
type mockListenerPauser struct {
	pauseCount  int
	resumeCount int
}

func (m *mockListenerPauser) PauseListener()  { m.pauseCount++ }
func (m *mockListenerPauser) ResumeListener() { m.resumeCount++ }

func TestEventSync_listenerPauser(t *testing.T) {
	cfg := createCentralCfg("https://abc.com", "mockenv")
	err := Initialize(cfg)
	assert.Nil(t, err)

	scopeName := cfg.Environment

	apiSvcFilter := management.WatchTopicSpecFilters{
		Group: management.APIServiceGVK().Group,
		Kind:  management.APIServiceGVK().Kind,
		Name:  "*",
		Scope: &management.WatchTopicSpecScope{Kind: "Environment", Name: scopeName},
	}
	wt := &management.WatchTopic{
		Spec: management.WatchTopicSpec{
			Filters: []management.WatchTopicSpecFilters{apiSvcFilter},
		},
	}

	tests := map[string]struct {
		run             func(t *testing.T, es *EventSync, pauser *mockListenerPauser)
		expectedPauses  int
		expectedResumes int
	}{
		"rebuildFilters pauses and resumes the listener": {
			run: func(t *testing.T, es *EventSync, _ *mockListenerPauser) {
				err := es.rebuildFilters([]management.WatchTopicSpecFilters{apiSvcFilter})
				assert.Nil(t, err)
			},
			expectedPauses:  1,
			expectedResumes: 1,
		},
		"pausedInitCache pauses and resumes the listener": {
			run: func(t *testing.T, es *EventSync, _ *mockListenerPauser) {
				// Use a harvester that returns an error so initCache exits early,
				// but the pause/resume wrapping in pausedInitCache still executes.
				es.harvester = &mockHarvester{err: fmt.Errorf("harvester error")}
				_ = es.pausedInitCache()
			},
			expectedPauses:  1,
			expectedResumes: 1,
		},
		"rebuildFilters resumes even when discoveryCache fails": {
			run: func(t *testing.T, es *EventSync, _ *mockListenerPauser) {
				// Replace client with one that always fails so execute() returns an error.
				dc := newDiscoveryCache(cfg, &mockRIClient{err: fmt.Errorf("simulated error")}, nil, wt)
				es.discoveryCache = dc
				_ = es.rebuildFilters([]management.WatchTopicSpecFilters{apiSvcFilter})
			},
			expectedPauses:  1,
			expectedResumes: 1,
		},
		"no listenerPauser set - rebuildFilters succeeds without panic": {
			run: func(t *testing.T, es *EventSync, _ *mockListenerPauser) {
				es.listenerPauser = nil
				assert.NotPanics(t, func() {
					_ = es.rebuildFilters([]management.WatchTopicSpecFilters{apiSvcFilter})
				})
			},
			expectedPauses:  0,
			expectedResumes: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			agent.cacheManager = agentcache.NewAgentCacheManager(cfg, false)
			pauser := &mockListenerPauser{}
			mockClient := &mockRIClient{svcs: newAPIServices(scopeName)}
			dc := newDiscoveryCache(cfg, mockClient, nil, wt)
			es := &EventSync{
				watchTopic:     wt,
				harvester:      &mockHarvester{},
				discoveryCache: dc,
				sequence:       &mockSequence{},
				listenerPauser: pauser,
			}

			tc.run(t, es, pauser)

			assert.Equal(t, tc.expectedPauses, pauser.pauseCount, "pause count mismatch")
			assert.Equal(t, tc.expectedResumes, pauser.resumeCount, "resume count mismatch")
		})
	}
}
