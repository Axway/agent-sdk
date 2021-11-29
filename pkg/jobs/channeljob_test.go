package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type channelJobImpl struct {
	Job
	name       string
	runTime    time.Duration
	ready      bool
	executions int
	status     error
	stopChan   chan interface{}
}

func (j *channelJobImpl) Execute() error {
	j.executions++
	for {
		select {
		case <-j.stopChan:
			return nil
		default:
			time.Sleep(j.runTime)
		}
	}
}

func (j *channelJobImpl) Status() error {
	return j.status
}

func (j *channelJobImpl) Ready() bool {
	return j.ready
}

func TestChannelJob(t *testing.T) {
	job := &channelJobImpl{
		name:     "ChannelJob",
		runTime:  5 * time.Millisecond,
		ready:    false,
		stopChan: make(chan interface{}),
	}

	jobID, _ := RegisterChannelJob(job, job.stopChan)
	globalPool.jobs[jobID].(*channelJob).backoff = newBackoffTimeout(time.Millisecond, time.Millisecond, 1)

	status := GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusInitializing], status)
	job.ready = true
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRunning], status)
	time.Sleep(50 * time.Millisecond) // Let the executions continue
	globalPool.cronJobs[jobID].stop()
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusStopped], status)
	assert.LessOrEqual(t, 1, job.executions)

	// restart the job
	go globalPool.cronJobs[jobID].start()
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRunning], status)
	UnregisterJob(jobID)
	time.Sleep(10 * time.Millisecond)
}
