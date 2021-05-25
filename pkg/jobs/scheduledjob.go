package jobs

import (
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
	baseJob
	scheduleJobProps
}

//newScheduledJob - creates a job that is ran at a specific time (@hourly,@daily,@weekly,min hour dow dom)
func newScheduledJob(newJob Job, schedule string, failJobChan chan string) (JobExecution, error) {
	exp, err := cronexpr.Parse(schedule)
	if err != nil {
		return nil, errors.Wrap(ErrRegisteringJob, err.Error()).FormatError("scheduled")
	}

	thisJob := scheduleJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			jobType:  JobTypeScheduled,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		scheduleJobProps{
			cronExp:  exp,
			schedule: schedule,
			stopChan: make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *scheduleJob) getNextExecution() time.Duration {
	nextTime := b.cronExp.Next(time.Now())
	return nextTime.Sub(time.Now())
}

//start - calls the Execute function from the Job definition
func (b *scheduleJob) start() {
	b.startLog()
	b.waitForReady()

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
			if b.err != nil {
				b.setExecutionError()
				b.SetStatus(JobStatusStopped)
			}
			ticker.Stop()
			ticker = time.NewTicker(b.getNextExecution())
		}
	}
}

//stop - write to the stop channel to stop the execution loop
func (b *scheduleJob) stop() {
	b.stopLog()
	b.stopChan <- true
}
