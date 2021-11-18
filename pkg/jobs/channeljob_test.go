package jobs

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type channelJobImpl struct {
	Job
	name        string
	runTime     time.Duration
	ready       bool
	executions  int
	failEvery   int
	status      error
	failTime    time.Duration
	wasFailed   bool
	wasRestored bool
	stopChan    chan interface{}
}

func (j *channelJobImpl) Execute() error {
	j.executions++
	for {
		select {
		case <-j.stopChan:
			return nil
		default:
			time.Sleep(j.runTime)
			if j.failEvery > 0 && j.executions%j.failEvery == 0 {
				j.status = fmt.Errorf("FAIL")
				j.wasFailed = true
			}
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
}
