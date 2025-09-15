package agent

import (
	"sync/atomic"
	"time"

	"github.com/Axway/agent-sdk/pkg/jobs"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// constants for retry interval for stream job
const (
	defaultRetryInterval   = 5 * time.Second
	maxRetryInterval       = 5 * time.Minute
	maxNumRetryForInterval = 3
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
	logger        log.FieldLogger
	streamer      eventsJob
	stop          chan interface{}
	jobID         atomic.Value
	retryInterval time.Duration
	numRetry      int
	name          string
}

// newEventProcessorJob creates a job for the streamerClient
func newEventProcessorJob(eventJob eventsJob, name string) jobs.Job {
	streamJob := &eventProcessorJob{
		logger:        log.NewFieldLogger().WithComponent("eventProcessorJob").WithPackage("agent"),
		streamer:      eventJob,
		stop:          make(chan interface{}, 1),
		retryInterval: defaultRetryInterval,
		numRetry:      0,
		name:          name,
		jobID:         atomic.Value{},
	}

	jobID, err := jobs.RegisterDetachedChannelJobWithName(streamJob, streamJob.stop, name)
	if err != nil {
		streamJob.logger.WithError(err).Error("failed to register")
	}
	streamJob.jobID.Store(jobID)
	jobs.RegisterIntervalJobWithName(newCentralHealthCheckJob(eventJob), time.Second*3, "Central Health Check")

	return streamJob
}

// Execute starts the stream
func (j *eventProcessorJob) Execute() error {
	go func() {
		<-j.stop
		j.streamer.Stop()
		j.renewRegistration()
	}()

	err := j.streamer.Start()
	if err != nil {
		return err
	}
	return nil
}

// Status gets the status
func (j *eventProcessorJob) Status() error {
	status := j.streamer.Status()
	if status == nil {
		j.retryInterval = defaultRetryInterval
		j.numRetry = 0
	}
	return status
}

// Ready checks if the job to start the stream is ready
func (j *eventProcessorJob) Ready() bool {
	return true
}

func (j *eventProcessorJob) renewRegistration() {
	defer time.AfterFunc(j.retryInterval, func() {
		jobID, err := jobs.RegisterDetachedChannelJobWithName(j, j.stop, j.name)
		if err != nil {
			j.logger.WithError(err).Error("failed to re-register")
		}
		j.jobID.Store(jobID)
	})

	jobID := j.jobID.Load().(string)
	if jobID == "" {
		return
	}

	j.logger.WithField("jobID", jobID).Trace("unregistering")
	defer j.logger.Info("renewing registration")
	jobs.UnregisterJob(jobID)

	j.jobID.Store("")
	j.numRetry++
	if j.numRetry == maxNumRetryForInterval {
		j.numRetry = 0
		j.retryInterval = j.retryInterval * 2
		if j.retryInterval > maxRetryInterval {
			j.retryInterval = maxRetryInterval
		}
	}
}
