package stream

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

// StreamerOpt func for setting fields on the StreamerClient
type StreamerOpt func(client *StreamerClient)

// WithWatchTopic sets the watch topic
func WithWatchTopic(wt *management.WatchTopic) StreamerOpt {
	return func(client *StreamerClient) {
		client.wt = wt
		client.topicSelfLink = wt.GetSelfLink()
	}
}

// WithCacheManager sets a cache manager
func WithCacheManager(cache agentcache.Manager) StreamerOpt {
	return func(client *StreamerClient) {
		client.cacheManager = cache
	}
}

// WithUserAgent sets the userAgent for gRPC stream
func WithUserAgent(userAgent string) StreamerOpt {
	return func(client *StreamerClient) {
		client.watchCfg.UserAgent = userAgent
	}
}

// WithHarvester configures the streaming client to use harvester for syncing initial events
func WithHarvester(hClient harvester.Harvest, sequence events.SequenceProvider) StreamerOpt {
	return func(client *StreamerClient) {
		client.sequence = sequence
		client.harvester = hClient
	}
}

// WithEventSyncError sets the callback func to run when there is an error syncing events
func WithEventSyncError(cb func()) StreamerOpt {
	return func(client *StreamerClient) {
		client.onEventSyncError = cb
	}
}

// WithOnStreamConnection func to execute when a connection to central is made
func WithOnStreamConnection() StreamerOpt {
	return func(pc *StreamerClient) {
		pc.onStreamConnection = func() {
			hc.RegisterHealthcheck(util.AmplifyCentral, util.CentralHealthCheckEndpoint, pc.Healthcheck)
		}
	}
}
