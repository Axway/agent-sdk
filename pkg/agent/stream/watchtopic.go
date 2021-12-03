package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/cache"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

// WatchTopicName naming pattern for creating watch topics
func WatchTopicName(env, agent string) string {
	return fmt.Sprintf("%s-%s", env, agent)
}

// GetWatchTopic checks the cache for a saved WatchTopic ResourceClient
func GetWatchTopic(c cache.Cache, key string) (*mv1.WatchTopic, error) {
	item, err := c.Get(key)
	if err != nil {
		return nil, err
	}

	v, ok := item.(*mv1.WatchTopic)
	if !ok {
		return nil, fmt.Errorf("found item for %s, but it is not a *WatchTopic", key)
	}

	return v, nil
}

// NewWatchTopic creates a WatchTopic ResourceClient
func NewWatchTopic(name, scope string) *mv1.WatchTopic {
	return &mv1.WatchTopic{
		ResourceMeta: apiv1.ResourceMeta{
			Name: name,
		},
		Spec: mv1.WatchTopicSpec{
			Description: fmt.Sprintf("Watch Topic for resources in the %s environment.", scope),
			Filters: []mv1.WatchTopicSpecFilters{
				{
					Group: "management",
					Name:  "*",
					Scope: mv1.WatchTopicSpecScope{
						Kind: "APIService",
						Name: scope,
					},
				},
				{
					Group: "management",
					Name:  "*",
					Scope: mv1.WatchTopicSpecScope{
						Kind: "APIServiceInstance",
						Name: scope,
					},
				},
				{
					Group: "catalog",
					Name:  "*",
				},
			},
		},
	}
}
