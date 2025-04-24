package jobs

import (
	"github.com/Axway/agent-sdk/pkg/util/errors"
)

type retryJobProps struct {
	retries int
}

type retryJob struct {
	baseJob
	retryJobProps
}

// newBaseJob - creates a single run job and sets up the structure for different job types
func newRetryJob(newJob Job, retries int, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := retryJob{
		createBaseJob(newJob, failJobChan, name, JobTypeRetry),
		retryJobProps{
			retries: retries,
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

// start - calls the Execute function from the Job definition
func (b *retryJob) start() {
	b.startLog()
	b.waitForReady()

	b.SetStatus(JobStatusRunning)
	for i := 0; i < b.retries; i++ {
		b.executeJob()
		if b.err.Load() == nil {
			// job was successful
			b.SetStatus(JobStatusFinished)
			return
		}
		b.setExecutionRetryError()
		b.SetStatus(JobStatusRetrying)
	}

	b.SetStatus(JobStatusFailed)
}

// stop - noop
func (b *retryJob) stop() {
	b.stopLog()
}

func (b *retryJob) setExecutionRetryError() {
	b.setError(errors.Wrap(ErrExecutingJob, b.getError().Error()).FormatError(b.jobType, b.id))
}
