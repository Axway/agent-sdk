package jobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type singleJobImpl struct {
	Job
	name      string
	runTime   time.Duration
	ready     bool
	readyLock *sync.Mutex
}

func (j *singleJobImpl) Execute() error {
	time.Sleep(j.runTime)
	return nil
}

func (j *singleJobImpl) Status() error {
	return nil
}

func (j *singleJobImpl) Ready() bool {
	j.readyLock.Lock()
	defer j.readyLock.Unlock()
	return j.ready
}

func (j *singleJobImpl) setReady(ready bool) {
	j.readyLock.Lock()
	defer j.readyLock.Unlock()
	j.ready = ready
}

func statusWaiter(ctx context.Context, t *testing.T, statuses []JobStatus, jobID string, doneChan chan interface{}) {
	for _, status := range statuses {
		for {
			select {
			case <-ctx.Done():
				assert.Fail(t, "did not get all statuses")
				doneChan <- nil
				return
			default:
			}
			curStat := GetJobStatus(jobID)
			if curStat == jobStatusToString[status] {
				break
			}
		}
	}

	doneChan <- nil
}

func TestSingleRunJob(t *testing.T) {
	job := &singleJobImpl{
		name:      "SingleJob",
		runTime:   1 * time.Second,
		ready:     false,
		readyLock: &sync.Mutex{},
	}

	jobID, _ := RegisterSingleRunJob(job)
	j, _ := globalPool.jobs.Load(jobID)
	j.(*baseJob).setBackoff(newBackoffTimeout(time.Millisecond, time.Millisecond, 1))

	statuses := []JobStatus{JobStatusRunning, JobStatusFinished}
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFunc()

	testDone := make(chan interface{})

	go statusWaiter(ctx, t, statuses, jobID, testDone)

	job.setReady(true)
	<-testDone

	assert.Nil(t, ctx.Err())
	UnregisterJob(jobID)
}
