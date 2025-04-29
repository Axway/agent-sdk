package events

import (
	"fmt"
	"testing"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
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

			wt := management.NewWatchTopic("")
			err := wt.FromInstance(tc.ri)
			assert.Nil(t, err)

			wt, err = createOrUpdateWatchTopic(wt, rc)
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.ri.Name, wt.Name)
			}
		})
	}

}

type mockWatchTopicFeatures struct {
	agentType   config.AgentType
	managedEnvs []string
	filterList  []config.ResourceFilter
}

func (m *mockWatchTopicFeatures) GetAgentType() config.AgentType {
	return m.agentType
}

func (m *mockWatchTopicFeatures) GetManagedEnvironments() []string {
	return m.managedEnvs
}

func (m *mockWatchTopicFeatures) GetWatchResourceFilters() []config.ResourceFilter {
	return m.filterList
}

func Test_parseWatchTopic(t *testing.T) {
	tests := []struct {
		name         string
		isMPSEnabled bool
	}{
		{
			name: "Should create a watch topic without marketplace subs enabled",
		},
		{
			name:         "Should create a watch topic with marketplace subs enabled",
			isMPSEnabled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			features := &mockWatchTopicFeatures{}

			wt, err := parseWatchTopicTemplate(NewDiscoveryWatchTopic("name", "scope", management.DiscoveryAgentGVK().GroupKind, features))
			assert.Nil(t, err)
			assert.NotNil(t, wt)

			wt, err = parseWatchTopicTemplate(NewTraceWatchTopic("name", "scope", management.TraceabilityAgentGVK().GroupKind, features))
			assert.Nil(t, err)
			assert.NotNil(t, wt)
		})
	}
}

func TestGetOrCreateWatchTopic(t *testing.T) {
	tests := []struct {
		name        string
		client      *mockAPIClient
		hasErr      bool
		agentType   config.AgentType
		filterList  []config.ResourceFilter
		managedEnvs []string
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
			filterList: []config.ResourceFilter{},
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
			filterList: []config.ResourceFilter{},
		},
		{
			name:      "should create a watch topic for a compliance agent if it does not exist",
			agentType: config.ComplianceAgent,
			hasErr:    false,
			client: &mockAPIClient{
				getErr: fmt.Errorf("not found"),
				ri: &apiv1.ResourceInstance{
					ResourceMeta: apiv1.ResourceMeta{
						Name: "wt-name",
					},
				},
			},
			filterList:  []config.ResourceFilter{},
			managedEnvs: []string{"managedEnv1", "managedEnv2"},
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
			filterList: []config.ResourceFilter{},
		},
		{
			name:      "should create a watch topic for a trace agent with custom filter if it does not exist",
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
			filterList: []config.ResourceFilter{
				{
					Group:      management.CredentialGVK().Group,
					Kind:       management.CredentialGVK().Kind,
					Name:       "*",
					EventTypes: []config.ResourceEventType{"created"},
					Scope: &config.ResourceScope{
						Kind: management.EnvironmentGVK().Kind,
						Name: "test-env",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := "agent-name"
			features := &mockWatchTopicFeatures{agentType: tc.agentType, filterList: tc.filterList, managedEnvs: tc.managedEnvs}

			wt, err := getOrCreateWatchTopic(name, "scope", tc.client, features)
			if tc.hasErr == true {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.client.ri.Name, wt.Name)
			}
			// validate watch topic with custom filter
			for _, filter := range tc.filterList {
				found := false
				for _, wtFilter := range wt.Spec.Filters {
					if wtFilter.Group == filter.Group && wtFilter.Kind == filter.Kind && wtFilter.Name == filter.Name {
						fs := filter.Scope
						wts := wtFilter.Scope
						if fs != nil && wts != nil && wts.Kind == fs.Kind && wts.Name == fs.Name {
							found = true
							break
						}
					}
				}
				assert.True(t, found)
			}
			// validate watch topic filter for compliance agent managed env filters
			if len(tc.managedEnvs) > 0 && tc.agentType == config.ComplianceAgent {
				for _, env := range tc.managedEnvs {
					foundEnv := false
					foundAPIS := false
					for _, wtFilter := range wt.Spec.Filters {
						if wtFilter.Group == management.EnvironmentGVK().Group && wtFilter.Kind == management.EnvironmentGVK().Kind && wtFilter.Name == env {
							foundEnv = true
						}
						if wtFilter.Group == management.APIServiceInstanceGVK().Group && wtFilter.Kind == management.APIServiceInstanceGVK().Kind && wtFilter.Scope.Name == env {
							foundAPIS = true
						}
					}
					assert.Truef(t, foundEnv, "managed env, %s, environment filter not found", env)
					assert.Truef(t, foundAPIS, "managed env, %s, api service instance filter not found", env)
				}
			}
		})
	}
}

