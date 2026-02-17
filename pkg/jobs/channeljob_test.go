package jobs

import (
	"context"
	"sync"
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
	jobMutex   *sync.Mutex
	readyMutex *sync.Mutex
}

func (j *channelJobImpl) Execute() error {
	j.incrementExecutions()
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
	j.readyMutex.Lock()
	defer j.readyMutex.Unlock()
	return j.ready
}

func (j *channelJobImpl) setReady(ready bool) {
	j.readyMutex.Lock()
	defer j.readyMutex.Unlock()
	j.ready = ready
}

func (j *channelJobImpl) getExecutions() int {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	return j.executions
}

func (j *channelJobImpl) incrementExecutions() {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.executions++
}

func (j *channelJobImpl) clearExecutions() {
	j.jobMutex.Lock()
	defer j.jobMutex.Unlock()
	j.executions = 0
}

func TestChannelJob(t *testing.T) {
	job := &channelJobImpl{
		name:       "ChannelJob",
		runTime:    1 * time.Second,
		ready:      false,
		stopChan:   make(chan interface{}),
		jobMutex:   &sync.Mutex{},
		readyMutex: &sync.Mutex{},
	}

	jobID, _ := RegisterChannelJob(job, job.stopChan)
	j, _ := globalPool.jobs.Load(jobID)
	j.(*channelJob).setBackoff(newBackoffTimeout(time.Millisecond, time.Millisecond, 1))

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

func TestDetachedChannelJob(t *testing.T) {
	job := &channelJobImpl{
		name:       "DetachedChannelJob",
		runTime:    5 * time.Millisecond,
		ready:      false,
		stopChan:   make(chan interface{}),
		jobMutex:   &sync.Mutex{},
		readyMutex: &sync.Mutex{},
	}

	jobID, _ := RegisterDetachedChannelJob(job, job.stopChan)
	assert.NotEmpty(t, jobID)
	j, _ := globalPool.detachedCronJobs.Load(jobID)
	assert.NotNil(t, j)

	j, _ = globalPool.cronJobs.Load(jobID)
	assert.Nil(t, j)
	UnregisterJob(jobID)
}
