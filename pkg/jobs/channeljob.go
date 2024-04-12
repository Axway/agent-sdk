package jobs

type channelJobProps struct {
	signalStop chan interface{}
	stopChan   chan bool
}

type channelJob struct {
	baseJob
	channelJobProps
}

// newDetachedChannelJob - creates a channel job, detached from other cron jobs
func newDetachedChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		createBaseJob(newJob, failJobChan, name, JobTypeDetachedChannel),
		channelJobProps{
			signalStop: signalStop,
			stopChan:   make(chan bool, 1),
		},
	}

	go thisJob.start()
	return &thisJob, nil
}

// newChannelJob - creates a channel run job
func newChannelJob(newJob Job, signalStop chan interface{}, name string, failJobChan chan string) (JobExecution, error) {
	thisJob := channelJob{
		createBaseJob(newJob, failJobChan, name, JobTypeChannel),
		channelJobProps{
			signalStop: signalStop,
			stopChan:   make(chan bool, 1),
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

// start - calls the Execute function from the Job definition
func (b *channelJob) start() {
	b.startLog()
	b.waitForReady()

	// This could happen while rescheduling the job, pool tries to start
	// and one of the job fails which triggers stop setting the flag to not ready
	// Return in this case to allow pool to reschedule the job
	if !b.IsReady() {
		return
	}

	go b.handleExecution() // start a single execution in a go routine as it runs forever
	b.SetStatus(JobStatusRunning)
	b.setIsStopped(false)

	// Wait for a write on the stop channel
	<-b.stopChan
	b.signalStop <- nil // signal the execution to stop
	b.SetStatus(JobStatusStopped)
}

// stop - write to the stop channel to stop the execution loop
func (b *channelJob) stop() {
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
