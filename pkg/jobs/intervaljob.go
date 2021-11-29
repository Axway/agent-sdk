package jobs

import (
	"time"

	"github.com/Axway/agent-sdk/pkg/util/log"
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
	if b.err != nil {
		b.setExecutionError()
		log.Error(b.err)
		b.SetStatus(JobStatusStopped)
		b.consecutiveFails++
	}
	b.consecutiveFails = 0
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
		log.Tracef("writing to %s stop channel", b.GetName())
		b.stopChan <- true
		log.Tracef("wrote to %s stop channel", b.GetName())
	} else {
		b.stopReadyChan <- nil
	}
	b.UnsetIsReady()
}
