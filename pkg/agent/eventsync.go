package agent

import (
	"context"
	"fmt"
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
	cacheValidator CacheValidator
}

// CacheValidator is satisfied by cacheValidator; allows EventSync to run cache validation.
type CacheValidator interface {
	Execute() ([]management.WatchTopicSpecFilters, error)
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
	hClient := harvester.NewClient(hCfg, harvester.WithPublishLock(PublishingLock, PublishingUnlock))

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
		hClient,
		sequence,
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

	// when no events returned by harvester the seqID will be 0, so not updated in sequence manager
	if seqID > 0 {
		es.sequence.SetSequence(seqID)
	}

	defer agent.cacheManager.SaveCache()
	if err = es.discoveryCache.execute(failedFilters...); err != nil {
		return err
	}

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

// RebuildCache is the single entry point for all cache rebuilds.
// When filters are given it attempts a targeted rebuild first, falling back to full on error.
// The 7-day timer is reset on success.
func (es *EventSync) RebuildCache(filters ...management.WatchTopicSpecFilters) error {
	logger.Info("rebuild cache")

	// close window so discovery doesn't happen during this cache rebuild
	PublishingLock()
	defer PublishingUnlock()

	return es.initCache(filters...)
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
// Returns the out-of-sync filters and a non-nil error if any kind failed validation.
// If all kinds pass, the 7-day timer is reset.
func (es *EventSync) ValidateCache() ([]management.WatchTopicSpecFilters, error) {
	failedFilters, err := es.validateCache()
	if err == nil {
		es.resetCacheTimer()
	}
	return failedFilters, err
}

// validateAndRebuildCache validates the cache and rebuilds only the out-of-sync kinds.
// Called when connection to Central is restored.
func (es *EventSync) validateAndRebuildCache() error {
	if es.cacheValidator == nil {
		return nil
	}

	failedFilters, err := es.validateCache()
	if err != nil {
		logger.WithError(err).Info("cache validation failed on reconnect, rebuilding out-of-sync kinds")
		return es.RebuildCache(failedFilters...)
	}
	return nil
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
		poller.WithOnConnect(),
		poller.WithOnReconnect(es.validateAndRebuildCache),
	)

	if err != nil {
		return fmt.Errorf("could not start the harvester poll client: %s", err)
	}

	if util.IsNotTest() {
		newEventProcessorJob(pc, "Poll Client")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := pc.WaitForReady(ctx); err != nil {
			return fmt.Errorf("poll client did not connect to Central within timeout: %w", err)
		}
	}

	return nil
}

func (es *EventSync) startStreamMode() error {
	handlers := newHandlers()

	sc, err := stream.NewStreamerClient(
		agent.apicClient,
		agent.cfg,
		agent.tokenRequester,
		handlers,
		stream.WithOnStreamConnection(),
		stream.WithOnReconnect(es.validateAndRebuildCache),
		stream.WithWatchTopic(es.watchTopic),
		stream.WithHarvester(es.harvester, es.sequence),
		stream.WithCacheManager(agent.cacheManager),
		stream.WithUserAgent(GetUserAgent()),
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	agent.streamer = sc

	if util.IsNotTest() {
		newEventProcessorJob(sc, "Stream Client")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := sc.WaitForReady(ctx); err != nil {
			return fmt.Errorf("stream client did not connect to Central within timeout: %w", err)
		}
	}

	return nil
}
