package agent

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/poller"
	"github.com/Axway/agent-sdk/pkg/agent/stream"
	mv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/migrate"
)

// EventSync struct for syncing events from central
type EventSync struct {
	mpEnabled      bool
	watchTopic     *mv1.WatchTopic
	sequence       events.SequenceProvider
	harvester      harvester.Harvest
	discoveryCache *discoveryCache
}

// NewEventSync creates an EventSync
func NewEventSync() (*EventSync, error) {
	migrations := []migrate.Migrator{
		migrate.NewAttributeMigration(agent.apicClient, agent.cfg),
	}

	isMpEnabled := agent.agentFeaturesCfg != nil && agent.agentFeaturesCfg.MarketplaceProvisioningEnabled()

	if isMpEnabled {
		marketplaceMigration := migrate.NewMarketplaceMigration(agent.apicClient, agent.cfg, agent.cacheManager)
		agent.marketplaceMigration = marketplaceMigration
		migrations = append(migrations, marketplaceMigration)
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
	return es.startCentralEventProcessor()
}

func (es *EventSync) initCache() error {
	seqID, err := es.harvester.ReceiveSyncEvents(es.watchTopic.GetSelfLink(), 0, nil)
	if err != nil {
		return err
	}
	// event channel is not ready yet, so subtract one from the latest sequence id to process the event
	// when the poll/stream client is ready
	es.sequence.SetSequence(seqID - 1)
	err = es.discoveryCache.execute()
	if err != nil {
		return err
	}
	agent.cacheManager.SaveCache()
	return nil
}

func (es *EventSync) rebuildCache() {
	agent.cacheManager.Flush()
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
		poller.WithOnClientStop(es.rebuildCache),
		poller.WithOnConnect(),
	)

	if err != nil {
		return fmt.Errorf("could not start the harvester poll client: %s", err)
	}

	newEventProcessorJob(pc, "Poll Client")

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
		stream.WithEventSyncError(es.rebuildCache),
		stream.WithWatchTopic(es.watchTopic),
		stream.WithHarvester(es.harvester, es.sequence),
		stream.WithCacheManager(agent.cacheManager),
	)

	if err != nil {
		return fmt.Errorf("could not start the watch manager: %s", err)
	}

	agent.streamer = sc
	newEventProcessorJob(sc, "Stream Client")

	return err
}