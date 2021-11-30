package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPoolCoordination(t *testing.T) {
	testPool := newPool() // create a new pool for this test to not interfere with other tests
	testPool.backoff = newBackoffTimeout(time.Millisecond, time.Millisecond, 1)
	testPool.retryInterval = time.Second
	failJob := &intervalJobImpl{
		name:      "FailedIntervalJob",
		runTime:   500 * time.Millisecond,
		ready:     true,
		failEvery: 3,
		failTime:  50 * time.Millisecond,
	}
	testPool.RegisterIntervalJob(failJob, 10*time.Millisecond)

	sJob := &scheduledJobImpl{
		name:    "ScheduledJob",
		runTime: time.Millisecond,
		ready:   true,
	}
	testPool.RegisterScheduledJob(sJob, "* * * * * * *")

	iJob := &intervalJobImpl{
		name:    "IntervalJob",
		runTime: time.Millisecond,
		ready:   true,
	}
	testPool.RegisterIntervalJob(iJob, 10*time.Millisecond)

	cJob := &channelJobImpl{
		name:     "ChannelJob",
		runTime:  time.Millisecond,
		ready:    true,
		stopChan: make(chan interface{}),
	}
	testPool.RegisterChannelJob(cJob, cJob.stopChan)

	diJob := &intervalJobImpl{
		name:    "DetachedIntervalJob",
		runTime: time.Millisecond,
		ready:   true,
	}
	testPool.RegisterDetachedIntervalJob(diJob, 10*time.Millisecond)

	time.Sleep(time.Second) // give enough time for scheduled job to run at least once

	// continue to get pool status to check that it was in a stopped state during test
	wasStopped := false
	stoppedThenStarted := false
	for i := 0; i < 200; i++ {
		if !wasStopped && testPool.GetStatus() == PoolStatusStopped.String() {
			wasStopped = true
			assert.GreaterOrEqual(t, sJob.executions, 1, "The scheduled job did not run at least once before failure")
			sJob.executions = 0
			assert.GreaterOrEqual(t, iJob.executions, 1, "The interval job did not run at least once before failure")
			iJob.executions = 0
			assert.GreaterOrEqual(t, failJob.executions, 1, "The failing interval did not run at least once before failure")
			failJob.executions = 0
			assert.GreaterOrEqual(t, diJob.executions, 1, "The detached interval job did not run at least once before other jobs were stopped")
			diJob.executions = 0
			assert.GreaterOrEqual(t, cJob.executions, 1, "The channel job did not run at least once before failure")
			iJob.executions = 0
		}
		if wasStopped && testPool.GetStatus() == PoolStatusRunning.String() {
			stoppedThenStarted = true
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(2 * time.Second) // give enough time for scheduled job to run at least once more

	assert.GreaterOrEqual(t, sJob.executions, 1, "The scheduled job did not run at least once after failure")
	assert.GreaterOrEqual(t, iJob.executions, 1, "The interval job did not run at least once after failure")
	assert.GreaterOrEqual(t, failJob.executions, 1, "The failing interval did not run at least once after failure")
	assert.GreaterOrEqual(t, diJob.executions, 1, "The detached interval did not run at least once after failure")
	assert.GreaterOrEqual(t, cJob.executions, 1, "The channel did not run at least once after failure")
	assert.True(t, wasStopped, "The pool status never showed as stopped")
	assert.True(t, stoppedThenStarted, "The pool status never restarted after it was stopped")
	assert.True(t, failJob.wasFailed, "The fail job never reported as failed")
	assert.True(t, failJob.wasRestored, "The fail job was not restored after failure")
}
