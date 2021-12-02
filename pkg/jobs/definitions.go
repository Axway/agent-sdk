package jobs

import "time"

const defaultRetryInterval = 30 * time.Second

//Job -  the job interface, users of this library need to implement these
type Job interface {
	Execute() error
	Status() error
	Ready() bool
}

//JobExecution - the wrapper interface for every type of job
//               controls the calling the methods defined in the Job interface
type JobExecution interface {
	GetStatus() JobStatus
	GetID() string
	GetName() string
	Ready() bool
	GetJob() JobExecution
	Lock()
	Unlock()
	start()
	stop()
	getConsecutiveFails() int
	updateStatus() JobStatus
}

//JobStatus - integer to represent the status of the job
type JobStatus int

const (
	//JobStatusInitializing - Initializing
	JobStatusInitializing JobStatus = iota
	//JobStatusRunning - Running
	JobStatusRunning
	//JobStatusRetrying - Retrying
	JobStatusRetrying
	//JobStatusStopped - Stopped
	JobStatusStopped
	//JobStatusFailed - Failed
	JobStatusFailed
	//JobStatusFinished - Finished
	JobStatusFinished
)

//statusToString - maps the PoolStatus integer to a string representation
var jobStatusToString = map[JobStatus]string{
	JobStatusInitializing: "Initializing",
	JobStatusRunning:      "Running",
	JobStatusRetrying:     "Retrying",
	JobStatusStopped:      "Stopped",
	JobStatusFailed:       "Failed",
	JobStatusFinished:     "Finished",
}

func (s JobStatus) String() string {
	return jobStatusToString[s]
}

//PoolStatus - integer to represent the status of the jobs in the pool
type PoolStatus int

const (
	//PoolStatusInitializing - Initializing
	PoolStatusInitializing PoolStatus = iota
	//PoolStatusRunning - Running
	PoolStatusRunning
	//PoolStatusStopped - Stopped
	PoolStatusStopped
)

//poolStatusToString - maps the PoolStatus integer to a string representation
var poolStatusToString = map[PoolStatus]string{
	PoolStatusInitializing: "Initializing",
	PoolStatusRunning:      "Running",
	PoolStatusStopped:      "Stopped",
}

func (s PoolStatus) String() string {
	return poolStatusToString[s]
}

// Job type strings
const (
	JobTypeSingleRun        = "single run"
	JobTypeRetry            = "retry"
	JobTypeInterval         = "interval"
	JobTypeChannel          = "channel"
	JobTypeDetachedInterval = "detached interval"
	JobTypeScheduled        = "scheduled"
)
