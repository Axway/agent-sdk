package jobs

import (
	"time"
)

type intervalJobProps struct {
	interval  time.Duration
	sleepChan chan bool
	stopChan  chan bool
}

type intervalJob struct {
	baseJob
	intervalJobProps
}

//newBaseJob - creates a single run job and sets up the structure for different job types
func newIntervalJob(newJob Job, interval time.Duration, failJobChan chan string) (JobExecution, error) {
	thisJob := intervalJob{
		baseJob{
			id:       newUUID(),
			job:      newJob,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		intervalJobProps{
			interval:  interval,
			sleepChan: make(chan bool),
			stopChan:  make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *intervalJob) sleep() {
	time.Sleep(b.interval)
	b.sleepChan <- true
}

//start - calls the Execute function from the Job definition
func (b *intervalJob) start() {
	b.waitForReady()

	b.status = JobStatusRunning
	for {
		// Non-blocking channel read, if stopped then exit
		select {
		case _ = <-b.stopChan:
			b.status = JobStatusStopped
			return
		default:
			b.executeCronJob()
		}

		go b.sleep()
		//Non-blocking sleep, incase a stopped is pushed before the interval is up
		select {
		case _ = <-b.stopChan:
			b.status = JobStatusStopped
			return
		case _ = <-b.sleepChan:
			break
		}
	}
}

//stop - write to the stop channel to stop the execution loop
func (b *intervalJob) stop() {
	b.stopChan <- true
}
