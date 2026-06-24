package poller

import (
	"context"
	"fmt"
	"testing"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

var cfg = &config.CentralConfiguration{
	AgentType:     1,
	TenantID:      "12345",
	Environment:   "stream-test",
	EnvironmentID: "123",
	AgentName:     "discoveryagents",
	URL:           "http://abc.com",
	TLS:           config.NewTLSConfig(),
	SingleURL:     "https://abc.com",
	PollInterval:  1 * time.Second,
}

func TestPollClientStart(t *testing.T) {
	wt := management.NewWatchTopic("mocktopic")
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}

	mockH := &mockHarvester{
		readyCh: make(chan struct{}),
	}

	cacheManager := agentcache.NewAgentCacheManager(cfg, false)
	seq := events.NewSequenceProvider(cacheManager, wt.Name)
	seq.SetSequence(1)

	pollClient, err := NewPollClient(httpClient, cfg, nil, WithHarvester(mockH, seq, wt.GetSelfLink()))
	assert.NotNil(t, pollClient)
	assert.Nil(t, err)

	pollClient.newPollManager = func(interval time.Duration, options ...executorOpt) *pollExecutor {
		p := newPollExecutor(cfg.PollInterval, options...)
		p.harvester = mockH
		return p
	}

	errCh := make(chan error)
	go func() {
		err := pollClient.Start()
		errCh <- err
	}()

	<-mockH.readyCh

	// assert the poller is healthy
	assert.Nil(t, pollClient.Status())
	assert.Equal(t, hc.OK, pollClient.Healthcheck("").Result)

	// should stop the poller and receive an error that it was closed
	pollClient.Stop()

	err = <-errCh
	assert.NotNil(t, err)

	assert.Equal(t, hc.FAIL, pollClient.Healthcheck("").Result)
	assert.NotNil(t, pollClient.Status())
	pollClient.poller = nil
	pollClient.listener = nil
}

type mockAPIClient struct {
	ri        *apiv1.ResourceInstance
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m mockAPIClient) GetResource(url string) (*apiv1.ResourceInstance, error) {
	return m.ri, m.getErr
}

func (m mockAPIClient) CreateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.ri, m.createErr
}

func (m mockAPIClient) UpdateResourceInstance(_ apiv1.Interface) (*apiv1.ResourceInstance, error) {
	return m.ri, m.updateErr
}

func (m mockAPIClient) DeleteResourceInstance(_ apiv1.Interface) error {
	return m.deleteErr
}

func (m mockAPIClient) GetAPIV1ResourceInstances(map[string]string, string) ([]*apiv1.ResourceInstance, error) {
	return nil, nil
}

type mockTokenGetter struct {
	token string
	err   error
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return m.token, m.err
}

type mockHarvester struct {
	eventCh chan *proto.Event
	err     error
	readyCh chan struct{}
}

func (m mockHarvester) EventCatchUp(_ context.Context, _ string, _ chan *proto.Event) error {
	return nil
}

func (m mockHarvester) ReceiveSyncEvents(_ context.Context, _ string, _ int64, _ chan *proto.Event) (int64, error) {
	if m.readyCh != nil {
		m.readyCh <- struct{}{}
	}

	if m.eventCh != nil {
		m.eventCh <- &proto.Event{
			Id: "1",
		}
	}
	return 0, m.err
}

func TestPollClientWaitForReady(t *testing.T) {
	catchUpErr := fmt.Errorf("harvester catch-up failed")
	blockCh := make(chan struct{})

	cases := map[string]struct {
		harvester   harvester.Harvest
		ctxTimeout  time.Duration
		expectedErr error
		cleanup     func()
	}{
		"connects successfully": {
			harvester:   &mockHarvester{},
			ctxTimeout:  2 * time.Second,
			expectedErr: nil,
		},
		"context deadline exceeded before connection": {
			harvester:   &blockingHarvester{block: blockCh},
			ctxTimeout:  50 * time.Millisecond,
			expectedErr: context.DeadlineExceeded,
			cleanup:     func() { close(blockCh) },
		},
		"start fails before connecting": {
			harvester:   &blockingHarvester{err: catchUpErr}, // nil block: EventCatchUp returns err immediately
			ctxTimeout:  2 * time.Second,
			expectedErr: catchUpErr,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			wt := management.NewWatchTopic("mocktopic")
			ri, _ := wt.AsInstance()
			httpClient := &mockAPIClient{ri: ri}

			cacheManager := agentcache.NewAgentCacheManager(cfg, false)
			seq := events.NewSequenceProvider(cacheManager, wt.Name)
			seq.SetSequence(1)

			pc, err := NewPollClient(httpClient, cfg, nil, WithHarvester(tc.harvester, seq, wt.GetSelfLink()))
			assert.Nil(t, err)

			pc.newPollManager = func(interval time.Duration, options ...executorOpt) *pollExecutor {
				p := newPollExecutor(cfg.PollInterval, options...)
				p.harvester = tc.harvester
				return p
			}

			go func() { _ = pc.Start() }()
			defer pc.Stop()
			if tc.cleanup != nil {
				defer tc.cleanup()
			}

			ctx, cancel := context.WithTimeout(context.Background(), tc.ctxTimeout)
			defer cancel()

			err = pc.WaitForReady(ctx)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

// blockingHarvester blocks EventCatchUp until block is closed (or returns err if block is nil).
type blockingHarvester struct {
	block chan struct{}
	err   error
}

func (m *blockingHarvester) EventCatchUp(_ context.Context, _ string, _ chan *proto.Event) error {
	if m.block == nil {
		return m.err
	}
	<-m.block
	return m.err
}

func (m *blockingHarvester) ReceiveSyncEvents(_ context.Context, _ string, _ int64, _ chan *proto.Event) (int64, error) {
	return 0, nil
}

var watchTopic = &management.WatchTopic{
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
					Name: "mockEnvName",
				},
			},
		},
	},
}
