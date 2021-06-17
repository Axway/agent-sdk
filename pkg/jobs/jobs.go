package jobs

import "time"

//globalPool - the default job pool
var globalPool *Pool

func init() {
	globalPool = newPool()
}

//RegisterSingleRunJob - Runs a single run job in the globalPool
func RegisterSingleRunJob(newJob Job) (string, error) {
	return globalPool.RegisterSingleRunJob(newJob)
}

//RegisterIntervalJob - Runs a job with a specific interval between each run in the globalPool
func RegisterIntervalJob(newJob Job, interval time.Duration) (string, error) {
	return globalPool.RegisterIntervalJob(newJob, interval)
}

//RegisterDetachedIntervalJob - Runs a job with a specific interval between each run in the globalPool, detached from other jobs to always run
func RegisterDetachedIntervalJob(newJob Job, interval time.Duration) (string, error) {
	return globalPool.RegisterDetachedIntervalJob(newJob, interval)
}

//RegisterScheduledJob - Runs a job on a specific schedule in the globalPool
func RegisterScheduledJob(newJob Job, schedule string) (string, error) {
	return globalPool.RegisterScheduledJob(newJob, schedule)
}

//RegisterRetryJob - Runs a job with a limited number of retries in the globalPool
func RegisterRetryJob(newJob Job, retries int) (string, error) {
	return globalPool.RegisterRetryJob(newJob, retries)
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
