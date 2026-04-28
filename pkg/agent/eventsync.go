package agent

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/poller"
	"github.com/Axway/agent-sdk/pkg/agent/stream"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/jobs"
	"github.com/Axway/agent-sdk/pkg/migrate"
	"github.com/Axway/agent-sdk/pkg/util"
)

// EventSync struct for syncing events from central
type EventSync struct {
	watchTopic     *management.WatchTopic
	sequence       events.SequenceProvider
	harvester      harvester.Harvest
	discoveryCache *discoveryCache
	cacheValidator *cacheValidator
	listenerPauser listenerPauser
}

// listenerPauser is satisfied by StreamerClient; allows EventSync to pause the
// live event listener while mutating the cache.
type listenerPauser interface {
	PauseListener()
	ResumeListener()
}

// newEventSync creates an EventSync
func newEventSync() (*EventSync, error) {
	migrations := []migrate.Migrator{}

	// Make sure only DA agents run migration processes
	runMigrations := agent.cfg.GetAgentType() == config.DiscoveryAgent

	if runMigrations {
		// add attribute migration to migrations
		attributeMigration := migrate.NewAttributeMigration(agent.apicClient, agent.cfg)
		ardMigration := migrate.NewArdMigration(agent.apicClient, agent.cfg)
		apisiMigration := migrate.NewAPISIMigration(agent.apicClient, agent.cfg)
		instanceMigration := migrate.NewInstanceMigration(agent.apicClient, agent.cfg)
		migrations = append(migrations, attributeMigration, ardMigration, apisiMigration, instanceMigration)
	}

	mig := migrate.NewMigrateAll(migrations...)

	opts := []discoveryOpt{
		withMigration(mig),
		preMarketplaceSetup(finalizeInitialization),
	}

	if agent.agentResourceManager != nil {
		opts = append(opts, withAdditionalDiscoverFuncs(agent.agentResourceManager.FetchAgentResource))
	}

	wt, err := events.GetWatchTopic(agent.cfg, GetCentralClient())
	if err != nil {
		return nil, err
	}

	sequence := events.NewSequenceProvider(agent.cacheManager, wt.Name)
	hCfg := harvester.NewConfig(agent.cfg, agent.tokenRequester, sequence)
	hClient := harvester.NewClient(hCfg)

	discoveryCache := newDiscoveryCache(
		agent.cfg,
		GetCentralClient(),
		newHandlers(),
		wt,
		opts...,
	)

	cacheValidator := newCacheValidator(
		GetCentralClient(),
		wt,
		agent.cacheManager,
	)

	return &EventSync{
		watchTopic:     wt,
		sequence:       sequence,
		harvester:      hClient,
		discoveryCache: discoveryCache,
		cacheValidator: cacheValidator,
	}, nil
}

// SyncCache initializes agent cache and starts the agent in stream or poll mode
func (es *EventSync) SyncCache() error {
	if !agent.cacheManager.HasLoadedPersistedCache() {
		if err := es.initCache(); err != nil {
			return err
		}
	} else {
		// Validate the persisted cache against the API server
		failedFilters, err := es.validateCache()
		if err != nil {
			logger.WithError(err).Info("persisted cache validation failed, rebuilding out-of-sync kinds")
			if err := es.initCache(failedFilters...); err != nil {
				return err
			}
		} else {
			err := finalizeInitialization()
			if err != nil {
				logger.WithError(err).Error("error finalizing setup prior to marketplace resource syncing")
				return err
			}
		}
	}

	err := es.startCentralEventProcessor()
	if err != nil {
		return err
	}

	isEnabled := agent.cfg.IsInstanceValidationEnabled()
	logger.WithField("instanceValidatorStatus", StatusString(isEnabled, "Enabled", "Disabled")).Trace("Checking instance validator status")
	if isEnabled {
		return es.registerInstanceValidator()
	}
	return nil
}

