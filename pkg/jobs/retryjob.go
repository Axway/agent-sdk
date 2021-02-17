package jobs

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
	b.waitForReady()

	b.status = JobStatusRunning
	for i := 0; i < b.retries; i++ {
		b.executeJob()
		if b.err == nil {
			// job was successful
			return
		}
		b.status = JobStatusRetrying
	}
	b.status = JobStatusFailed
}
