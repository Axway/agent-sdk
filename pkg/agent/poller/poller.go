package poller

import (
	"context"
	"fmt"
	"sync"
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
	isReady       bool
	lock          sync.RWMutex
}

type newPollExecutorFunc func(interval time.Duration, options ...executorOpt) *pollExecutor

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
	m.logger.Trace("register watch topic for polling and publishing events")
	if m.harvester == nil {
		go func() {
			m.Stop()
			errChan <- fmt.Errorf("harvester is not configured for the polling client")
		}()
		return
	}

	if m.sequence.GetSequence() < 0 {
		m.onHarvesterErr()
		go func() {
			m.Stop()
			errChan <- fmt.Errorf("do not have a sequence id, stopping poller")
		}()
		return
	}

	if err := m.harvester.EventCatchUp(m.topicSelfLink, eventChan); err != nil {
		m.logger.WithError(err).Error("harvester returned an error when syncing events")
		m.onHarvesterErr()
		go func() {
			m.Stop()
			errChan <- err
		}()
		return
	}

	m.lock.Lock()
	m.isReady = true
	m.lock.Unlock()

	go func() {
		err := m.sync(m.topicSelfLink, eventChan)
		m.Stop()
		errChan <- err
	}()
}

func (m *pollExecutor) sync(topicSelfLink string, eventChan chan *proto.Event) error {
	m.logger.Trace("sync events")

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

func (m *pollExecutor) tick(topicSelfLink string, eventChan chan *proto.Event) (ret error) {
	sequence := m.sequence.GetSequence()
	logger := m.logger.WithField("sequence-id", sequence)
	logger.Debug("retrieving harvester events")

	defer func() {
		if ret == nil {
			m.timer.Reset(m.interval)
		}
	}()

	if lastSeqID, err := m.harvester.ReceiveSyncEvents(topicSelfLink, sequence, eventChan); err != nil {
		if _, ok := err.(*harvester.ErrSeqGone); ok {
			m.sequence.SetSequence(lastSeqID)
			return
		}

		logger.WithError(err).Error("harvester returned an error when syncing events")
		m.onHarvesterErr()
		ret = err
	}

	return
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

	m.lock.Lock()
	defer m.lock.Unlock()
	m.isReady = false

	m.logger.Debug("poller has been stopped")
}

// Status returns a bool indicating the status of the poller
func (m *pollExecutor) Status() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if m.ctx.Err() != nil {
		return false
	}

	return m.isReady
}