func (es *EventSync) registerInstanceValidator() error {
	if agent.apiValidatorJobID == "" && agent.cfg.GetAgentType() == config.DiscoveryAgent {
		jobID, err := jobs.RegisterScheduledJobWithName(newInstanceValidator(), agent.cfg.GetAPIValidationCronSchedule(), "API service instance validator")
		agent.apiValidatorJobID = jobID
		return err
	}
	return nil
}

func (es *EventSync) initCache(failedFilters ...management.WatchTopicSpecFilters) error {
	seqID, err := es.harvester.ReceiveSyncEvents(context.Background(), es.watchTopic.GetSelfLink(), 0, nil)
	if err != nil {
		return err
	}

	// Attempt a targeted rebuild when only specific kinds are out of sync
	if len(failedFilters) > 0 {
		for _, f := range failedFilters {
			agent.cacheManager.FlushKind(f.Kind)
		}
		if rebuildErr := es.discoveryCache.execute(failedFilters...); rebuildErr == nil {
			if seqID > 0 {
				es.sequence.SetSequence(seqID - 1)
			}
			agent.cacheManager.SaveCache()
			es.resetCacheTimer()
			return nil
		} else {
			logger.WithError(rebuildErr).Info("targeted cache rebuild failed, falling back to full rebuild")
		}
	}

	// Full rebuild: flush everything and re-populate
	// event channel is not ready yet, so subtract one from the latest sequence id to process the event
	// when the poll/stream client is ready
	// when no events returned by harvester the seqID will be 0, so not updated in sequence manager
	agent.cacheManager.Flush()
	if seqID > 0 {
		es.sequence.SetSequence(seqID - 1)
	}
	err = es.discoveryCache.execute()
	if err != nil {
		// flush cache again to clear out anything that may have been saved before the error to ensure a clean state for the next time through
		agent.cacheManager.Flush()
		return err
	}
	agent.cacheManager.SaveCache()

	es.resetCacheTimer()
	return nil
}

// resetCacheTimer persists a new cacheUpdateTime 7 days from now in the agent's x-agent-details.
func (es *EventSync) resetCacheTimer() {
	agentInstance := agent.agentResourceManager.GetAgentResource()
	nextCacheUpdateTime := time.Now().Add(7 * 24 * time.Hour)
	util.SetAgentDetailsKey(agentInstance, "cacheUpdateTime", strconv.FormatInt(nextCacheUpdateTime.UnixNano(), 10))
	agent.apicClient.CreateSubResource(agentInstance.ResourceMeta, util.GetSubResourceDetails(agentInstance))
	logger.Tracef("setting next cache update time to - %s", time.Unix(0, nextCacheUpdateTime.UnixNano()).Format("2006-01-02 15:04:05.000000"))
}

func (es *EventSync) RebuildCache() {
	// SDB - NOTE : Do we need to pause jobs.
	logger.Info("rebuild cache")

	// close window so discovery doesn't happen during this cache rebuild
	PublishingLock()
	defer PublishingUnlock()

	es.waitForCacheRebuild()
}

// pausedInitCache pauses the event listener, calls initCache, then resumes via defer.
// A separate function ensures ResumeListener is always called even on panic.
func (es *EventSync) pausedInitCache(filters ...management.WatchTopicSpecFilters) error {
	if es.listenerPauser != nil {
		es.listenerPauser.PauseListener()
		defer es.listenerPauser.ResumeListener()
	}
	return es.initCache(filters...)
}

// waitForCacheRebuild continuously attempts to rebuild the cache until successful
func (es *EventSync) waitForCacheRebuild() {
	adjustment := 2
	maxBackoff := 5 * time.Minute
	currentBackoff := 30 * time.Second
	for {
		err := es.pausedInitCache()
		if err == nil {
			return
		}

		logger.
			WithError(err).
			WithField("waitTime", currentBackoff.String()).
			Error("failed to rebuild cache, retrying after waitTime")
		time.Sleep(currentBackoff)
		currentBackoff = time.Duration(math.Min(float64(maxBackoff), float64(currentBackoff)*float64(adjustment)))
	}
}

