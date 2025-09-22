package poller

import (
	"context"

	"github.com/Axway/agent-sdk/pkg/agent/events"
	"github.com/Axway/agent-sdk/pkg/harvester"
	"github.com/Axway/agent-sdk/pkg/util"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

// ClientOpt func for setting fields on the PollClient
type ClientOpt func(pc *PollClient)

// WithHarvester configures the polling client to use harvester
func WithHarvester(hClient harvester.Harvest, sequence events.SequenceProvider, topicSelfLink string) ClientOpt {
	return func(pc *PollClient) {
		pc.harvesterConfig.hClient = hClient
		pc.harvesterConfig.topicSelfLink = topicSelfLink
		pc.harvesterConfig.sequence = sequence
	}
}

// WithOnClientStop func to execute when the client shuts down
func WithOnClientStop(cb func()) ClientOpt {
	return func(pc *PollClient) {
		pc.onClientStop = cb
	}
}

// WithOnConnect func to execute when a connection to central is made
func WithOnConnect() ClientOpt {
	return func(pc *PollClient) {
		pc.onStreamConnection = func() {
			hc.RegisterHealthcheck(util.AmplifyCentral, util.CentralHealthCheckEndpoint, pc.Healthcheck)
		}
	}
}

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

func WithContext(ctx context.Context, cancel context.CancelFunc) executorOpt {
	return func(m *pollExecutor) {
		m.ctx = ctx
		m.cancel = cancel
	}
}
