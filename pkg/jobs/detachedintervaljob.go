package jobs

import (
	"time"
)

// newDetachedIntervalJob - creates an interval run job, detached from other cron jobs
func newDetachedIntervalJob(newJob Job, interval time.Duration, name string, opts ...jobOpt) (JobExecution, error) {
	thisJob := intervalJob{
		createBaseJob(newJob, nil, name, JobTypeDetachedInterval),
		intervalJobProps{
			interval: interval,
			stopChan: make(chan bool, 1),
		},
	}

	for _, o := range opts {
		o(thisJob.baseJob)
	}

	go thisJob.start()
	return &thisJob, nil
}
