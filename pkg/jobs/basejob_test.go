package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type singleJobImpl struct {
	Job
	name    string
	runTime time.Duration
	ready   bool
}

func (j *singleJobImpl) Execute() error {
	time.Sleep(j.runTime)
	return nil
}

func (j *singleJobImpl) Status() error {
	return nil
}

func (j *singleJobImpl) Ready() bool {
	return j.ready
}

func TestSingleRunJob(t *testing.T) {
	job := &singleJobImpl{
		name:    "SingleJob",
		runTime: 50 * time.Millisecond,
		ready:   false,
	}

	jobID, _ := RegisterSingleRunJob(job)

	time.Sleep(10 * time.Millisecond)
	status := GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusInitializing], status)
	job.ready = true
	time.Sleep(10 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusRunning], status)
	time.Sleep(50 * time.Millisecond)
	status = GetJobStatus(jobID)
	assert.Equal(t, jobStatusToString[JobStatusFinished], status)
}
