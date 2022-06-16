package jobs

import (
	"context"
	"sync"
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
	jobMutex   *sync.Mutex
}

func (j *scheduledJobImpl) Execute() error {
	j.jobMutex.Lock()
	j.executions++
	j.jobMutex.Unlock()
	time.Sleep(j.runTime)
	return nil
}

func (j *scheduledJobImpl) Status() error {
	return nil
}

func (j *scheduledJobImpl) Ready() bool {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.ready
}

func (j *scheduledJobImpl) setReady(ready bool) {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.ready = ready
}

func (j *scheduledJobImpl) getExecutions() int {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.executions
}

func (j *scheduledJobImpl) clearExecutions() {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.executions = 0
}

func TestScheduledJob(t *testing.T) {
	job := &scheduledJobImpl{
		name:     "ScheduledJob",
		runTime:  5 * time.Millisecond,
		ready:    false,
		jobMutex: &sync.Mutex{},
	}

	// scheduled job with bad schedule
	_, err := RegisterScheduledJob(job, "@time")
	assert.NotNil(t, err, "expected an error with a bad schedule")

	// create a scheduled job that runs every second
	jobID, err := RegisterScheduledJob(job, "* * * * * * *")
	assert.Nil(t, err)

	statuses := []JobStatus{JobStatusRunning}
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFunc()

	testDone := make(chan interface{})

	go statusWaiter(ctx, t, statuses, jobID, testDone)

	job.setReady(true)
	<-testDone
	assert.Nil(t, ctx.Err())
	UnregisterJob(jobID)
}
