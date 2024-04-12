package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Pool - represents a pool of jobs that are related in such a way that when one is not running none of them should be
type Pool struct {
	jobs                    map[string]JobExecution // All jobs that are in this pool
	cronJobs                map[string]JobExecution // Jobs that run continuously, not just ran once
	detachedCronJobs        map[string]JobExecution // Jobs that run continuously, not just ran once, detached from all others
	poolStatus              PoolStatus              // Holds the current status of the pool of jobs
	failedJob               string                  // Holds the ID of the job that is the reason for a non-running status
	jobsMapLock             sync.Mutex
	cronJobsMapLock         sync.Mutex
	detachedCronJobsMapLock sync.Mutex
	poolStatusLock          sync.Mutex
	backoffLock             sync.Mutex
	failedJobLock           sync.Mutex
	failJobChan             chan string
	stopJobsChan            chan bool
	backoff                 *backoff
	logger                  log.FieldLogger
	startStopLock           sync.Mutex
	isStartStopping         bool
	isStartStopLock         sync.Mutex
}

func newPool() *Pool {
	logger := log.NewFieldLogger().
		WithComponent("Pool").
		WithPackage("sdk.jobs")

	newPool := Pool{
		jobs:             make(map[string]JobExecution),
		cronJobs:         make(map[string]JobExecution),
		detachedCronJobs: make(map[string]JobExecution),
		failedJob:        "",
		startStopLock:    sync.Mutex{},
		isStartStopLock:  sync.Mutex{},
		failJobChan:      make(chan string, 1),
		stopJobsChan:     make(chan bool, 1),
		backoff:          newBackoffTimeout(defaultRetryInterval, 10*time.Minute, 2),
		logger:           logger,
	}
	newPool.SetStatus(PoolStatusInitializing)

	// start routine to check all job status funcs and catch any failures
	go newPool.jobChecker()
	// start the pool watcher
	go newPool.watchJobs()

	return &newPool
}

// getBackoff - get the job backoff
func (p *Pool) getBackoff() *backoff {
	p.backoffLock.Lock()
	defer p.backoffLock.Unlock()
	return p.backoff
}

// setBackoff - set the job backoff
func (p *Pool) setBackoff(backoff *backoff) {
	p.backoffLock.Lock()
	defer p.backoffLock.Unlock()
	p.backoff = backoff
}

// recordJob - Adds a job to the jobs map
func (p *Pool) recordJob(job JobExecution) string {
	p.jobsMapLock.Lock()
	defer p.jobsMapLock.Unlock()
	p.logger.
		WithField("job-id", job.GetID()).
		WithField("job-name", job.GetName()).
		Trace("registered job")
	p.jobs[job.GetID()] = job
	return job.GetID()
}

func (p *Pool) setCronJob(job JobExecution) {
	p.cronJobsMapLock.Lock()
	defer p.cronJobsMapLock.Unlock()
	p.cronJobs[job.GetID()] = job
}

func (p *Pool) getCronJob(jobID string) (JobExecution, bool) {
	p.cronJobsMapLock.Lock()
	defer p.cronJobsMapLock.Unlock()
	value, exists := p.cronJobs[jobID]
	return value, exists
}

func (p *Pool) getCronJobs() map[string]JobExecution {
	p.cronJobsMapLock.Lock()
	defer p.cronJobsMapLock.Unlock()

	// Create the target map
	newMap := make(map[string]JobExecution)

	// Copy from the original map to the target map to avoid race conditions
	for key, value := range p.cronJobs {
		newMap[key] = value
	}
	return newMap
}

func (p *Pool) setDetachedCronJob(job JobExecution) {
	p.detachedCronJobsMapLock.Lock()
	defer p.detachedCronJobsMapLock.Unlock()
	p.detachedCronJobs[job.GetID()] = job
}

func (p *Pool) getDetachedCronJob(jobID string) (JobExecution, bool) {
	p.detachedCronJobsMapLock.Lock()
	defer p.detachedCronJobsMapLock.Unlock()
	value, exists := p.detachedCronJobs[jobID]
	return value, exists
}

