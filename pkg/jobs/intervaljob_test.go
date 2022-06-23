package jobs

import (
	"context"
	"sync"
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
	jobMutex    *sync.Mutex
	readyMutex  *sync.Mutex
	statusMutex *sync.Mutex
}

func (j *intervalJobImpl) Execute() error {
	j.incrementExecutions()
	time.Sleep(j.runTime)
	if j.failEvery > 0 && j.getExecutions()%j.failEvery == 0 {
		j.setWasFailed(true)
		go func() {
			time.Sleep(j.failTime)
			j.setStatus(nil)
			j.setWasRestored(true)
		}()
	}
	return nil
}

func (j *intervalJobImpl) Status() error {
	j.statusMutex.Lock()
	defer j.statusMutex.Unlock()
	return j.status
}

func (j *intervalJobImpl) setStatus(status error) {
	j.statusMutex.Lock()
	defer j.statusMutex.Unlock()
	j.status = status
}

func (j *intervalJobImpl) Ready() bool {
	j.readyMutex.Lock()
	defer j.readyMutex.Unlock()
	return j.ready
}

func (j *intervalJobImpl) setReady(ready bool) {
	j.readyMutex.Lock()
	defer j.readyMutex.Unlock()
	j.ready = ready
}

func (j *intervalJobImpl) getWasFailed() bool {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.wasFailed
}

func (j *intervalJobImpl) setWasFailed(wasFailed bool) {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.wasFailed = wasFailed
}

func (j *intervalJobImpl) getWasRestored() bool {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.wasRestored
}

func (j *intervalJobImpl) setWasRestored(wasRestored bool) {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.wasRestored = wasRestored
}

func (j *intervalJobImpl) getExecutions() int {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.executions
}

func (j *intervalJobImpl) incrementExecutions() {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.executions++
}

func (j *intervalJobImpl) clearExecutions() {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.executions = 0
}

func TestIntervalJob(t *testing.T) {
	job := &intervalJobImpl{
		name:        "IntervalJob",
		runTime:     5 * time.Millisecond,
		jobMutex:    &sync.Mutex{},
		statusMutex: &sync.Mutex{},
		readyMutex:  &sync.Mutex{},
		ready:       false,
	}

	jobID, _ := RegisterIntervalJob(job, time.Millisecond)

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
