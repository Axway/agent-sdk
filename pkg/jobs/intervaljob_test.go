package jobs

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type intervalJobImpl struct {
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
}

func (j *intervalJobImpl) Execute() error {
	j.executions++
	time.Sleep(j.runTime)
	if j.failEvery > 0 && j.executions%j.failEvery == 0 {
		j.status = fmt.Errorf("FAIL")
		j.wasFailed = true
		go func() {
			time.Sleep(j.failTime)
			j.status = nil
			j.wasRestored = true
		}()
	}
	return nil
}

func (j *intervalJobImpl) Status() error {
	return j.status
}

func (j *intervalJobImpl) Ready() bool {
	return j.ready
}

func TestIntervalJob(t *testing.T) {
	job := &intervalJobImpl{
		name:    "IntervalJob",
		runTime: 5 * time.Millisecond,
		ready:   false,
	}

	jobID, _ := RegisterIntervalJob(job, time.Millisecond)

	status := GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusInitializing], status)
	job.ready = true
	time.Sleep(30 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRunning], status)
	time.Sleep(100 * time.Millisecond) // Let the executions continue
	globalPool.cronJobs[jobID].stop()
	time.Sleep(20 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusStopped], status)
	assert.GreaterOrEqual(t, job.executions, 2)
}
