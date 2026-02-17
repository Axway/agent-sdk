package jobs

import (
	"time"
)

type intervalJobProps struct {
	interval time.Duration
	stopChan chan bool
}

type intervalJob struct {
	*baseJob
	intervalJobProps
}

// newIntervalJob - creates an interval run job
func newIntervalJob(newJob Job, interval time.Duration, name string, failJobChan chan string, opts ...jobOpt) (JobExecution, error) {
	thisJob := intervalJob{
		createBaseJob(newJob, failJobChan, name, JobTypeInterval),
		intervalJobProps{
			interval: interval,
			stopChan: make(chan bool, 1),
		},
	}

	for _, o := range opts {
		o(thisJob.baseJob)
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *intervalJob) handleExecution() {
	// Execute the job now and then start the interval period
	b.executeCronJob()
	if b.getError() != nil {
		b.setExecutionError()
		b.logger.Error(b.getError())
		b.stop() // stop the job on error
		b.incrementConsecutiveFails()
		return
	}
	b.resetConsecutiveFails()
}

// start - calls the Execute function from the Job definition
func (b *intervalJob) start() {
	b.startLog()
	b.waitForReady()

	// This could happen while rescheduling the job, pool tries to start
	// and one of the job fails which triggers stop setting the flag to not ready
	// Return in this case to allow pool to reschedule the job
	if !b.IsReady() {
		return
	}

	b.setIsStopped(false)
	b.SetStatus(JobStatusRunning)

	// Execute the job now and then start the interval period
	b.handleExecution()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		// Non-blocking channel read, if stopped then exit
		select {
		case <-b.stopChan:
			b.SetStatus(JobStatusStopped)
			return
		case <-ticker.C:
			b.handleExecution()
			ticker.Stop()
			ticker = time.NewTicker(b.interval)
		}
	}
}

// stop - write to the stop channel to stop the execution loop
func (b *intervalJob) stop() {
	if !b.isStopped.CompareAndSwap(false, true) {
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
