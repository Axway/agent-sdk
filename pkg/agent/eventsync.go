package agent

import (
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
	mpEnabled      bool
	watchTopic     *management.WatchTopic
	sequence       events.SequenceProvider
	harvester      harvester.Harvest
	discoveryCache *discoveryCache
}

// NewEventSync creates an EventSync
func NewEventSync() (*EventSync, error) {
	migrations := []migrate.Migrator{}

	// Make sure only DA and Governance agents run migration processes
	runMigrations := agent.cfg.GetAgentType() != config.TraceabilityAgent

	// Check if marketplace is enabled
	isMpEnabled := agent.agentFeaturesCfg != nil && agent.agentFeaturesCfg.MarketplaceProvisioningEnabled()

	if runMigrations {
		// add attribute migration to migrations
		attributeMigration := migrate.NewAttributeMigration(agent.apicClient, agent.cfg)
		ardMigration := migrate.NewArdMigration(agent.apicClient, agent.cfg)
		apisiMigration := migrate.NewAPISIMigration(agent.apicClient, agent.cfg)
		instanceMigration := migrate.NewInstanceMigration(agent.apicClient, agent.cfg)
		migrations = append(migrations, attributeMigration, ardMigration, apisiMigration, instanceMigration)

		if isMpEnabled {
			// add marketplace migration to migrations
			marketplaceMigration := migrate.NewMarketplaceMigration(agent.apicClient, agent.cfg, agent.cacheManager)
			agent.marketplaceMigration = marketplaceMigration
			migrations = append(migrations, marketplaceMigration)
		}
	}

	mig := migrate.NewMigrateAll(migrations...)

	opts := []discoveryOpt{
		withMigration(mig),
		withMpEnabled(isMpEnabled),
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

	return &EventSync{
		mpEnabled:      isMpEnabled,
		watchTopic:     wt,
		sequence:       sequence,
		harvester:      hClient,
		discoveryCache: discoveryCache,
	}, nil
}

// SyncCache initializes agent cache and starts the agent in stream or poll mode
func (es *EventSync) SyncCache() error {
	if !agent.cacheManager.HasLoadedPersistedCache() {
		if err := es.initCache(); err != nil {
			return err
		}
	}

	err := registerExternalIDPs()
	if err != nil {
		logger.WithError(err).Warn("failed to register CRDs for external IdP config")
	}

	err = es.startCentralEventProcessor()
	if err != nil {
		return err
	}

	return es.registerInstanceValidator()
}

func (es *EventSync) registerInstanceValidator() error {
	if agent.apiValidatorJobID == "" && agent.cfg.GetAgentType() == config.DiscoveryAgent {
		jobID, err := jobs.RegisterScheduledJobWithName(newInstanceValidator(), agent.cfg.GetAPIValidationCronSchedule(), "API service instance validator")
		agent.apiValidatorJobID = jobID
		return err
	}
	return nil
}

func (es *EventSync) initCache() error {
	seqID, err := es.harvester.ReceiveSyncEvents(es.watchTopic.GetSelfLink(), 0, nil)
	if err != nil {
		return err
	}
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

	agentInstance := agent.agentResourceManager.GetAgentResource()

	// add 7 days to the current date for the next rebuild cache
	nextCacheUpdateTime := time.Now().Add(7 * 24 * time.Hour)

	// persist cacheUpdateTime
	util.SetAgentDetailsKey(agentInstance, "cacheUpdateTime", strconv.FormatInt(nextCacheUpdateTime.UnixNano(), 10))
	agent.apicClient.CreateSubResource(agentInstance.ResourceMeta, util.GetSubResourceDetails(agentInstance))
	logger.Tracef("setting next cache update time to - %s", time.Unix(0, nextCacheUpdateTime.UnixNano()).Format("2006-01-02 15:04:05.000000"))
	return nil
}

func (es *EventSync) RebuildCache() {
	// SDB - NOTE : Do we need to pause jobs.
	logger.Info("rebuild cache")

	// close window so discovery doesn't happen during this cache rebuild
	PublishingLock()
	defer PublishingUnlock()

	if err := es.initCache(); err != nil {
		logger.WithError(err).Error("failed to rebuild cache")
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
		stream.WithWatchTopic(es.watchTopic),
		stream.WithHarvester(es.harvester, es.sequence),
		stream.WithCacheManager(agent.cacheManager),
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	agent.streamer = sc

	if util.IsNotTest() {
		newEventProcessorJob(sc, "Stream Client")
	}

	return err
}
