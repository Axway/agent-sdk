package poller

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type manager struct {
	harvester harvester.Harvest
	logger    log.FieldLogger
	timer     *time.Timer
	sequence  events.SequenceProvider
	ctx       context.Context
	cancel    context.CancelFunc
	interval  time.Duration
	onStop    onClientStopCb
}

type newPollManagerFunc func(cfg *harvester.Config, interval time.Duration, onStop onClientStopCb) *manager

func newPollManager(cfg *harvester.Config, interval time.Duration, onStop onClientStopCb) *manager {
	logger := log.NewFieldLogger().
		WithComponent("manager").
		WithPackage("sdk.agent.poller")

	ctx, cancel := context.WithCancel(context.Background())

	return &manager{
		harvester: harvester.NewClient(cfg),
		logger:    logger,
		timer:     time.NewTimer(interval),
		ctx:       ctx,
		cancel:    cancel,
		interval:  interval,
		sequence:  cfg.SequenceProvider,
		onStop:    onStop,
	}
}

// RegisterWatch registers a watch topic for polling events and publishing events on a channel
func (m *manager) RegisterWatch(topic string, eventChan chan *proto.Event, errChan chan error) {
	go func() {
		err := m.sync(topic, eventChan)
		m.Stop()
		errChan <- err
	}()
}

func (m *manager) sync(topic string, eventChan chan *proto.Event) error {
	if err := m.harvester.EventCatchUp(topic, eventChan); err != nil {
		m.logger.WithError(err).Error("harvester returned an error when syncing events")
		m.onHarvesterErr()
		return err
	}

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("harvester polling has been stopped")
			return nil
		case <-m.timer.C:
			if err := m.tick(topic, eventChan); err != nil {
				return err
			}
		}
	}
}

func (m *manager) tick(topic string, eventChan chan *proto.Event) error {
	seq := m.sequence.GetSequence()
	logger := m.logger.WithField("sequenceID", seq)
	logger.Debug("retrieving harvester events")

	_, err := m.harvester.ReceiveSyncEvents(topic, seq, eventChan)
	if err != nil {
		logger.WithError(err).Error("harvester returned an error when syncing events")
		m.onHarvesterErr()
		return err
	}

	m.timer.Reset(m.interval)
	return nil
}

func (m *manager) onHarvesterErr() {
	if m.onStop == nil {
		return
	}

	m.onStop()
}

// Stop stops the poller
func (m *manager) Stop() {
	m.timer.Stop()
	m.cancel()
	m.logger.Debug("poller has been stopped")
}

// Status returns a bool indicating the status of the poller
func (m *manager) Status() bool {
	return m.ctx.Err() == nil
}
