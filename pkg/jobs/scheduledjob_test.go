package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type scheduledJobImpl struct {
	Job
	name       string
	runTime    time.Duration
	ready      bool
	executions int
}

func (j *scheduledJobImpl) Execute() error {
	j.executions++
	time.Sleep(j.runTime)
	return nil
}

func (j *scheduledJobImpl) Status() error {
	return nil
}

func (j *scheduledJobImpl) Ready() bool {
	return j.ready
}

func TestScheduledJob(t *testing.T) {
	job := &scheduledJobImpl{
		name:    "ScheduledJob",
		runTime: 5 * time.Millisecond,
		ready:   false,
	}

	// scheduled job with bad schedule
	_, err := RegisterScheduledJob(job, "@time")
	assert.NotNil(t, err, "expected an error with a bad schedule")

	// create a scheduled job that runs every second
	jobID, err := RegisterScheduledJob(job, "* * * * * * *")
	assert.Nil(t, err)

	status := GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusInitializing], status)
	job.ready = true
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRunning], status)
	time.Sleep(3 * time.Second) // Let the executions continue
	globalPool.cronJobs[jobID].stop()
	time.Sleep(1 * time.Second)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusStopped], status)
	assert.LessOrEqual(t, 3, job.executions)
}
