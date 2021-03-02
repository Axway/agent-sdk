package jobs

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type intervalJobProps struct {
	interval time.Duration
	stopChan chan bool
}

type intervalJob struct {
	baseJob
	intervalJobProps
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newIntervalJob(newJob Job, interval time.Duration, failJobChan chan string) (JobExecution, error) {
	thisJob := intervalJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		intervalJobProps{
			interval: interval,
			stopChan: make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

//start - calls the Execute function from the Job definition
func (b *intervalJob) start() {
	log.Debugf("Starting %v job %v", JobTypeInterval, b.id)
	b.waitForReady()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	b.SetStatus(JobStatusRunning)
	for {
		// Non-blocking channel read, if stopped then exit
		select {
		case <-b.stopChan:
			b.SetStatus(JobStatusStopped)
			return
		case <-ticker.C:
			b.executeCronJob()
			if b.err != nil {
				b.err = errors.Wrap(ErrExecutingJob, b.err.Error()).FormatError(JobTypeInterval, b.id)
				log.Error(b.err)
				b.SetStatus(JobStatusStopped)
			}
			ticker.Stop()
			ticker = time.NewTicker(b.interval)
		}
	}
}

//stop - write to the stop channel to stop the execution loop
func (b *intervalJob) stop() {
	log.Debugf("Stopping %v job %v", JobTypeInterval, b.id)
	b.stopChan <- true
}
