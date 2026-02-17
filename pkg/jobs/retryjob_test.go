package jobs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type retryJobImpl struct {
	Job
	name     string
	runTime  time.Duration
	fails    int
	ready    bool
	jobMutex *sync.Mutex
}

func (j *retryJobImpl) Execute() error {
	for j.fails > 0 {
		j.fails--
		time.Sleep(j.runTime)
		return fmt.Errorf("retry job failed")
	}

	return nil
}

func (j *retryJobImpl) Status() error {
	return nil
}

func (j *retryJobImpl) Ready() bool {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.ready
}

func (j *retryJobImpl) setReady(ready bool) {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.ready = ready
}
func TestRetryJob(t *testing.T) {
	job := &retryJobImpl{
		name:     "RetryJob",
		runTime:  500 * time.Millisecond,
		fails:    2,
		ready:    false,
		jobMutex: &sync.Mutex{},
	}

	jobID, _ := RegisterRetryJob(job, 3)
	j, _ := globalPool.jobs.Load(jobID)
	j.(*retryJob).setBackoff(newBackoffTimeout(time.Millisecond, time.Millisecond, 1))

	statuses := []JobStatus{JobStatusRunning, JobStatusRetrying, JobStatusFinished}
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFunc()

	testDone := make(chan interface{})

	go statusWaiter(ctx, t, statuses, jobID, testDone)

	job.setReady(true)
	<-testDone

	assert.Nil(t, ctx.Err())
	UnregisterJob(jobID)
}
