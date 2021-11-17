package jobs

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type channelJobProps struct {
	signalStop chan interface{}
	stopChan   chan bool
}

type channelJob struct {
	baseJob
	channelJobProps
}

//newChannelJob - creates a channel run job
func newChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		baseJob{
			id:       newUUID(),
			name:     name,
			job:      newJob,
			jobType:  JobTypeInterval,
			status:   JobStatusInitializing,
			failChan: failJobChan,
		},
		channelJobProps{
			signalStop: signalStop,
			stopChan:   make(chan bool),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *channelJob) handleExecution() {
	// Execute the job
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
func (b *channelJob) start() {
	b.startLog()
	b.waitForReady()

	for {
		// Non-blocking channel read, if stopped then exit
		select {
		case <-b.stopChan:
			b.SetStatus(JobStatusStopped)
			b.signalStop <- nil
			return
		default:
			b.handleExecution()
		}
	}
}

//stop - write to the stop channel to stop the execution loop
func (b *channelJob) stop() {
	b.stopLog()
	log.Tracef("writing to %s stop channel", b.GetName())
	b.stopChan <- true
	log.Tracef("wrote to %s stop channel", b.GetName())
}