// recordCronJob - Adds a job to the cron jobs map
func (p *Pool) recordCronJob(job JobExecution) string {
	p.setCronJob(job)
	p.logger.Tracef("added new cron job, now running %v cron jobs", len(p.cronJobs))
	return p.recordJob(job)
}

// recordDetachedCronJob - Adds a job to the detached cron jobs map
func (p *Pool) recordDetachedCronJob(job JobExecution) string {
	p.setDetachedCronJob(job)
	p.logger.Tracef("added new cron job, now running %v detached cron jobs", len(p.detachedCronJobs))
	return p.recordJob(job)
}

// recordJob - Removes the specified job from jobs map
func (p *Pool) removeJob(jobID string) {
	p.jobsMapLock.Lock()
	job, ok := p.jobs[jobID]
	if ok {
		job.stop()
		delete(p.jobs, jobID)
	}
	p.jobsMapLock.Unlock()

	// remove from cron jobs, if present
	_, found := p.getCronJob(jobID)
	p.cronJobsMapLock.Lock()
	if found {
		delete(p.cronJobs, jobID)
	}
	p.cronJobsMapLock.Unlock()

	// remove from detached cron jobs, if present
	_, found = p.getDetachedCronJob(jobID)
	p.detachedCronJobsMapLock.Lock()
	if found {
		delete(p.detachedCronJobs, jobID)
	}
	p.detachedCronJobsMapLock.Unlock()
}

// RegisterSingleRunJob - Runs a single run job
func (p *Pool) RegisterSingleRunJob(newJob Job) (string, error) {
	return p.RegisterSingleRunJobWithName(newJob, JobTypeSingleRun)
}

// RegisterSingleRunJobWithName - Runs a single run job
func (p *Pool) RegisterSingleRunJobWithName(newJob Job, name string) (string, error) {
	job, err := newBaseJob(newJob, p.failJobChan, name)
	if err != nil {
		return "", err
	}
	return p.recordJob(job), nil
}

// RegisterIntervalJob - Runs a job with a specific interval between each run
func (p *Pool) RegisterIntervalJob(newJob Job, interval time.Duration, opts ...jobOpt) (string, error) {
	return p.RegisterIntervalJobWithName(newJob, interval, JobTypeInterval, opts...)
}

// RegisterIntervalJobWithName - Runs a job with a specific interval between each run
func (p *Pool) RegisterIntervalJobWithName(newJob Job, interval time.Duration, name string, opts ...jobOpt) (string, error) {
	job, err := newIntervalJob(newJob, interval, name, p.failJobChan, opts...)
	if err != nil {
		return "", err
	}
	return p.recordCronJob(job), nil
}

// RegisterChannelJob - Runs a job with a specific interval between each run
func (p *Pool) RegisterChannelJob(newJob Job, stopChan chan interface{}) (string, error) {
	return p.RegisterChannelJobWithName(newJob, stopChan, JobTypeChannel)
}

