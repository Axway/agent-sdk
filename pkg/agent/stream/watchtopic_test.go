package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/config"

	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"

	"github.com/stretchr/testify/assert"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/cache"
)

func TestCreateWatchTopic(t *testing.T) {
	tests := []struct {
		name   string
		ri     *apiv1.ResourceInstance
		hasErr bool
		err    error
	}{
		{
			name:   "Should call create and return a WatchTopic",
			hasErr: false,
			err:    nil,
			ri: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{
					Name: "wt-name",
				},
			},
		},
		{
			name:   "Should return an error when calling create",
			hasErr: true,
			err:    fmt.Errorf("error"),
			ri: &apiv1.ResourceInstance{
				ResourceMeta: apiv1.ResourceMeta{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rc := &mockAPIClient{
				ri:        tc.ri,
				createErr: tc.err,
			}

			bts, err := tc.ri.MarshalJSON()
			assert.Nil(t, err)

			wt, err := createWatchTopic(bts, rc)
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.ri.Name, wt.Name)
			}
		})
	}

}

func TestGetCachedWatchTopic(t *testing.T) {
	c1 := mockCacheGet{
		item: &mv1.WatchTopic{
			ResourceMeta: apiv1.ResourceMeta{
				Name: "wt-name",
			},
		},
		err: nil,
	}

	c2 := mockCacheGet{
		item: nil,
		err:  fmt.Errorf("err"),
	}

	tests := []struct {
		name   string
		key    string
		hasErr bool
		err    error
		cache  cache.GetItem
	}{
		{
			name:   "should get a watch topic from the cache",
			hasErr: false,
			err:    nil,
			cache:  c1,
			key:    "wt-name",
		},
		{
			name:   "should get a watch topic from the cache",
			hasErr: true,
			err:    fmt.Errorf("error"),
			cache:  c2,
			key:    "asdsf",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wt, err := getCachedWatchTopic(tc.cache, tc.key)

			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.key, wt.Name)
			}
		})
	}
}

func Test_parseWatchTopic(t *testing.T) {
	bts, err := parseWatchTopicTemplate(NewDiscoveryWatchTopic("name", "scope", mv1.DiscoveryAgentGVK().GroupKind))
	assert.Nil(t, err)

	assert.True(t, len(bts) > 0)

	bts, err = parseWatchTopicTemplate(NewTraceWatchTopic("name", "scope", mv1.TraceabilityAgentGVK().GroupKind))
	assert.Nil(t, err)

	assert.True(t, len(bts) > 0)

	bts, err = parseWatchTopicTemplate(NewGovernanceAgentWatchTopic("name", "scope", mv1.GovernanceAgentGVK().GroupKind))
	assert.Nil(t, err)

	assert.True(t, len(bts) > 0)
}

func TestGetOrCreateWatchTopic(t *testing.T) {
	tests := []struct {
		name      string
		client    *mockAPIClient
		hasErr    bool
		agentType config.AgentType
	}{
		{
			name:      "should retrieve a watch topic if it exists",
			hasErr:    false,
			agentType: config.DiscoveryAgent,
			client: &mockAPIClient{
				ri: &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Name: "wt-name",
					},
				},
			},
		},
		{
			name:      "should create a watch topic for a trace agent if it does not exist",
			agentType: config.TraceabilityAgent,
			hasErr:    false,
			client: &mockAPIClient{
				getErr: fmt.Errorf("not found"),
				ri: &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Name: "wt-name",
					},
				},
			},
		},
		{
			name:      "should create a watch topic for a discovery agent if it does not exist",
			agentType: config.DiscoveryAgent,
			hasErr:    false,
			client: &mockAPIClient{
				getErr: fmt.Errorf("not found"),
				ri: &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Name: "wt-name",
					},
				},
			},
		},
		{
			name:      "should create a watch topic for a governance agent if it does not exist",
			agentType: config.GovernanceAgent,
			hasErr:    false,
			client: &mockAPIClient{
				getErr: fmt.Errorf("not found"),
				ri: &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Name: "wt-name",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := "agent-name"

			wt, err := getOrCreateWatchTopic(name, "scope", tc.client, tc.agentType)
			if tc.hasErr == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.client.ri.Name, wt.Name)
			}
		})
	}
}

type mockCacheGet struct {
	item interface{}
	err  error
}

func (m mockCacheGet) Get(_ string) (interface{}, error) {
	return m.item, m.err
}
