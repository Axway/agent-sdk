package jobs

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobLocks(t *testing.T) {
	failJob := &intervalJobImpl{
		name:        "FailedIntervalJob",
		runTime:     50 * time.Millisecond,
		ready:       true,
		jobMutex:    &sync.Mutex{},
		statusMutex: &sync.Mutex{},
		readyMutex:  &sync.Mutex{},
	}
	jobID, _ := RegisterIntervalJob(failJob, 10*time.Millisecond)

	// The job is running, waiting 120 milliseconds before locking
	time.Sleep(120 * time.Millisecond)
	JobLock(jobID) // Lock the job
	curExecutions := failJob.executions

	// sleep another 120 milliseconds and validate the job has not continued
	time.Sleep(120 * time.Millisecond)
	newExecutions := failJob.executions
	assert.Equal(t, curExecutions, newExecutions, "The job ran more executions after locking")

	// Unlock the job then sleep another 120 milliseconds to check more executions have happened
	JobUnlock(jobID) // Unlock the job
	time.Sleep(120 * time.Millisecond)
	newExecutions = failJob.getExecutions()
	assert.Greater(t, newExecutions, curExecutions, "The job did not run more executions after unlocking")

	// Run test again using the jobExecution to get the locks
	// Get the job
	jobExecution := GetJob(jobID)

	// The job is running, waiting 120 milliseconds before locking
	time.Sleep(120 * time.Millisecond)
	jobExecution.Lock() // Lock the job
	curExecutions = failJob.executions

	// sleep another 120 milliseconds and validate the job has not continued
	time.Sleep(120 * time.Millisecond)
	newExecutions = failJob.executions
	assert.Equal(t, curExecutions, newExecutions, "The job ran more executions after locking")

	// Unlock the job then sleep another 120 milliseconds to check more executions have happened
	jobExecution.Unlock() // Unlock the job
	time.Sleep(120 * time.Millisecond)
	newExecutions = failJob.getExecutions()
	assert.Greater(t, newExecutions, curExecutions, "The job did not run more executions after unlocking")
}
