package jobs

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type retryJobImpl struct {
	Job
	name    string
	runTime time.Duration
	fails   int
	ready   bool
}

func (j *retryJobImpl) Execute() error {
	if j.fails > 0 {
		j.fails--
		return fmt.Errorf("fail")
	}
	time.Sleep(j.runTime)
	return nil
}

func (j *retryJobImpl) Status() error {
	return nil
}

func (j *retryJobImpl) Ready() bool {
	return j.ready
}

func TestRetryJob(t *testing.T) {
	job := &retryJobImpl{
		name:    "RetryJob",
		runTime: 50 * time.Millisecond,
		fails:   2,
		ready:   false,
	}

	jobID, _ := RegisterRetryJob(job, 3)
	globalPool.jobs[jobID].(*retryJob).backoff = newBackoffTimeout(time.Millisecond, time.Millisecond, 1)

	time.Sleep(10 * time.Millisecond)
	status := GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusInitializing], status)
	job.ready = true
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRetrying], status)
	time.Sleep(50 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusFinished], status)
}
