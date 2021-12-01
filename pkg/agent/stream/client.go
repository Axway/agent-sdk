package stream

import (
	"fmt"

	"github.com/Axway/agent-sdk/pkg/jobs"
	wm "github.com/Axway/agent-sdk/pkg/watchmanager"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

// streamer interface for starting a service
type streamer interface {
	Start() error
	HealthCheck() error
}

// Client a client for creating a grpc stream, and handling the received events.
type Client struct {
	manager  wm.Manager
	topic    string
	listener Listener
	events   chan *proto.Event
}

// NewClient creates a Client
func NewClient(
	topic string,
	manager wm.Manager,
	listener Listener,
	events chan *proto.Event,
) *Client {
	return &Client{
		manager:  manager,
		topic:    topic,
		listener: listener,
		events:   events,
	}
}

func (sc *Client) newStreamService() error {
	errors := make(chan error)

	id, err := sc.manager.RegisterWatch(sc.topic, sc.events, errors)
	if err != nil {
		return err
	}

	eventErrorCh := make(chan error)

	go func() {
		err := sc.listener.Listen()
		sc.manager.CloseWatch(id)
		eventErrorCh <- err
	}()

	select {
	case streamErr := <-errors:
		return streamErr
	case eventErr := <-eventErrorCh:
		return eventErr
	}
}

// Start starts the streaming client
func (sc *Client) Start() error {
	return sc.newStreamService()
}

// HealthCheck a health check endpoint for the connection to central.
func (sc *Client) HealthCheck() error {
	ok := sc.manager.Status()

	if !ok {
		return fmt.Errorf("grpc client is not connected to central")
	}

	return nil
}

// NewClientStreamJob creates a job for the stream client
func NewClientStreamJob(starter streamer) jobs.Job {
	return &ClientStreamJob{
		starter: starter,
	}
}

// ClientStreamJob job wrapper for a client that starts a stream and an event manager.
type ClientStreamJob struct {
	starter streamer
}

// Execute starts the stream
func (j ClientStreamJob) Execute() error {
	return j.starter.Start()
}

// Status gets the status
func (j ClientStreamJob) Status() error {
	return j.starter.HealthCheck()
}

// Ready checks if the job to start the stream is ready
func (j ClientStreamJob) Ready() bool {
	return true
}
