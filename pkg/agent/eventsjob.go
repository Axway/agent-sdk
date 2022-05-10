package agent

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
)

// constants for retry interval for stream job
const (
	defaultRetryInterval = 100 * time.Millisecond
	maxRetryInterval     = 5 * time.Minute
	clientStreamJobName  = "Stream Client"
)

// eventsJob interface for a job to execute to retrieve events in either stream or poll mode
type eventsJob interface {
	Start() error
	Status() error
	Stop()
	Healthcheck(_ string) *hc.Status
}

// eventProcessorJob job wrapper for a streamerClient that starts a stream and an event manager.
type eventProcessorJob struct {
	streamer      eventsJob
	stop          chan interface{}
	jobID         string
	retryInterval time.Duration
}

// newEventProcessorJob creates a job for the streamerClient
func newEventProcessorJob(streamer eventsJob) jobs.Job {
	streamJob := &eventProcessorJob{
		streamer:      streamer,
		stop:          make(chan interface{}),
		retryInterval: defaultRetryInterval,
	}
	streamJob.jobID, _ = jobs.RegisterDetachedChannelJobWithName(streamJob, streamJob.stop, clientStreamJobName)

	return streamJob
}

// Execute starts the stream
func (j *eventProcessorJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
		j.renewRegistration()
	}()

	return j.streamer.Start()
}

// Status gets the status
func (j *eventProcessorJob) Status() error {
	status := j.streamer.Status()
	if status == nil {
		j.retryInterval = defaultRetryInterval
	}
	return status
}

// Ready checks if the job to start the stream is ready
func (j *eventProcessorJob) Ready() bool {
	return true
}

func (j *eventProcessorJob) renewRegistration() {
	if j.jobID != "" {
		jobs.UnregisterJob(j.jobID)
		j.jobID = ""

		j.retryInterval = j.retryInterval * 2
		if j.retryInterval > maxRetryInterval {
			j.retryInterval = defaultRetryInterval
		}

		time.AfterFunc(j.retryInterval, func() {
			j.jobID, _ = jobs.RegisterDetachedChannelJobWithName(j, j.stop, clientStreamJobName)
		})
		return
	}
}
