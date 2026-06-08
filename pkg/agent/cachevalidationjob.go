package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/agent/events"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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

	filters := cv.watchTopic.Spec.Filters
	ch := make(chan management.WatchTopicSpecFilters, len(filters))

	var wg sync.WaitGroup
	for _, filter := range filters {
		wg.Add(1)
		go func(f management.WatchTopicSpecFilters) {
			defer wg.Done()
			if !cv.validateKind(f, seqInSync) {
				ch <- f
			}
		}(filter)
	}
	wg.Wait()
	close(ch)

	var failed []management.WatchTopicSpecFilters
	for f := range ch {
		failed = append(failed, f)
	}

	if len(failed) > 0 {
		return failed, errCacheOutOfSync
	}

	cv.logger.Debug("cache validation passed")
	return nil, nil
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
		return false
	}

	cachedSeq := cv.sequence.GetSequence()
	if serverSeq != cachedSeq {
		cv.logger.
			WithField("cachedSeq", cachedSeq).
			WithField("serverSeq", serverSeq).
			Info("sequence mismatch detected, proceeding with per-kind cache validation")
		return false
	}

	return true
}

func (cv *cacheValidator) validateKind(filter management.WatchTopicSpecFilters, seqInSync bool) bool {
	logger := cv.logger.WithField("kind", filter.Kind).WithField("group", filter.Group)

	if !agentcache.IsCachedKind(filter.Kind) {
		logger.Trace("skipping validation for kind")
		return true
	}

	ri := apiv1.ResourceInstance{
		ResourceMeta: apiv1.ResourceMeta{
			GroupVersionKind: apiv1.GroupVersionKind{
				GroupKind: apiv1.GroupKind{
					Group: filter.Group,
					Kind:  filter.Kind,
				},
				APIVersion: "v1alpha1",
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

	// HEAD pre-check: compare resource count before fetching full metadata.
	// Returns early only when the count matches; otherwise logs and falls through
	// to the full metadata fetch.
	serverCount, err := cv.client.GetAPIV1ResourceCount(url)
	if err != nil {
		logger.WithError(err).Error("HEAD request failed, falling back to full metadata fetch")
	} else if serverCount == len(cachedResources) && seqInSync {
		logger.WithField("count", serverCount).Trace("cache validation passed for kind (count and sequence match)")
		return true
	} else if serverCount != len(cachedResources) {
		logger.
			WithField("serverCount", serverCount).
			WithField("cacheCount", len(cachedResources)).
			Info("cache validation: count mismatch, fetching metadata for timestamp check")
	}

	// Full metadata fetch for timestamp-level confirmation.
	query := map[string]string{
		"fields": "name,kind,metadata.audit,metadata.scope",
	}

	logger.Trace("fetching resource metadata from API server for validation")
	serverResources, err := cv.client.GetAPIV1ResourceInstances(query, url)
	if err != nil {
		logger.WithError(err).Error("failed to fetch resource metadata for cache validation")
		return false
	}

	// Build a map from server resources: composite key -> modifyTimestamp
	serverMap := make(map[string]time.Time, len(serverResources))
	for _, res := range serverResources {
		modTime := time.Time(res.Metadata.Audit.ModifyTimestamp)
		serverMap[agentcache.ResourceCacheKey(res.Kind, res.Metadata.Scope.Name, res.Name)] = modTime
	}

	// Each server resource must exist in cache with a matching mod timestamp.
	for name, serverModTime := range serverMap {
		cacheModTime, exists := cachedResources[name]
		if !exists {
			logger.
				WithField("resource", name).
				Info("cache validation failed: resource not found in cache")
			return false
		}

		if !serverModTime.IsZero() && !cacheModTime.IsZero() && !serverModTime.Equal(cacheModTime) {
			logger.
				WithField("resource", name).
				WithField("serverModTime", serverModTime).
				WithField("cacheModTime", cacheModTime).
				Info("cache validation failed: modification time mismatch")
			return false
		}
	}

	logger.WithField("count", len(serverMap)).Trace("cache validation passed for kind")
	return true
}
