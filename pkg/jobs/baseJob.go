package jobs

import (
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type baseJob struct {
	JobExecution
	id               string       // UUID generated for this job
	name             string       // Name of the job
	job              Job          // the job definition
	jobType          string       // type of job
	status           JobStatus    // current job status
	err              error        // the error thrown
	statusLock       sync.RWMutex // lock on preventing status write/read at the same time
	failChan         chan string  // channel to send signal to pool of failure
	jobLock          sync.Mutex   // lock used for signalling that the job is being executed
	consecutiveFails int
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newBaseJob(newJob Job, failJobChan chan string, name string) (JobExecution, error) {
	thisJob := baseJob{
		id:       newUUID(),
		name:     name,
		job:      newJob,
		jobType:  JobTypeSingleRun,
		status:   JobStatusInitializing,
		failChan: failJobChan,
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *baseJob) executeJob() {
	b.err = b.job.Execute()
	b.SetStatus(JobStatusFinished)
	if b.err != nil {
		b.SetStatus(JobStatusFailed)
	}
}

func (b *baseJob) executeCronJob() {
	// check status before execute
	b.updateStatus()

	// Lock the mutex for external syn with the job
	b.jobLock.Lock()
	defer b.jobLock.Unlock()

	b.err = b.job.Execute()
	if b.err != nil {
		if b.failChan != nil {
			b.failChan <- b.id
		}
		b.SetStatus(JobStatusFailed)
	}
}

//SetStatus - locks the job, execution can not take place until the Unlock func is called
func (b *baseJob) SetStatus(status JobStatus) {
	b.statusLock.Lock()
	defer b.statusLock.Unlock()
	b.status = status
}

//Lock - locks the job, execution can not take place until the Unlock func is called
func (b *baseJob) Lock() {
	b.jobLock.Lock()
}

//Unlock - unlocks the job, execution can now take place
func (b *baseJob) Unlock() {
	b.jobLock.Unlock()
}

func (b *baseJob) getConsecutiveFails() int {
	return b.consecutiveFails
}

//GetStatusValue - returns the job status
func (b *baseJob) updateStatus() JobStatus {
	newStatus := JobStatusRunning // reset to running before checking
	jobStatus := b.job.Status()   // get the current status
	if jobStatus != nil {         // on error set the status to failed
		b.failChan <- b.id
		log.Errorf("job with ID %s failed: %s", b.id, jobStatus.Error())
		newStatus = JobStatusFailed
	}
	b.statusLock.Lock()
	defer b.statusLock.Unlock()
	b.status = newStatus
	return b.status
}

//GetStatusValue - returns the job status
func (b *baseJob) GetStatus() JobStatus {
	b.statusLock.Lock()
	defer b.statusLock.Unlock()
	return b.status
}

//GetID - returns the ID for this job
func (b *baseJob) GetID() string {
	return b.id
}

//GetName - returns the name for this job, returns the ID if name is blank
func (b *baseJob) GetName() string {
	if b.name == "" {
		return b.id
	}
	return b.name
}

//GetJob - returns the Job interface
func (b *baseJob) GetJob() JobExecution {
	return b
}

//Ready - checks that the job is ready
func (b *baseJob) Ready() bool {
	return b.job.Ready()
}

//waitForReady - waits for the Ready func to return true
func (b *baseJob) waitForReady() {
	for !b.job.Ready() { // Wait for the job to be ready before starting
		time.Sleep(time.Millisecond)
	}
}

//start - waits for Ready to return true then calls the Execute function from the Job definition
func (b *baseJob) start() {
	b.startLog()
	b.waitForReady()

	b.SetStatus(JobStatusRunning)
	b.executeJob()
}

//stop - noop in base
func (b *baseJob) stop() {
	b.stopLog()
	return
}

func (b *baseJob) startLog() {
	if b.name != "" {
		log.Debugf("Starting %v (%v) job %v", b.jobType, b.name, b.id)
	} else {
		log.Debugf("Starting %v job %v", b.jobType, b.id)
	}
}

func (b *baseJob) stopLog() {
	if b.name != "" {
		log.Debugf("Stopping %v (%v) job %v", b.jobType, b.name, b.id)
	} else {
		log.Debugf("Stopping %v job %v", b.jobType, b.id)
	}
}

func (b *baseJob) setExecutionError() {
	b.err = errors.Wrap(ErrExecutingJob, b.err.Error()).FormatError(b.jobType, b.id)
}
