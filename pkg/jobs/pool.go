package jobs

import (
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
)

//Pool - represents a pool of jobs that are related in such a way that when one is not running none of them should be
type Pool struct {
	jobs            map[string]JobExecution // All jobs that are in this pool
	cronJobs        map[string]JobExecution // Jobs that run continuously, not just ran once
	poolStatus      PoolStatus              // Holds the current status of the pool of jobs
	failedJob       string                  // Holds the ID of the job that is the reason for a non-running status
	jobsMapLock     sync.Mutex
	cronJobsMapLock sync.Mutex
	failJobChan     chan string
	stopJobsChan    chan bool
}

func newPool() *Pool {
	newPool := Pool{
		jobs:         make(map[string]JobExecution),
		cronJobs:     make(map[string]JobExecution),
		poolStatus:   PoolStatusInitializing,
		failedJob:    "",
		failJobChan:  make(chan string),
		stopJobsChan: make(chan bool),
	}

	// start routine that catches all failures whenever written and acts on first
	go newPool.catchFails()
	// start the pool watcher
	go newPool.watchJobs()

	return &newPool
}

//recordJob - Adds a job to the jobs map
func (p *Pool) recordJob(job JobExecution) string {
	p.jobsMapLock.Lock()
	defer p.jobsMapLock.Unlock()
	p.jobs[job.GetID()] = job
	return job.GetID()
}

//recordCronJob - Adds a job to the cron jobs map
func (p *Pool) recordCronJob(job JobExecution) string {
	p.cronJobsMapLock.Lock()
	defer p.cronJobsMapLock.Unlock()
	p.cronJobs[job.GetID()] = job
	return p.recordJob(job)
}

//RegisterSingleRunJob - Runs a single run job
func (p *Pool) RegisterSingleRunJob(newJob Job) (string, error) {
	job, err := newBaseJob(newJob, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordJob(job), nil
}

//RegisterIntervalJob - Runs a job with a specific interval between each run
func (p *Pool) RegisterIntervalJob(newJob Job, interval time.Duration) (string, error) {
	job, err := newIntervalJob(newJob, interval, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordCronJob(job), nil
}

//RegisterScheduledJob - Runs a job on a specific schedule
func (p *Pool) RegisterScheduledJob(newJob Job, schedule string) (string, error) {
	job, err := newScheduledJob(newJob, schedule, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordCronJob(job), nil
}

//RegisterRetryJob - Runs a job with a limited number of retries
func (p *Pool) RegisterRetryJob(newJob Job, retries int) (string, error) {
	job, err := newRetryJob(newJob, retries, p.failJobChan)
	if err != nil {
		return "", err
	}
	return p.recordJob(job), nil
}

//GetJob - Returns the Job based on the id
func (p *Pool) GetJob(id string) JobExecution {
	return p.jobs[id].GetJob()
}

//JobLock - Locks the job, returns when the lock is granted
func (p *Pool) JobLock(id string) {
	p.jobs[id].Lock()
}

//JobUnlock - Unlocks the job
func (p *Pool) JobUnlock(id string) {
	p.jobs[id].Unlock()
}

//GetJobStatus - Returns the Status of the Job based on the id
func (p *Pool) GetJobStatus(id string) string {
	return p.jobs[id].GetStatus().String()
}

//GetStatus - returns the status of the pool of jobs
func (p *Pool) GetStatus() string {
	return p.poolStatus.String()
}

//startAll - starts all jobs defined in the cronJobs map, used by watchJobs
//           other jobs are single run and never restarted
func (p *Pool) startAll() {
	// Check that all are ready before starting
	log.Debug("Checking for all cron jobs to be ready")
	for _, job := range p.cronJobs {
		if !job.Ready() {
			return
		}
	}
	log.Debug("Starting all cron jobs")
	p.poolStatus = PoolStatusRunning
	for _, job := range p.cronJobs {
		go job.start()
	}
}

//stopAll - stops all jobs defined in the cronJobs map, used by watchJobs
//           other jobs are single run and should not need stopped
func (p *Pool) stopAll() {
	log.Debug("Stopping all cron jobs")
	p.poolStatus = PoolStatusStopped
	for _, job := range p.cronJobs {
		job.stop()
	}
}

//catchFails - catches all writes to the failJobChan and only sends one stop signal
func (p *Pool) catchFails() {
	for {
		// continue rading all failed jobs, only stop the first time
		failedJob := <-p.failJobChan
		if p.poolStatus == PoolStatusRunning {
			p.failedJob = failedJob // this is the job for the current fail loop
			p.stopJobsChan <- true
		}
	}
}

//watchJobs - the main loop of a pool of jobs, constantly checks for status of jobs and acts accordingly
func (p *Pool) watchJobs() {
	p.poolStatus = PoolStatusRunning
	for {
		if p.poolStatus == PoolStatusRunning {
			// The pool is running, wait for any signal that a job went down
			<-p.stopJobsChan
			log.Debugf("Job with id %v failed, stop all jobs", p.failedJob)
			p.stopAll()
		} else {
			log.Debug("Pool not running, start all jobs")
			// attempt to restart all jobs
			p.startAll()
			time.Sleep(time.Second)
		}
	}
}
