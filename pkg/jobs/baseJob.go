package jobs

import (
	"fmt"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type baseJob struct {
	JobExecution
	id               string        // UUID generated for this job
	name             string        // Name of the job
	job              Job           // the job definition
	jobType          string        // type of job
	status           JobStatus     // current job status
	err              error         // the error thrown
	statusLock       *sync.RWMutex // lock on preventing status write/read at the same time
	isReadyLock      *sync.RWMutex // lock on preventing isReady write/read at the same time
	failChan         chan string   // channel to send signal to pool of failure
	jobLock          sync.Mutex    // lock used for signalling that the job is being executed
	consecutiveFails int
	backoff          *backoff
	isReady          bool
	stopReadyChan    chan interface{}
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newBaseJob(newJob Job, failJobChan chan string, name string) (JobExecution, error) {
	thisJob := createBaseJob(newJob, failJobChan, name, JobTypeSingleRun)

	go thisJob.start()
	return &thisJob, nil
}

//createBaseJob - creates a single run job and returns it
func createBaseJob(newJob Job, failJobChan chan string, name string, jobType string) baseJob {
	return baseJob{
		id:            newUUID(),
		name:          name,
		job:           newJob,
		jobType:       jobType,
		status:        JobStatusInitializing,
		failChan:      failJobChan,
		statusLock:    &sync.RWMutex{},
		isReadyLock:   &sync.RWMutex{},
		backoff:       newBackoffTimeout(10*time.Millisecond, 10*time.Minute, 2),
		isReady:       false,
		stopReadyChan: make(chan interface{}),
	}
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

	// execution time limit is set
	if executionTimeLimit > 0 {
		// start a go routine to execute the job
		executed := make(chan error)
		go func() {
			executed <- b.job.Execute()
		}()

		// either the job finishes or a timeout is hit
		select {
		case err := <-executed:
			b.err = err
		case <-time.After(executionTimeLimit): // execute the job with a time limit
			b.err = fmt.Errorf("job %s (%s) timed out", b.name, b.id)
		}
	} else {
		b.err = b.job.Execute()
	}

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

//SetIsReady - set that the job is now ready
func (b *baseJob) SetIsReady() {
	b.isReadyLock.Lock()
	defer b.isReadyLock.Unlock()
	b.isReady = true
}

//UnsetIsReady - set that the job is now ready
func (b *baseJob) UnsetIsReady() {
	b.isReadyLock.Lock()
	defer b.isReadyLock.Unlock()
	b.isReady = false
}

//IsReady - set that the job is now ready
func (b *baseJob) IsReady() bool {
	b.isReadyLock.Lock()
	defer b.isReadyLock.Unlock()
	return b.isReady
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
		log.Errorf("job %s (%s) failed: %s", b.name, b.id, jobStatus.Error())
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
	log.Debugf("Waiting for %s (%s) to be ready", b.name, b.id)
	for {
		select {
		case <-b.stopReadyChan:
			b.backoff.reset()
			b.UnsetIsReady()
			return
		default:
			if b.job.Ready() {
				b.backoff.reset()
				log.Debugf("%s (%s) is ready", b.name, b.id)
				b.SetIsReady()
				return
			}
			log.Tracef("Job %s (%s) not ready, checking again in %v seconds", b.name, b.id, b.backoff.getCurrentTimeout())
			b.backoff.sleep()
			b.backoff.increaseTimeout()
		}
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
