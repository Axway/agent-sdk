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
	backoffLock      *sync.RWMutex // lock on preventing backoff write/read at the same time
	failsLock        *sync.RWMutex // lock on preventing consecutiveFails write/read at the same time
	failChan         chan string   // channel to send signal to pool of failure
	errorLock        *sync.RWMutex // lock on preventing error write/read at the same time
	jobLock          sync.Mutex    // lock used for signalling that the job is being executed
	consecutiveFails int
	backoff          *backoff
	isReady          bool
	stopReadyChan    chan interface{}
	logger           log.FieldLogger
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newBaseJob(newJob Job, failJobChan chan string, name string) (JobExecution, error) {
	thisJob := createBaseJob(newJob, failJobChan, name, JobTypeSingleRun)

	go thisJob.start()
	return &thisJob, nil
}

//createBaseJob - creates a single run job and returns it
func createBaseJob(newJob Job, failJobChan chan string, name string, jobType string) baseJob {
	id := newUUID()
	logger := log.NewFieldLogger().
		WithPackage("sdk.jobs").
		WithComponent("baseJob").
		WithField("job-name", name).
		WithField("job-id", id)

	backoff := newBackoffTimeout(10*time.Millisecond, 10*time.Minute, 2)
	if jobType != JobTypeDetachedChannel && jobType != JobTypeDetachedInterval {
		backoff = nil
	}

	return baseJob{
		id:            id,
		name:          name,
		job:           newJob,
		jobType:       jobType,
		status:        JobStatusInitializing,
		failChan:      failJobChan,
		statusLock:    &sync.RWMutex{},
		isReadyLock:   &sync.RWMutex{},
		backoffLock:   &sync.RWMutex{},
		failsLock:     &sync.RWMutex{},
		errorLock:     &sync.RWMutex{},
		backoff:       backoff,
		isReady:       false,
		stopReadyChan: make(chan interface{}),
		logger:        logger,
	}
}

func (b *baseJob) executeJob() {
	b.setError(b.job.Execute())
	b.SetStatus(JobStatusFinished)
	if b.getError() != nil {
		b.SetStatus(JobStatusFailed)
	}
}

func (b *baseJob) callWithTimeout(execution func() error) error {
	var executionError error
	// execution time limit is set
	if executionTimeLimit > 0 {
		// start a go routine to execute the job
		executed := make(chan error)
		go func() {
			executed <- execution()
		}()

		// either the job finishes or a timeout is hit
		select {
		case err := <-executed:
			executionError = err
		case <-time.After(executionTimeLimit): // execute the job with a time limit
			executionError = fmt.Errorf("job %s (%s) timed out", b.name, b.id)
		}
	} else {
		executionError = execution()
	}

	return executionError
}

func (b *baseJob) executeCronJob() {
	// Lock the mutex for external syn with the job
	b.jobLock.Lock()
	defer b.jobLock.Unlock()

	b.setError(b.callWithTimeout(b.job.Execute))
	if b.getError() != nil {
		if b.failChan != nil {
			b.failChan <- b.id
		}
		b.SetStatus(JobStatusFailed)
	}
}

// getBackoff - get the job backoff
func (b *baseJob) getBackoff() *backoff {
	b.backoffLock.Lock()
	defer b.backoffLock.Unlock()
	return b.backoff
}

// setBackoff - set the job backoff
func (b *baseJob) setBackoff(backoff *backoff) {
	b.backoffLock.Lock()
	defer b.backoffLock.Unlock()
	b.backoff = backoff
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
	b.failsLock.Lock()
	defer b.failsLock.Unlock()
	return b.consecutiveFails
}

func (b *baseJob) setConsecutiveFails(fails int) {
	b.failsLock.Lock()
	defer b.failsLock.Unlock()
	b.consecutiveFails = fails
}

func (b *baseJob) getError() error {
	b.errorLock.Lock()
	defer b.errorLock.Unlock()
	return b.err
}

func (b *baseJob) setError(err error) {
	b.errorLock.Lock()
	defer b.errorLock.Unlock()
	b.err = err
}

//GetStatusValue - returns the job status
func (b *baseJob) updateStatus() JobStatus {
	b.statusLock.Lock()
	defer b.statusLock.Unlock()
	newStatus := JobStatusRunning // reset to running before checking
	jobStatus := b.callWithTimeout(b.job.Status)
	if jobStatus != nil { // on error set the status to failed
		b.failChan <- b.id
		b.logger.WithError(jobStatus).Error("job failed")

		newStatus = JobStatusFailed
	}

	b.status = newStatus
	return b.status
}

//GetStatus - returns the job status
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
	b.logger.Debugf("waiting for job to be ready: %s", b.GetName())
	for {
		select {
		case <-b.stopReadyChan:
			if b.getBackoff() != nil {
				b.getBackoff().reset()
			}

			b.UnsetIsReady()
			return
		default:
			if b.job.Ready() {
				if b.getBackoff() != nil {
					b.getBackoff().reset()
				}
				b.logger.Debug("job is ready")
				b.SetIsReady()
				return
			}
			if b.getBackoff() != nil {
				b.logger.Tracef("job is not ready, checking again in %v seconds", b.getBackoff().getCurrentTimeout())
				b.getBackoff().sleep()
				b.getBackoff().increaseTimeout()
			}
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
	b.logger.Debugf("Starting %v", b.jobType)
}

func (b *baseJob) stopLog() {
	b.logger.Debugf("Stopping %v ", b.jobType)
}

func (b *baseJob) setExecutionError() {
	b.setError(errors.Wrap(ErrExecutingJob, b.getError().Error()).FormatError(b.jobType, b.id))
}
