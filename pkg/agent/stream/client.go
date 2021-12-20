package stream

import (
	"github.com/Axway/agent-sdk/pkg/jobs"
)

// Streamer interface for starting a service
type Streamer interface {
	Start() error
	Status() error
	Stop()
}

// NewClientStreamJob creates a job for the stream client
func NewClientStreamJob(streamer Streamer, stop chan interface{}) jobs.Job {
	return &ClientStreamJob{
		streamer: streamer,
		stop:     stop,
	}
}

// ClientStreamJob job wrapper for a client that starts a stream and an event manager.
type ClientStreamJob struct {
	streamer Streamer
	stop     chan interface{}
}

// Execute starts the stream
func (j *ClientStreamJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
	}()

	return j.streamer.Start()
}

// Status gets the status
func (j *ClientStreamJob) Status() error {
	return j.streamer.Status()
}

// Ready checks if the job to start the stream is ready
func (j *ClientStreamJob) Ready() bool {
	return true
}
