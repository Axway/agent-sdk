package jobs

import (
	"sync"
	"time"
)

type baseJob struct {
	JobExecution
	id       string    // UUID generated for this job
	job      Job       // the job definition
	status   JobStatus // current job status
	err      error     // the error thrown
	failChan chan string
	jobLock  sync.Mutex
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newBaseJob(newJob Job, failJobChan chan string) (JobExecution, error) {
	thisJob := baseJob{
		id:       newUUID(),
		job:      newJob,
		status:   JobStatusInitializing,
		failChan: failJobChan,
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *baseJob) executeJob() {
	b.err = b.job.Execute()
	b.status = JobStatusFinished
	if b.err != nil {
		b.status = JobStatusFailed
	}
}

func (b *baseJob) executeCronJob() {
	// check status before execute
	b.GetStatusValue()

	// Lock the mutex for external syn with the job
	b.jobLock.Lock()
	defer b.jobLock.Unlock()

	b.err = b.job.Execute()
	if b.err != nil {
		b.failChan <- b.id
		b.status = JobStatusFailed
	}
}

//Lock - locks the job, execution can not take place until the Unlock func is called
func (b *baseJob) Lock() {
	b.jobLock.Lock()
}

//Unlock - unlocks the job, execution can now take place
func (b *baseJob) Unlock() {
	b.jobLock.Unlock()
}

//GetStatus - returns the string representation of the job status
func (b *baseJob) GetStatus() string {
	return jobStatusToString[b.status]
}

//GetStatusValue - returns the job status
func (b *baseJob) GetStatusValue() JobStatus {
	b.status = JobStatusRunning // reset to running before checking
	jobStatus := b.job.Status() // get the current status
	if jobStatus != nil {       // on error set the status to failed
		b.failChan <- b.id
		b.status = JobStatusFailed
	}
	return b.status
}

//GetID - returns the ID for this job
func (b *baseJob) GetID() string {
	return b.id
}

//GetJob - returns the Job interface
func (b *baseJob) GetJob() JobExecution {
	return b
}

//waitForReady - waits for the Ready func to return true
func (b *baseJob) waitForReady() {
	for !b.job.Ready() { // Wait for the job to be ready before starting
		time.Sleep(time.Millisecond)
	}
}

//start - waits for Ready to return true then calls the Execute function from the Job definition
func (b *baseJob) start() {
	b.waitForReady()

	b.status = JobStatusRunning
	b.executeJob()
}

//stop - noop in base
func (b *baseJob) stop() {
	return
}
