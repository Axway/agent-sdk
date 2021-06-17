package jobs

import (
	"time"
)

//newDetachedIntervalJob - creates an interval run job, detached from other cron jobs
func newDetachedIntervalJob(newJob Job, interval time.Duration) (JobExecution, error) {
	thisJob := intervalJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			jobType:  JobTypeDetachedInterval,
			status:   JobStatusInitializing,
			failChan: nil,
		},
		intervalJobProps{
			interval: interval,
			stopChan: make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}
