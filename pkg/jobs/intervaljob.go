package jobs

import (
	"time"
)

type intervalJobProps struct {
	interval time.Duration
	stopChan chan bool
}

type intervalJob struct {
	baseJob
	intervalJobProps
}

//newIntervalJob - creates an interval run job
func newIntervalJob(newJob Job, interval time.Duration, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := intervalJob{
		createBaseJob(newJob, failJobChan, name, JobTypeInterval),
		intervalJobProps{
			interval: interval,
			stopChan: make(chan bool),
		},
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
		b.SetStatus(JobStatusStopped)
		b.setConsecutiveFails(b.getConsecutiveFails() + 1)
	}
	b.setConsecutiveFails(0)
}

//start - calls the Execute function from the Job definition
func (b *intervalJob) start() {
	b.startLog()
	b.waitForReady()

	// Execute the job now and then start the interval period
	b.handleExecution()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	b.SetStatus(JobStatusRunning)
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

//stop - write to the stop channel to stop the execution loop
func (b *intervalJob) stop() {
	b.stopLog()
	if b.IsReady() {
		b.baseJob.logger.Tracef("writing to %s stop channel", b.GetName())
		b.stopChan <- true
		b.baseJob.logger.Tracef("wrote to %s stop channel", b.GetName())
		b.UnsetIsReady()
	} else {
		b.stopReadyChan <- nil
	}
}
