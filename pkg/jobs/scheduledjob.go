package jobs

import (
	"time"

	"github.com/gorhill/cronexpr"
)

type scheduleJobProps struct {
	schedule      string
	cronExp       *cronexpr.Expression
	nextExecution time.Duration
	sleepChan     chan bool
	stopChan      chan bool
}

type scheduleJob struct {
	baseJob
	scheduleJobProps
}

//newScheduledJob - creates a job that is ran at a specific time (@hourly,@daily,@weekly,min hour dow dom)
func newScheduledJob(newJob Job, schedule string, failJobChan chan string) (JobExecution, error) {
	exp, err := cronexpr.Parse(schedule)
	if err != nil {
		return nil, err
	}

	thisJob := scheduleJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		scheduleJobProps{
			cronExp:   exp,
			schedule:  schedule,
			stopChan:  make(chan bool),
			sleepChan: make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *scheduleJob) setNextExecution() {
	nextTime := b.cronExp.Next(time.Now())
	b.nextExecution = nextTime.Sub(time.Now())
	go b.sleep()
}

func (b *scheduleJob) sleep() {
	time.Sleep(b.nextExecution)
	b.sleepChan <- true
}

//start - calls the Execute function from the Job definition
func (b *scheduleJob) start() {
	b.waitForReady()

	b.status = JobStatusRunning
	for {
		// Set the amount of time until the next execution then wait for it or a stop signal
		b.setNextExecution()
		select {
		case _ = <-b.stopChan:
			b.status = JobStatusStopped
			return
		case _ = <-b.sleepChan:
			break
		}

		b.executeCronJob()
	}
}

//stop - write to the stop channel to stop the execution loop
func (b *scheduleJob) stop() {
	b.stopChan <- true
}
