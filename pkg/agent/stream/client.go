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
	Status() error
	Stop()
}

// Client a client for creating a grpc stream, and handling the received events.
type Client struct {
	manager  wm.Manager
	topic    string
	listener Listener
	events   chan *proto.Event
	ids      []string
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
		ids:      make([]string, 0),
	}
}

func (sc *Client) newStreamService() error {
	streamErrCh := make(chan error)

	id, err := sc.manager.RegisterWatch(sc.topic, sc.events, streamErrCh)
	if err != nil {
		return err
	}

	sc.ids = append(sc.ids, id)

	eventErrorCh := make(chan error)

	go func() {
		err := sc.listener.Listen()
		eventErrorCh <- err
	}()

	select {
	case streamErr := <-streamErrCh:
		sc.manager.CloseWatch(id)
		return streamErr
	case eventErr := <-eventErrorCh:
		sc.manager.CloseWatch(id)
		return eventErr
	}
}

// Stop stops all watch clients created by this stream client
func (sc *Client) Stop() {
	sc.listener.Stop()

	for _, id := range sc.ids {
		sc.manager.CloseWatch(id)
	}

	sc.ids = make([]string, 0)
}

// Start starts the streaming client
func (sc *Client) Start() error {
	return sc.newStreamService()
}

// Status a health check endpoint for the connection to central.
func (sc *Client) Status() error {
	ok := sc.manager.Status()

	if !ok {
		return fmt.Errorf("grpc client is not connected to central")
	}

	return nil
}

// NewClientStreamJob creates a job for the stream client
func NewClientStreamJob(streamer streamer, stop chan interface{}) jobs.Job {
	return &ClientStreamJob{
		streamer: streamer,
		stop:     stop,
	}
}

// ClientStreamJob job wrapper for a client that starts a stream and an event manager.
type ClientStreamJob struct {
	streamer streamer
	stop     chan interface{}
}

// Execute starts the stream
func (j ClientStreamJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
	}()

	return j.streamer.Start()
}

// Status gets the status
func (j ClientStreamJob) Status() error {
	return j.streamer.Status()
}

// Ready checks if the job to start the stream is ready
func (j ClientStreamJob) Ready() bool {
	return true
}
