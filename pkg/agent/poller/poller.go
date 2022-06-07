package poller

import (
	"context"
	"fmt"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type pollExecutor struct {
	harvester     harvester.Harvest
	sequence      events.SequenceProvider
	topicSelfLink string
	logger        log.FieldLogger
	timer         *time.Timer
	ctx           context.Context
	cancel        context.CancelFunc
	interval      time.Duration
	onStop        onClientStopCb
}

type newPollExecutorFunc func(interval time.Duration, options ...executorOpt) *pollExecutor

type executorOpt func(m *pollExecutor)

func withHarvester(cfg harvesterConfig) executorOpt {
	return func(m *pollExecutor) {
		m.harvester = cfg.hClient
		m.sequence = cfg.sequence
		m.topicSelfLink = cfg.topicSelfLink
	}
}

func withOnStop(cb onClientStopCb) executorOpt {
	return func(m *pollExecutor) {
		m.onStop = cb
	}
}

func newPollExecutor(interval time.Duration, options ...executorOpt) *pollExecutor {
	logger := log.NewFieldLogger().
		WithComponent("pollExecutor").
		WithPackage("sdk.agent.poller")

	ctx, cancel := context.WithCancel(context.Background())

	pm := &pollExecutor{
		logger:   logger,
		timer:    time.NewTimer(interval),
		ctx:      ctx,
		cancel:   cancel,
		interval: interval,
	}

	for _, opt := range options {
		opt(pm)
	}

	return pm
}

// RegisterWatch registers a watch topic for polling events and publishing events on a channel
func (m *pollExecutor) RegisterWatch(eventChan chan *proto.Event, errChan chan error) {
	if m.harvester == nil {
		go func() {
			m.Stop()
			errChan <- fmt.Errorf("harvester is not configured for the polling client")
		}()
		return
	}

	go func() {
		err := m.sync(m.topicSelfLink, eventChan)
		m.Stop()
		errChan <- err
	}()
}

func (m *pollExecutor) sync(topicSelfLink string, eventChan chan *proto.Event) error {
	if err := m.harvester.EventCatchUp(topicSelfLink, eventChan); err != nil {
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
			if err := m.tick(topicSelfLink, eventChan); err != nil {
				return err
			}
		}
	}
}

func (m *pollExecutor) tick(topicSelfLink string, eventChan chan *proto.Event) error {
	sequence := m.sequence.GetSequence()
	logger := m.logger.WithField("sequenceID", sequence)
	logger.Debug("retrieving harvester events")

	if _, err := m.harvester.ReceiveSyncEvents(topicSelfLink, sequence, eventChan); err != nil {
		logger.WithError(err).Error("harvester returned an error when syncing events")
		m.onHarvesterErr()
		return err
	}

	m.timer.Reset(m.interval)
	return nil
}

func (m *pollExecutor) onHarvesterErr() {
	if m.onStop == nil {
		return
	}

	m.onStop()
}

// Stop stops the poller
func (m *pollExecutor) Stop() {
	m.timer.Stop()
	m.cancel()
	m.logger.Debug("poller has been stopped")
}

// Status returns a bool indicating the status of the poller
func (m *pollExecutor) Status() bool {
	return m.ctx.Err() == nil
}
