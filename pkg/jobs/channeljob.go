package jobs

import (
	"sync"
)

type channelJobProps struct {
	signalStop  chan interface{}
	stopChan    chan bool
	isStopped   bool
	stoppedLock *sync.Mutex
}

type channelJob struct {
	baseJob
	channelJobProps
}

//newDetachedChannelJob - creates a channel job, detached from other cron jobs
func newDetachedChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		createBaseJob(newJob, failJobChan, name, JobTypeDetachedChannel),
		channelJobProps{
			signalStop:  signalStop,
			stopChan:    make(chan bool),
			stoppedLock: &sync.Mutex{},
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

//newChannelJob - creates a channel run job
func newChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		createBaseJob(newJob, failJobChan, name, JobTypeChannel),
		channelJobProps{
			signalStop:  signalStop,
			stopChan:    make(chan bool),
			stoppedLock: &sync.Mutex{},
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

func (b *channelJob) handleExecution() {
	// Execute the job
	b.setError(b.job.Execute())
	if b.getError() != nil {
		b.setExecutionError()
		b.baseJob.logger.Error(b.err)
		b.stop() // stop the job on error
		b.consecutiveFails++
	}
	b.setConsecutiveFails(0)
}

//start - calls the Execute function from the Job definition
func (b *channelJob) start() {
	b.startLog()
	b.waitForReady()
	go b.handleExecution() // start a single execution in a go routine as it runs forever
	b.SetStatus(JobStatusRunning)
	b.setIsStopped(false)

	// Wait for a write on the stop channel
	<-b.stopChan
	b.signalStop <- nil // signal the execution to stop
	b.SetStatus(JobStatusStopped)
}

func (b *channelJob) getIsStopped() bool {
	b.stoppedLock.Lock()
	defer b.stoppedLock.Unlock()
	return b.isStopped
}

func (b *channelJob) setIsStopped(stopped bool) {
	b.stoppedLock.Lock()
	defer b.stoppedLock.Unlock()
	b.isStopped = stopped
}

//stop - write to the stop channel to stop the execution loop
func (b *channelJob) stop() {
	if b.getIsStopped() {
		b.baseJob.logger.Tracef("job has already been stopped")
		return
	}
	b.stopLog()
	if b.IsReady() {
		b.baseJob.logger.Tracef("writing to %s stop channel", b.GetName())
		b.stopChan <- true
		b.baseJob.logger.Tracef("wrote to %s stop channel", b.GetName())
		b.UnsetIsReady()
	} else {
		b.stopReadyIfWaiting(0)
	}
	b.setIsStopped(true)
}
