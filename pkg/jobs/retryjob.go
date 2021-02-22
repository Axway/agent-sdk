package jobs

import (
	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type retryJobProps struct {
	retries int
}

type retryJob struct {
	baseJob
	retryJobProps
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newRetryJob(newJob Job, retries int, failJobChan chan string) (JobExecution, error) {
	thisJob := retryJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		retryJobProps{
			retries: retries,
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

//start - calls the Execute function from the Job definition
func (b *retryJob) start() {
	log.Debugf("Starting %v job %v", JobTypeRetry, b.id)
	b.waitForReady()

	b.SetStatus(JobStatusRunning)
	for i := 0; i < b.retries; i++ {
		b.executeJob()
		if b.err == nil {
			// job was successful
			return
		}
		b.err = errors.Wrap(ErrExecutingRetryJob, b.err.Error()).FormatError(JobTypeRetry, b.id, b.retries)
		b.SetStatus(JobStatusRetrying)
	}
	b.SetStatus(JobStatusFailed)
}

//stop - noop
func (b *retryJob) stop() {
	log.Debugf("Stopping %v job %v", JobTypeRetry, b.id)
	return
}