// RegisterChannelJobWithName - Runs a job with a specific interval between each run
func (p *Pool) RegisterChannelJobWithName(newJob Job, stopChan chan interface{}, name string) (string, error) {
	job, err := newChannelJob(newJob, stopChan, name, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordCronJob(job), nil
}

// RegisterDetachedChannelJob - Runs a job with a stop channel, detached from other jobs
func (p *Pool) RegisterDetachedChannelJob(newJob Job, stopChan chan interface{}) (string, error) {
	return p.RegisterDetachedChannelJobWithName(newJob, stopChan, JobTypeDetachedChannel)
}

// RegisterDetachedChannelJobWithName - Runs a named job with a stop channel, detached from other jobs
func (p *Pool) RegisterDetachedChannelJobWithName(newJob Job, stopChan chan interface{}, name string) (string, error) {
	job, err := newDetachedChannelJob(newJob, stopChan, name, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordDetachedCronJob(job), nil
}

// RegisterDetachedIntervalJob - Runs a job with a specific interval between each run, detached from other jobs
func (p *Pool) RegisterDetachedIntervalJob(newJob Job, interval time.Duration, opts ...jobOpt) (string, error) {
	return p.RegisterDetachedIntervalJobWithName(newJob, interval, JobTypeDetachedInterval, opts...)
}

// RegisterDetachedIntervalJobWithName - Runs a job with a specific interval between each run, detached from other jobs
func (p *Pool) RegisterDetachedIntervalJobWithName(newJob Job, interval time.Duration, name string, opts ...jobOpt) (string, error) {
	job, err := newDetachedIntervalJob(newJob, interval, name, opts...)
	if err != nil {
		return "", err
	}
	return p.recordDetachedCronJob(job), nil
}

// RegisterScheduledJob - Runs a job on a specific schedule
func (p *Pool) RegisterScheduledJob(newJob Job, schedule string, opts ...jobOpt) (string, error) {
	return p.RegisterScheduledJobWithName(newJob, schedule, JobTypeScheduled, opts...)
}

// RegisterScheduledJobWithName - Runs a job on a specific schedule
func (p *Pool) RegisterScheduledJobWithName(newJob Job, schedule, name string, opts ...jobOpt) (string, error) {
	job, err := newScheduledJob(newJob, schedule, name, p.failJobChan, opts...)
	if err != nil {
		return "", err
	}
	return p.recordCronJob(job), nil
}

// RegisterRetryJob - Runs a job with a limited number of retries
func (p *Pool) RegisterRetryJob(newJob Job, retries int) (string, error) {
	return p.RegisterRetryJobWithName(newJob, retries, JobTypeRetry)
}

// RegisterRetryJobWithName  - Runs a job with a limited number of retries
func (p *Pool) RegisterRetryJobWithName(newJob Job, retries int, name string) (string, error) {
	job, err := newRetryJob(newJob, retries, name, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordJob(job), nil
}

// UnregisterJob - Removes the specified job
func (p *Pool) UnregisterJob(jobID string) {
	p.removeJob(jobID)
}

// GetJob - Returns the Job based on the id
func (p *Pool) GetJob(id string) JobExecution {
	return p.jobs[id].GetJob()
}

// JobLock - Locks the job, returns when the lock is granted
func (p *Pool) JobLock(id string) {
	p.jobs[id].Lock()
}

// JobUnlock - Unlocks the job
func (p *Pool) JobUnlock(id string) {
	p.jobs[id].Unlock()
}

func (p *Pool) getFailedJob() string {
	p.failedJobLock.Lock()
	defer p.failedJobLock.Unlock()
	return p.failedJob
}

func (p *Pool) setFailedJob(job string) {
	p.failedJobLock.Lock()
	defer p.failedJobLock.Unlock()
	p.failedJob = job
}

// GetJobStatus - Returns the Status of the Job based on the id
func (p *Pool) GetJobStatus(id string) string {
	return p.jobs[id].GetStatus().String()
}

// GetStatus - returns the status of the pool of jobs
func (p *Pool) GetStatus() string {
	p.poolStatusLock.Lock()
	defer p.poolStatusLock.Unlock()
	return p.poolStatus.String()
}

// SetStatus - Sets the status of the pool of jobs
func (p *Pool) SetStatus(status PoolStatus) {
	p.poolStatusLock.Lock()
	defer p.poolStatusLock.Unlock()
	p.poolStatus = status
}

// waits with timeout for the specified status in all cron jobs
func (p *Pool) waitStartStop(jobStatus JobStatus) bool {
	ctx, cancel := context.WithTimeout(context.Background(), getStatusCheckInterval())
	defer cancel()

	done := make(chan bool)
	go func() {
		for {
			running := true
			for _, job := range p.getCronJobs() {
				if job.GetStatus() != jobStatus {
					running = false
				}
			}
			if running {
				done <- true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	select {
	case b := <-done:
		return b
	case <-ctx.Done():
		return false
	}
}

func (p *Pool) setIsStartStop(isStartStop bool) {
	p.isStartStopLock.Lock()
	defer p.isStartStopLock.Unlock()
	p.isStartStopping = isStartStop
}

func (p *Pool) getIsStartStop() bool {
	p.isStartStopLock.Lock()
	defer p.isStartStopLock.Unlock()
	return p.isStartStopping
}

// startAll - starts all jobs defined in the cronJobs map, used by watchJobs
//
//	          other jobs are single run and never restarted
//						 returns true when successful, false when not
func (p *Pool) startAll() bool {
	p.stopAll()

	// Check that all are ready before starting
	p.logger.Debug("Checking for all cron jobs to be ready")
	for _, job := range p.getCronJobs() {
		if !job.Ready() {
			p.logger.WithField("job-id", job.GetID()).Debugf("job is not ready")
			return false
		}
	}
	p.logger.Debug("Starting all cron jobs")
	for _, job := range p.getCronJobs() {
		go job.start()
	}

	if p.waitStartStop(JobStatusRunning) {
		p.SetStatus(PoolStatusRunning)
	}

	return true
}

// stopAll - stops all jobs defined in the cronJobs map, used by watchJobs
//
//	other jobs are single run and should not need stopped
func (p *Pool) stopAll() {
	p.logger.Debug("Stopping all cron jobs")

	// Must do the map copy so that the loop can run without a race condition.
	// Can NOT do a defer on this unlock, or will get stuck
	mapCopy := make(map[string]JobExecution)
	for key, value := range p.getCronJobs() {
		mapCopy[key] = value
	}
	for _, job := range mapCopy {
		p.logger.WithField("job-name", job.GetName()).Trace("stopping job")
		job.stop()
		p.logger.WithField("job-name", job.GetName()).Tracef("finished stopping job")
	}

	if p.waitStartStop(JobStatusStopped) {
		p.SetStatus(PoolStatusStopped)
	}
}

// jobChecker - regularly checks the status of cron jobs, stopping jobs if error returned
func (p *Pool) jobChecker() {
	ticker := time.NewTicker(getStatusCheckInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go func() {
				failedJob := ""
				for _, job := range p.getCronJobs() {
					job.updateStatus()
					if job.GetStatus() != JobStatusRunning {
						failedJob = job.GetID()
						break
					}
				}

				if !p.getIsStartStop() {
					if failedJob != "" {
						p.failJobChan <- failedJob
					} else {
						p.SetStatus(PoolStatusRunning)
					}
				}
			}()
		case failedJob := <-p.failJobChan:
			p.setFailedJob(failedJob) // this is the job for the current fail loop
			p.stopJobsChan <- true
			p.SetStatus(PoolStatusStopped)
		}
	}
}

func (p *Pool) stopPool() {
	p.startStopLock.Lock()
	defer p.startStopLock.Unlock()

	p.setIsStartStop(true)
	defer p.setIsStartStop(false)
	p.stopAll()
}

func (p *Pool) startPool() {
	p.startStopLock.Lock()
	defer p.startStopLock.Unlock()

	if p.GetStatus() == PoolStatusStopped.String() {
		p.setIsStartStop(true)
		defer p.setIsStartStop(false)
		// attempt to restart all jobs
		if p.startAll() {
			p.getBackoff().reset()
		} else {
			p.getBackoff().increaseTimeout()
		}
		p.setFailedJob("")
	}
}

// watchJobs - the main loop of a pool of jobs, constantly checks for status of jobs and acts accordingly
func (p *Pool) watchJobs() {
	p.SetStatus(PoolStatusRunning)
	ticker := time.NewTicker(p.getBackoff().getCurrentTimeout())
	defer ticker.Stop()
	for {
		select {
		case <-p.stopJobsChan:
			if job, found := p.getCronJob(p.getFailedJob()); found {
				p.logger.
					WithField("job-name", job.GetName()).
					WithField("failed-job", p.getFailedJob()).
					Debug("Job failed, stop all jobs")
			}
			p.stopPool()
		case <-ticker.C:
			p.startPool()
			ticker = time.NewTicker(p.getBackoff().getCurrentTimeout())
			p.logger.
				WithField("interval", p.getBackoff().getCurrentTimeout()).
				Trace("setting next job restart backoff interval")
		}
	}
}
