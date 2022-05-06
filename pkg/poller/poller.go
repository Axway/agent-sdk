package poller

import (
	"context"
	"time"

	"github.com/Axway/agent-sdk/pkg/harvester"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/google/uuid"
)

// Poller interface for starting a polling service
type Poller interface {
	Start() error
	Status() error
	Stop()
	HealthCheck(_ string) *hc.Status
}

type manager struct {
	harvester harvester.Harvest
	logger    log.FieldLogger
	timer     *time.Timer
	sequence  harvester.SequenceProvider
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewPollManager(cfg *harvester.Config, interval time.Duration) *manager {
	logger := log.NewFieldLogger().
		WithComponent("manager").
		WithPackage("poller")

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
			m.logger.Debug("retrieving harvester events")
			seqID := m.sequence.GetSequence()
			seqID, err := m.harvester.ReceiveSyncEvents(topic, seqID, eventChan)
			m.sequence.SetSequence(seqID)
			if err != nil {
				return err
			}
		}
	}
}

func (m *manager) CloseWatch(_ string) error {
	m.cancel()
	return nil
}

func (m *manager) Status() bool {
	return m.ctx.Err() == nil
}

// HealthCheck - health check poll client
func (m *manager) HealthCheck(_ string) *hc.Status {
	ok := m.Status()
	if !ok {
		return &hc.Status{
			Result:  hc.FAIL,
			Details: "harvester client is not connected to central and unable to poll events",
		}
	}
	return &hc.Status{
		Result: hc.OK,
	}
}