func Test_shouldPushUpdate(t *testing.T) {
	type args struct {
		cur []management.WatchTopicSpecFilters
		new []management.WatchTopicSpecFilters
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should not push update",
			args: args{
				cur: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
				new: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
			},
			want: false,
		},
		{
			name: "should push update, second more",
			args: args{
				cur: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
				new: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind1",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
			},
			want: true,
		},
		{
			name: "should push update, first more",
			args: args{
				cur: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind1",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
				new: []management.WatchTopicSpecFilters{
					{
						Group: "group",
						Scope: nil,
						Kind:  "kind",
						Name:  "name",
						Type:  []string{"type1", "type2", "type3"},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createWatchTopic := func(filters []management.WatchTopicSpecFilters) *management.WatchTopic {
				wt := management.NewWatchTopic("")
				wt.Spec.Filters = filters
				return wt
			}

			if got := shouldPushUpdate(createWatchTopic(tt.args.cur), createWatchTopic(tt.args.new)); got != tt.want {
				t.Errorf("shouldPushUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filtersEqual(t *testing.T) {
	type args struct {
		a management.WatchTopicSpecFilters
		b management.WatchTopicSpecFilters
	}
	tests := []struct {
		name      string
		args      args
		wantEqual bool
	}{
		{
			name: "group diff",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group1",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "kind diff",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind1",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "name diff",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name1",
					Type:  []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff 1",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff 2",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: nil,
					Kind:  "kind",
					Name:  "name",
					Type:  []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff name",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name1",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff name",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind1",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff types 1",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2"},
				},
			},
			wantEqual: false,
		},
		{
			name: "scope diff types 2",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: false,
		},
		{
			name: "equal",
			args: args{
				a: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
				b: management.WatchTopicSpecFilters{
					Group: "group",
					Scope: &management.WatchTopicSpecScope{
						Kind: "kind",
						Name: "name",
					},
					Kind: "kind",
					Name: "name",
					Type: []string{"type1", "type2", "type3"},
				},
			},
			wantEqual: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotEqual := filtersEqual(tt.args.a, tt.args.b); gotEqual != tt.wantEqual {
				t.Errorf("filtersEqual() = %v, want %v", gotEqual, tt.wantEqual)
			}
		})
	}
}

func Test_getWatchTopic(t *testing.T) {
	wt := &management.WatchTopic{}
	ri, _ := wt.AsInstance()
	httpClient := &mockAPIClient{
		ri: ri,
	}
	cfg := &config.CentralConfiguration{
		AgentType:     1,
		TenantID:      "12345",
		Environment:   "stream-test",
		EnvironmentID: "123",
		AgentName:     "discoveryagents",
		URL:           "http://abc.com",
		TLS:           &config.TLSConfiguration{},
	}

	wt, err := GetWatchTopic(cfg, httpClient)
	assert.NotNil(t, wt)
	assert.Nil(t, err)

	wt, err = GetWatchTopic(cfg, httpClient)
	assert.NotNil(t, wt)
	assert.Nil(t, err)
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

func (m *mockAPIClient) GetAPIV1ResourceInstances(_ map[string]string, _ string) ([]*apiv1.ResourceInstance, error) {
	return nil, nil
}
