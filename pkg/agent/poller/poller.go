package poller

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/google/uuid"
)

type manager struct {
	harvester harvester.Harvest
	logger    log.FieldLogger
	timer     *time.Timer
	sequence  events.SequenceProvider
	ctx       context.Context
	cancel    context.CancelFunc
}

func newPollManager(cfg *harvester.Config, interval time.Duration) *manager {
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
	}
}

func (m *manager) RegisterWatch(topic string, eventChan chan *proto.Event, errChan chan error) (string, error) {
	subscriptionID, _ := uuid.NewUUID()
	subID := subscriptionID.String()

	go func() {
		err := m.sync(topic, eventChan)
		errChan <- err
		m.cancel()
	}()

	return subID, nil
}

func (m *manager) sync(topic string, eventChan chan *proto.Event) error {
	for {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		case <-m.timer.C:
			m.logger.Trace("retrieving harvester events")
			seqID := m.sequence.GetSequence()
			seqID, err := m.harvester.ReceiveSyncEvents(topic, seqID, eventChan)
			m.sequence.SetSequence(seqID)
			if err != nil {
				return err
			}
		}
	}
}

func (m *manager) Stop() {
	m.cancel()
}

func (m *manager) Status() bool {
	return m.ctx.Err() == nil
}