// rebuildFilters rebuilds the cache for the given subset of filters, then persists the cache.
func (es *EventSync) rebuildFilters(filters []management.WatchTopicSpecFilters) error {
	if es.listenerPauser != nil {
		es.listenerPauser.PauseListener()
		defer es.listenerPauser.ResumeListener()
	}
	if err := es.discoveryCache.execute(filters...); err != nil {
		return err
	}
	agent.cacheManager.SaveCache()
	return nil
}

// validateCache runs the cache validator to check if the persisted cache is in sync.
// Returns the filters that are out of sync, plus a non-nil error if any failed.
func (es *EventSync) validateCache() ([]management.WatchTopicSpecFilters, error) {
	if es.cacheValidator == nil {
		return nil, nil
	}
	return es.cacheValidator.Execute()
}

// ValidateCache validates the cache against the API server.
// If the cache is fully valid, the timer is reset to 7 days from now.
// If only some filters are out of sync, only those kinds are rebuilt.
func (es *EventSync) ValidateCache() error {
	failedFilters, err := es.validateCache()
	if err != nil {
		logger.WithError(err).Info("periodic cache validation failed, rebuilding out-of-sync kinds")
		if rebuildErr := es.rebuildFilters(failedFilters); rebuildErr != nil {
			return rebuildErr
		}
	}
	es.resetCacheTimer()
	return nil
}

// validateAndRebuildCache validates the cache and rebuilds only the out-of-sync kinds.
// Called when connection to Engage is restored.
func (es *EventSync) validateAndRebuildCache() {
	if es.cacheValidator == nil {
		return
	}

	failedFilters, err := es.validateCache()
	if err != nil {
		logger.WithError(err).Info("cache validation failed on reconnect, rebuilding out-of-sync kinds")
		if rebuildErr := es.rebuildFilters(failedFilters); rebuildErr != nil {
			logger.WithError(rebuildErr).Error("targeted cache rebuild failed on reconnect, falling back to full rebuild")
			es.RebuildCache()
		}
	}
}

func (es *EventSync) startCentralEventProcessor() error {
	if agent.cfg.IsUsingGRPC() {
		return es.startStreamMode()
	}
	return es.startPollMode()
}

func (es *EventSync) startPollMode() error {
	handlers := newHandlers()

	pc, err := poller.NewPollClient(
		agent.apicClient,
		agent.cfg,
		handlers,
		poller.WithHarvester(es.harvester, es.sequence, es.watchTopic.GetSelfLink()),
		poller.WithOnClientStop(es.RebuildCache),
		poller.WithOnConnect(),
		poller.WithOnReconnect(es.validateAndRebuildCache),
	)

	if err != nil {
		return fmt.Errorf("could not start the harvester poll client: %s", err)
	}

	if util.IsNotTest() {
		newEventProcessorJob(pc, "Poll Client")
	}

	return err
}

func (es *EventSync) startStreamMode() error {
	handlers := newHandlers()

	sc, err := stream.NewStreamerClient(
		agent.apicClient,
		agent.cfg,
		agent.tokenRequester,
		handlers,
		stream.WithOnStreamConnection(),
		stream.WithEventSyncError(es.RebuildCache),
		stream.WithOnReconnect(es.validateAndRebuildCache),
		stream.WithWatchTopic(es.watchTopic),
		stream.WithHarvester(es.harvester, es.sequence),
		stream.WithCacheManager(agent.cacheManager),
		stream.WithUserAgent(GetUserAgent()),
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	es.listenerPauser = sc
	agent.streamer = sc

	if util.IsNotTest() {
		newEventProcessorJob(sc, "Stream Client")
	}

	return err
}
