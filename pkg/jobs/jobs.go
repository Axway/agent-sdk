package jobs

import "time"

//globalPool - the default job pool
var globalPool *Pool
var executionTimeLimit time.Duration = 5 * time.Minute
var statusCheckInterval time.Duration = defaultRetryInterval

func init() {
	globalPool = newPool()
}

// UpdateDurations - updates settings int he jobs library
func UpdateDurations(retryInterval time.Duration, executionTimeout time.Duration) {
	executionTimeLimit = executionTimeout
	globalPool.backoff = newBackoffTimeout(retryInterval, 10*time.Minute, 2)
	statusCheckInterval = retryInterval
}

//RegisterSingleRunJob - Runs a single run job in the globalPool
func RegisterSingleRunJob(newJob Job) (string, error) {
	return globalPool.RegisterSingleRunJob(newJob)
}

//RegisterSingleRunJobWithName - Runs a single run job in the globalPool
func RegisterSingleRunJobWithName(newJob Job, name string) (string, error) {
	return globalPool.RegisterSingleRunJobWithName(newJob, name)
}

//RegisterIntervalJob - Runs a job with a specific interval between each run in the globalPool
func RegisterIntervalJob(newJob Job, interval time.Duration) (string, error) {
	return globalPool.RegisterIntervalJob(newJob, interval)
}

//RegisterIntervalJobWithName - Runs a job with a specific interval between each run in the globalPool
func RegisterIntervalJobWithName(newJob Job, interval time.Duration, name string) (string, error) {
	return globalPool.RegisterIntervalJobWithName(newJob, interval, name)
}

//RegisterChannelJob - Runs a job with a specific interval between each run in the globalPool
func RegisterChannelJob(newJob Job, stopChan chan interface{}) (string, error) {
	return globalPool.RegisterChannelJob(newJob, stopChan)
}

//RegisterChannelJobWithName - Runs a job with a specific interval between each run in the globalPool
func RegisterChannelJobWithName(newJob Job, stopChan chan interface{}, name string) (string, error) {
	return globalPool.RegisterChannelJobWithName(newJob, stopChan, name)
}

//RegisterDetachedIntervalJob - Runs a job with a specific interval between each run in the globalPool, detached from other jobs to always run
func RegisterDetachedIntervalJob(newJob Job, interval time.Duration) (string, error) {
	return globalPool.RegisterDetachedIntervalJob(newJob, interval)
}

//RegisterDetachedIntervalJobWithName - Runs a job with a specific interval between each run in the globalPool, detached from other jobs to always run
func RegisterDetachedIntervalJobWithName(newJob Job, interval time.Duration, name string) (string, error) {
	return globalPool.RegisterDetachedIntervalJobWithName(newJob, interval, name)
}

//RegisterScheduledJob - Runs a job on a specific schedule in the globalPool
func RegisterScheduledJob(newJob Job, schedule string) (string, error) {
	return globalPool.RegisterScheduledJob(newJob, schedule)
}

//RegisterScheduledJobWithName - Runs a job on a specific schedule in the globalPool
func RegisterScheduledJobWithName(newJob Job, schedule, name string) (string, error) {
	return globalPool.RegisterScheduledJobWithName(newJob, schedule, name)
}

//RegisterRetryJob - Runs a job with a WithName
func RegisterRetryJob(newJob Job, retries int) (string, error) {
	return globalPool.RegisterRetryJob(newJob, retries)
}

//RegisterRetryJobWithName - Runs a job with a limited number of retries in the globalPool
func RegisterRetryJobWithName(newJob Job, retries int, name string) (string, error) {
	return globalPool.RegisterRetryJobWithName(newJob, retries, name)
}

//UnregisterJob - Removes the specified job in the globalPool
func UnregisterJob(jobID string) {
	globalPool.UnregisterJob(jobID)
}

//JobLock - Locks the job, returns when the lock is granted
func JobLock(id string) {
	globalPool.JobLock(id)
}

//JobUnlock - Unlocks the job
func JobUnlock(id string) {
	globalPool.JobUnlock(id)
}

//GetStatus - Returns the status from the globalPool
func GetStatus() string {
	return globalPool.GetStatus()
}

//GetJob - Returns the Job based on the id from the globalPool
func GetJob(id string) JobExecution {
	return globalPool.GetJob(id)
}

//GetJobStatus - Returns the Status of the Job based on the id in the globalPool
func GetJobStatus(id string) string {
	return globalPool.GetJobStatus(id)
}
