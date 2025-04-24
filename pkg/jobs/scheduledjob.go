package jobs

import (
	"sync"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/Axway/agent-sdk/pkg/util/errors"
)

type scheduleJobProps struct {
	schedule string
	cronExp  *cronexpr.Expression
	stopChan chan bool
}

type scheduleJob struct {
	*baseJob
	scheduleJobProps
	cronLock *sync.Mutex
}

// newScheduledJob - creates a job that is ran at a specific time (@hourly,@daily,@weekly,min hour dow dom)
func newScheduledJob(newJob Job, schedule, name string, failJobChan chan string, opts ...jobOpt) (JobExecution, error) {
	exp, err := cronexpr.Parse(schedule)
	if err != nil {
		return nil, errors.Wrap(ErrRegisteringJob, err.Error()).FormatError("scheduled")
	}

	thisJob := scheduleJob{
		createBaseJob(newJob, failJobChan, name, JobTypeScheduled),
		scheduleJobProps{
			cronExp:  exp,
			schedule: schedule,
			stopChan: make(chan bool, 1),
		},
		&sync.Mutex{},
	}

	for _, o := range opts {
		o(thisJob.baseJob)
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *scheduleJob) getNextExecution() time.Duration {
	b.cronLock.Lock()
	defer b.cronLock.Unlock()
	nextTime := b.cronExp.Next(time.Now())
	return time.Until(nextTime)
}

// start - calls the Execute function from the Job definition
func (b *scheduleJob) start() {
	b.startLog()
	b.waitForReady()

	// This could happen while rescheduling the job, pool tries to start
	// and one of the job fails which triggers stop setting the flag to not ready
	// Return in this case to allow pool to reschedule the job
	if !b.IsReady() {
		return
	}
	b.setIsStopped(false)
	ticker := time.NewTicker(b.getNextExecution())
	defer ticker.Stop()
	b.SetStatus(JobStatusRunning)
	for {
		// Non-blocking channel read, if stopped then exit
		select {
		case <-b.stopChan:
			b.SetStatus(JobStatusStopped)
			return
		case <-ticker.C:
			b.executeCronJob()
			if b.getError() != nil {
				b.setExecutionError()
			}
			ticker.Stop()
			ticker = time.NewTicker(b.getNextExecution())
		}
	}
}

// stop - write to the stop channel to stop the execution loop
func (b *scheduleJob) stop() {
	if b.getIsStopped() {
		b.logger.Tracef("job has already been stopped")
		return
	}

	b.stopLog()
	if b.IsReady() {
		b.logger.Tracef("writing to %s stop channel", b.GetName())
		b.stopChan <- true
		b.logger.Tracef("wrote to %s stop channel", b.GetName())
		b.UnsetIsReady()
	} else {
		b.stopReadyIfWaiting(0)
	}
	b.setIsStopped(true)
}
