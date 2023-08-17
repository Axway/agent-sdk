package agent

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/agent/poller"
	"github.com/Axway/agent-sdk/pkg/agent/stream"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/definitions"
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

	err := es.startCentralEventProcessor()
	if err != nil {
		return err
	}

	return es.registerInstanceValidator()
}

func (es *EventSync) registerInstanceValidator() error {
	if agent.apiValidatorJobID == "" && agent.cfg.GetAgentType() == config.DiscoveryAgent {
		jobID, err := jobs.RegisterIntervalJobWithName(newInstanceValidator(), agent.cfg.GetAPIValidationFrequency(), "API service instance validator")
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
	return nil
}

func (es *EventSync) RebuildCache() {
	// SDB - NOTE : Do we need to pause jobs.
	logger.Info("rebuild cache")

	agent.cacheManager.Flush()
	if err := es.initCache(); err != nil {
		logger.WithError(err).Error("failed to rebuild cache")
	}

	agentInstance := agent.agentResourceManager.GetAgentResource()
	agentDetails := util.GetAgentDetails(agentInstance)
	if agentDetails == nil {
		agentDetails = make(map[string]interface{})
	}

	cacheUpdateTime := int64(0)

	// update cacheUpdateTime in agent resource
	value, exits := agentDetails["cacheUpdateTime"]
	if !exits {
		// cache update time hasn't been established yet.  Update time to 7 days from current now time
		currentTime := time.Now().Add(7 * 24 * time.Hour)
		cacheUpdateTime = currentTime.UnixNano()
	} else {
		// if the cache update time already exists, update to 7 days from current cache update time
		parsedTime, _ := time.Parse(time.RFC3339, value.(string))
		parsedTime.Add(7 * 24 * time.Hour)
		cacheUpdateTime = parsedTime.UnixNano()
	}

	// persist cacheUpdateTime
	agentDetails["cacheUpdateTime"] = strconv.FormatInt(cacheUpdateTime, 10)
	agent.apicClient.CreateSubResource(agentInstance.ResourceMeta, map[string]interface{}{definitions.XAgentDetails: agentDetails})
	logger.Tracef("setting next cache update time to - %s", time.Unix(0, cacheUpdateTime).Format("2006-01-02 15:04:05.000000"))
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
