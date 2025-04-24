package jobs

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/errors"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type baseJob struct {
	JobExecution
	logger           log.FieldLogger
	id               string                // UUID generated for this job
	name             string                // Name of the job
	job              Job                   // the job definition
	jobType          string                // type of job
	status           atomic.Value          // current job status (atomic.Value for thread-safe access)
	err              atomic.Pointer[error] // atomic pointer to store the error (thread-safe)
	isReady          atomic.Bool           // atomic boolean to track readiness
	isReadyWait      atomic.Bool           // atomic boolean to track if the job is waiting for readiness
	backoff          atomic.Value          // atomic.Value to store the backoff (thread-safe)
	isStopped        atomic.Bool           // atomic boolean to track if the job is stopped
	failChan         chan string           // channel to send signal to pool of failure
	jobLock          sync.Mutex            // lock used for signalling that the job is being executed
	consecutiveFails atomic.Int32          // atomic counter for consecutive failures
	stopReadyChan    chan int
	timeout          time.Duration
}

type jobOpt func(*baseJob)

func WithJobTimeout(timeout time.Duration) jobOpt {
	return func(b *baseJob) {
		b.timeout = timeout
	}
}

// newBaseJob - creates a single run job and sets up the structure for different job types
func newBaseJob(newJob Job, failJobChan chan string, name string) (JobExecution, error) {
	thisJob := createBaseJob(newJob, failJobChan, name, JobTypeSingleRun)

	go thisJob.start()
	return thisJob, nil
}

// createBaseJob - creates a single run job and returns it
func createBaseJob(newJob Job, failJobChan chan string, name string, jobType string) *baseJob {
	id := newUUID()
	logger := log.NewFieldLogger().
		WithPackage("sdk.jobs").
		WithComponent("baseJob").
		WithField("jobName", name).
		WithField("jobId", id).
		WithField("jobType", jobType)

	backoff := newBackoffTimeout(10*time.Millisecond, 10*time.Minute, 2)

	job := &baseJob{
		id:            id,
		name:          name,
		job:           newJob,
		jobType:       jobType,
		failChan:      failJobChan,
		stopReadyChan: make(chan int, 1),
		logger:        logger,
	}

	// Initialize the status with JobStatusInitializing
	job.status.Store(JobStatusInitializing)

	// Initialize isReady, isReadyWait, and isStopped to false
	job.isReady.Store(false)
	job.isReadyWait.Store(false)
	job.isStopped.Store(false)

	// Initialize the error to nil
	job.err.Store(nil)

	// Initialize the backoff
	job.backoff.Store(backoff)

	return job
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
	timeLimit := executionTimeLimit
	if b.timeout > 0 {
		timeLimit = b.timeout
	}
	if timeLimit > 0 {
		// start a go routine to execute the job
		executed := make(chan error)
		go func() {
			executed <- execution()
		}()

		// either the job finishes or a timeout is hit
		select {
		case err := <-executed:
			executionError = err
		case <-time.After(timeLimit): // execute the job with a time limit
			executionError = fmt.Errorf("job %s (%s) timed out", b.name, b.id)
		}
	} else {
		executionError = execution()
	}

	return executionError
}

