package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var errCacheOutOfSync = fmt.Errorf("persisted cache is out of sync with API server")

type cacheGetter interface {
	GetCachedResourcesByKind(group, kind, scopeName string) map[string]time.Time
}

// cacheValidator validates the persisted cache against the API server
type cacheValidator struct {
	logger     log.FieldLogger
	client     resourceClient
	watchTopic *management.WatchTopic
	cacheMan   cacheGetter
	harvester  harvester.Harvest
	sequence   events.SequenceProvider
}

func newCacheValidator(
	client resourceClient,
	watchTopic *management.WatchTopic,
	cacheMan cacheGetter,
	hClient harvester.Harvest,
	seq events.SequenceProvider,
) *cacheValidator {
	logger := log.NewFieldLogger().
		WithPackage("sdk.agent").
		WithComponent("cacheValidator")

	return &cacheValidator{
		logger:     logger,
		client:     client,
		watchTopic: watchTopic,
		cacheMan:   cacheMan,
		harvester:  hClient,
		sequence:   seq,
	}
}

// Execute validates each filter in the watch topic against the API server.
// It returns the slice of filters whose cache is out of sync, plus errCacheOutOfSync
// if any failed, or a nil slice and nil error if all filters are in sync.
func (cv *cacheValidator) Execute() ([]management.WatchTopicSpecFilters, error) {
	cv.logger.Debug("executing cache validation")

	seqInSync := cv.sequenceInSync()
	if !seqInSync {
		return nil, errCacheOutOfSync
	}

	filters := cv.watchTopic.Spec.Filters
	ch := make(chan management.WatchTopicSpecFilters, len(filters))

	cachedFilters := make([]management.WatchTopicSpecFilters, 0, len(filters))
	var wg sync.WaitGroup
	apiSvcFilter := management.WatchTopicSpecFilters{}
	for _, filter := range filters {
		if agentcache.IsCachedKind(filter.Kind) {
			wg.Add(1)
			cachedFilters = append(cachedFilters, filter)
			if filter.Kind == management.APIServiceGVK().Kind {
				apiSvcFilter = filter
			}
			go func(f management.WatchTopicSpecFilters) {
				defer wg.Done()
				if !cv.validateKind(f) {
					ch <- f
				}
			}(filter)
		}
	}
	wg.Wait()
	close(ch)

	failed := make(map[string]management.WatchTopicSpecFilters)
	for f := range ch {
		failed[f.Kind] = f
	}

	// APIService cache is keyed by external API ID instead of resource ID.
	// Write APIService filter along with APIServiceInstance filter when APIServiceInstance counts are off.
	_, ok := failed[management.APIServiceInstanceGVK().Kind]
	if ok && apiSvcFilter.Kind != "" {
		failed[apiSvcFilter.Kind] = apiSvcFilter
	}

	if len(failed) > 0 {
		filtersToProcess := make([]management.WatchTopicSpecFilters, 0, len(failed))
		for _, f := range failed {
			filtersToProcess = append(filtersToProcess, f)
		}
		return filtersToProcess, errCacheOutOfSync
	}

	cv.logger.Debug("cache validation passed")
	return cachedFilters, nil
}

// sequenceInSync fetches the latest sequence ID from the harvester and compares it
// with the locally stored sequence. Returns true when they match, allowing Execute
// to skip per-kind validation entirely. Returns false on any error or mismatch,
// logging the discrepancy so callers have context for the validation that follows.
func (cv *cacheValidator) sequenceInSync() bool {
	if cv.harvester == nil || cv.sequence == nil {
		return false
	}

	serverSeq, err := cv.harvester.ReceiveSyncEvents(
		context.Background(), cv.watchTopic.GetSelfLink(), 0, nil,
	)
	if err != nil {
		cv.logger.WithError(err).Debug("could not fetch latest harvester sequence, proceeding with per-kind validation")
		return true
	}

	cachedSeq := cv.sequence.GetSequence()
	// At start-up or agent restart, because of the status change event, the serverSeq will always be higher with 1 than the cachedSeq.
	// This avoids unnecessary rebuilding at start-up.
	if serverSeq != cachedSeq+1 && serverSeq != cachedSeq {
		cv.logger.
			WithField("cachedSeq", cachedSeq).
			WithField("serverSeq", serverSeq).
			Info("sequence mismatch detected")
		return false
	}

	return true
}

func (cv *cacheValidator) validateKind(filter management.WatchTopicSpecFilters) bool {
	logger := cv.logger.WithField("kind", filter.Kind).WithField("group", filter.Group)

	ri := apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: filter.Group,
					Kind:  filter.Kind,
				},
				APIVersion: "v1",
			},
		},
	}
	if filter.Scope != nil {
		ri.Metadata.Scope.Kind = filter.Scope.Kind
		ri.Metadata.Scope.Name = filter.Scope.Name
	}

	url := ri.GetKindLink()
	if url == "" {
		logger.Trace("skipping validation, could not build resource URL")
		return true
	}

	scopeName := ""
	if filter.Scope != nil {
		scopeName = filter.Scope.Name
	}
	cachedResources := cv.cacheMan.GetCachedResourcesByKind(filter.Group, filter.Kind, scopeName)

	// Count check: if the number of resources in the cache does not match the number on the server, we know the cache is out of sync.
	// Note: APIService cache with existing duplicates on server may always fail this check, as it gets cached based on externalAPIID and not the resource id.
	serverCount, err := cv.client.GetAPIV1ResourceCount(url)
	if err != nil {
		logger.WithError(err).Error("HEAD request failed, falling back to full metadata fetch")
		return false
	}

	if filter.Kind == management.APIServiceGVK().Kind {
		if serverCount < len(cachedResources) {
			logger.
				WithField("serverCount", serverCount).
				WithField("cacheCount", len(cachedResources)).
				Info("cache validation: APIService count mismatch, cache is out of sync")
			return false
		}
		return true
	}

	if serverCount != len(cachedResources) {
		logger.
			WithField("serverCount", serverCount).
			WithField("cacheCount", len(cachedResources)).
			Info("cache validation: count mismatch, cache is out of sync")
		return false
	}
	return true
}
