package jobs

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

type channelJobProps struct {
	signalStop chan interface{}
	stopChan   chan bool
	isStopped  bool
}

type channelJob struct {
	baseJob
	channelJobProps
}

//newChannelJob - creates a channel run job
func newChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		createBaseJob(newJob, failJobChan, name, JobTypeChannel),
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
	b.err = b.job.Execute()
	if b.err != nil {
		b.setExecutionError()
		log.Error(b.err)
		b.stop() // stop the job on error
		b.consecutiveFails++
	}
	b.consecutiveFails = 0
}

//start - calls the Execute function from the Job definition
func (b *channelJob) start() {
	b.startLog()
	b.waitForReady()
	go b.handleExecution() // start a single execution in a go routine as it runs forever
	b.SetStatus(JobStatusRunning)
	b.isStopped = false

	// Wait for a write on the stop channel
	<-b.stopChan
	b.signalStop <- nil // signal the execution to stop
	b.SetStatus(JobStatusStopped)
}

//stop - write to the stop channel to stop the execution loop
func (b *channelJob) stop() {
	if b.isStopped {
		log.Tracef("job has already been stopped")
		return
	}
	b.stopLog()
	if b.IsReady() {
		log.Tracef("writing to %s stop channel", b.GetName())
		b.stopChan <- true
		log.Tracef("wrote to %s stop channel", b.GetName())
	} else {
		b.stopReadyChan <- nil
	}
	b.isStopped = true
	b.UnsetIsReady()
}