func (b *baseJob) executeCronJob() {
	// Lock the mutex for external synchronization with the job
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

// getBackoff - retrieves the job backoff atomically
func (b *baseJob) getBackoff() *backoff {
	backoffAny := b.backoff.Load()
	if backoffAny != nil {
		return backoffAny.(*backoff)
	}
	return nil
}

// setBackoff - sets the job backoff atomically
func (b *baseJob) setBackoff(backoff *backoff) {
	b.backoff.Store(backoff)
}

// SetStatus - sets the job status atomically
func (b *baseJob) SetStatus(status JobStatus) {
	b.status.Store(status)
}

// setReadyWait - set flag to indicate the job is waiting for ready atomically
func (b *baseJob) setReadyWait(waitReady bool) {
	b.isReadyWait.Store(waitReady)
}

// isWaitingForReady - return true if job is waiting for ready atomically
func (b *baseJob) isWaitingForReady() bool {
	return b.isReadyWait.Load()
}

// SetIsReady - sets the job as ready atomically
func (b *baseJob) SetIsReady() {
	b.isReady.Store(true)
}

// UnsetIsReady - sets the job as not ready atomically
func (b *baseJob) UnsetIsReady() {
	b.isReady.Store(false)
}

// IsReady - checks if the job is ready atomically
func (b *baseJob) IsReady() bool {
	return b.isReady.Load()
}

// getIsStopped - checks if the job is stopped atomically
func (b *baseJob) getIsStopped() bool {
	return b.isStopped.Load()
}

// setIsStopped - sets the job as stopped atomically
func (b *baseJob) setIsStopped(stopped bool) {
	b.isStopped.Store(stopped)
}

// Lock - locks the job, execution can not take place until the Unlock func is called
func (b *baseJob) Lock() {
	b.jobLock.Lock()
}

// Unlock - unlocks the job, execution can now take place
func (b *baseJob) Unlock() {
	b.jobLock.Unlock()
}

// incrementConsecutiveFails - increments the value of consecutiveFails
func (b *baseJob) incrementConsecutiveFails() {
	b.consecutiveFails.Add(1)
}

// resetConsecutiveFails - resets the value of consecutiveFails to zero
func (b *baseJob) resetConsecutiveFails() {
	b.consecutiveFails.Store(0)
}

// getError - retrieves the job error atomically
func (b *baseJob) getError() error {
	errPtr := b.err.Load()
	if errPtr != nil {
		return *errPtr
	}
	return nil
}

// setError - sets the job error atomically
func (b *baseJob) setError(err error) {
	if err == nil {
		b.err.Store(nil)
	} else {
		b.err.Store(&err)
	}
}

// GetStatusValue - returns the job status
func (b *baseJob) updateStatus() JobStatus {
	newStatus := b.GetStatus()
	jobStatus := b.callWithTimeout(b.job.Status)
	if jobStatus != nil { // on error set the status to failed
		b.logger.WithError(jobStatus).Error("job failed")
		newStatus = JobStatusFailed
	}

	b.SetStatus(newStatus)
	b.logger.Tracef("current job status %s", jobStatusToString[newStatus])
	return newStatus
}

// GetStatus - retrieves the job status atomically
func (b *baseJob) GetStatus() JobStatus {
	return b.status.Load().(JobStatus)
}

// GetID - returns the ID for this job
func (b *baseJob) GetID() string {
	return b.id
}

// GetName - returns the name for this job, returns the ID if name is blank
func (b *baseJob) GetName() string {
	if b.name == "" {
		return b.id
	}
	return b.name
}

// GetJob - returns the Job interface
func (b *baseJob) GetJob() JobExecution {
	return b
}

// Ready - checks that the job is ready
func (b *baseJob) Ready() bool {
	return b.job.Ready()
}

// waitForReady - waits for the Ready func to return true
func (b *baseJob) waitForReady() {
	b.logger.Debugf("waiting for job to be ready: %s", b.GetName())
	b.setReadyWait(true)
	defer b.setReadyWait(false)

	for {
		select {
		case ready := <-b.stopReadyChan:
			if b.getBackoff() != nil {
				b.getBackoff().reset()
			}
			if ready == 1 {
				b.SetIsReady()
			} else {
				b.UnsetIsReady()
			}
			return
		default:
			if b.job.Ready() {
				b.logger.Debug("job is ready")
				b.stopReadyIfWaiting(1)
			} else {
				if b.getBackoff() != nil {
					b.logger.Tracef("job is not ready, checking again in %v seconds", b.getBackoff().getCurrentTimeout())
					b.getBackoff().sleep()
					b.getBackoff().increaseTimeout()
				}
			}
		}
	}
}

func (b *baseJob) stopReadyIfWaiting(ready int) {
	if b.isWaitingForReady() {
		b.stopReadyChan <- ready
	}
}

// start - waits for Ready to return true then calls the Execute function from the Job definition
func (b *baseJob) start() {
	b.startLog()
	b.waitForReady()

	b.SetStatus(JobStatusRunning)
	b.executeJob()
}

// stop - noop in base
func (b *baseJob) stop() {
	b.stopLog()
}

func (b *baseJob) startLog() {
	b.logger.Debug("Starting")
}

func (b *baseJob) stopLog() {
	b.logger.Debug("Stopping")
}

func (b *baseJob) setExecutionError() {
	b.setError(errors.Wrap(ErrExecutingJob, b.getError().Error()).FormatError(b.jobType, b.id))
}
