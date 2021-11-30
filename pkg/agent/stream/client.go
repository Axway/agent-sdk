package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// starter interface for starting a service
type starter interface {
	Start() error
}

// Client a client for creating a grpc stream, and handling the received events.
type Client struct {
	handlers        []Handler
	manager         wm.Manager
	newEventManager eventManagerFunc
	resourceClient  ResourceClient
	topic           string
	stopCh          chan interface{}
}

// NewClient creates a Client
func NewClient(
	topic string,
	manager wm.Manager,
	rc ResourceClient,
	stopCh chan interface{},
	handlers ...Handler,
) *Client {
	return &Client{
		handlers:        handlers,
		manager:         manager,
		newEventManager: NewEventListener,
		resourceClient:  rc,
		stopCh:          stopCh,
		topic:           topic,
	}
}

func (sc *Client) newStreamService() error {
	events, errors := make(chan *proto.Event), make(chan error)

	em := sc.newEventManager(
		events,
		sc.stopCh,
		sc.resourceClient,
		sc.handlers...,
	)

	_, err := sc.manager.RegisterWatch(sc.topic, events, errors)
	if err != nil {
		return err
	}

	return em.Listen()
}

// Start starts the streaming client
func (sc *Client) Start() error {
	return sc.newStreamService()
}

// HealthCheck a health check endpoint for the connection to central.
func (sc *Client) HealthCheck() hc.CheckStatus {
	return func(_ string) *hc.Status {
		ok := sc.manager.Status()
		status := &hc.Status{
			Result: hc.OK,
		}

		if !ok {
			status.Result = hc.FAIL
			status.Details = "grpc client is not connected to central"
		}

		return status
	}
}

// NewClientStreamJob creates a job for the stream client
func NewClientStreamJob(starter starter, getHealthStatus hc.GetStatusLevel) jobs.Job {
	return &ClientStreamJob{
		starter:         starter,
		getHealthStatus: getHealthStatus,
	}
}

// ClientStreamJob job wrapper for a client that starts a stream and an event manager.
type ClientStreamJob struct {
	starter         starter
	getHealthStatus hc.GetStatusLevel
}

// Execute starts the stream
func (j ClientStreamJob) Execute() error {
	return j.starter.Start()
}

// Status gets the status
func (j ClientStreamJob) Status() error {
	status := j.getHealthStatus("central")

	if status != hc.OK {
		return fmt.Errorf("central is in %s state", status)
	}

	return nil
}

// Ready checks if the job to start the stream is ready
func (j ClientStreamJob) Ready() bool {
	return true
}
