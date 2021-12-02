# SDK Jobs

The jobs library is used to coordinate tasks that run within an agent.  There are 4 job types, explained below, that may be used.  Single run, Retry, Interval, and Scheduled jobs.

The jobs library keeps track of all jobs that are registered and executes them appropriately.  Scheduled and Interval jobs are continuous jobs that execute more than once.  These
jobs are continuously executed according to their settings.  If any of these continuous jobs begin to fail the library will pause execution of all jobs until a running status is  
achieved again.

When using the jobs library, remember that the main process of the agent can not exit, otherwise all jobs will exit

## Job status

The following are the possible job status values that can be returned by the GetJobStatus method

| Status       | Definition                                                                              |
|--------------|-----------------------------------------------------------------------------------------|
| Initializing | Returned when the has been created but not yet started                                  |
| Running      | Returned when a job is being executed, or between executions in a working state         |
| Retrying     | Returned only for retry job types that return an error in a call to Execute             |
| Stopped      | Returned when a continuous job is in a non-working state and is waiting to be restarted |
| Failed       | Returned when a single run or retry job does not Execute without error                  |
| Finished     | Returned when a single run or retry job Executes properly                               |

## Implementing the job interface

Before registering a job, the Job interface has to be implemented for your job

```go
package main

import (
  github.com/Axway/agent-sdk/pkg/jobs
)

type MyJob struct {
  jobs.Job // implements interface
}

func (j *MyJob) Status() error {
  // continually called determining the status of any dependencies for the job
  // returning an error means the job should not be executed
}

func (j *MyJob) Ready() bool {
  // called prior to executing the job the first time
  // return true when the job can begin execution, false otherwise
}

func (j *MyJob) Execute() error {
  // called each time the job should be executed
  // returning an error stops continuous jobs from executing
}
```

## Job types

The section covers the following job types

- [Single run jobs](#single-run-jobs)
- [Retry jobs](#retry-jobs)
- [Interval jobs](#interval-jobs)
- [Detached interval jobs](#detached-interval-jobs)
- [Scheduled jobs](#scheduled-jobs)
- [Channel jobs](#channel-jobs)

### Single run jobs

Single run jobs are used to run a specific task exactly once, regardless of pass or fail.

#### Register Single run job

Register the job and get its status

```go
package main

import (
  "fmt"

  "github.com/Axway/agent-sdk/pkg/jobs"
)

func main() {
  myJob := MyJob{}
  jobID, err := jobs.RegisterSingleRunJobWithName(myJob, "My Job")
  if err != nil {
    panic(err) // error registering the job
  }
  fmt.Println(GetJobStatus(jobID))
}
```

### Retry jobs

Retry jobs are like single run jobs, but are allowed to retry a specified number of times before failing

#### Register Retry job

Register the job and get its status

```go
package main

import (
  "fmt"

  "github.com/Axway/agent-sdk/pkg/jobs"
)

func main() {
  myJob := MyJob{}
  retries := 3
  jobID, err := jobs.RegisterRetryJobWithName(myJob, retries, "My Job")
  if err != nil {
    panic(err) // error registering the job
  }
  fmt.Println(GetJobStatus(jobID))
}
```

### Interval jobs

Interval jobs are executed with a certain time duration between the end of one execution to the beginning of the next

#### Register Interval job

Register the job and get its status

```go
package main

import (
  "fmt"

  "github.com/Axway/agent-sdk/pkg/jobs"
)

func main() {
  myJob := MyJob{}
  interval := 30 * time.Second
  jobID, err := jobs.RegisterIntervalJobWithName(myJob, interval, "My Job")
  if err != nil {
    panic(err) // error registering the job
  }
  fmt.Println(GetJobStatus(jobID))
}
```

### Detached interval jobs

Detached interval jobs are just like Interval jobs except they are not stopped with other jobs fail, they run independent of the job pool

### Scheduled jobs

Scheduled jobs are executed on a certain time frame, the previous execution has to end prior to the next execution starting.

#### Defining a schedule

Scheduled jobs use a cronjob expressions to set up their schedule.  The fields are Seconds, Minutes, Hours, Day of month, Month,
Day of week, and Year.  There are also predefined expressions that may be used.

Cron expressions are defined with the above fields in a single string each field value separated by a space.

Allowed values

| Seconds | Minutes | Hour   | Day of month | Month  | Day of week | Year        |
|---------|---------|--------|--------------|--------|-------------|-------------|
| 0 - 59  | 0 - 59  | 0 - 23 | 1 - 31       | 1 - 12 | 0 - 6       | 1970 - 2099 |

All of the fields can also utilize the following characters within their schedule

- Asterick (*) - matches all values for this field
- Slash (/) - describes increments steps within the field
- Comma (,) - separates values within the field to match
- Hyphen (-) - defines a range of values within the field to match

Examples

| Expression                  | Description                                                                           |
|-----------------------------|---------------------------------------------------------------------------------------|
| \* \* \* \* \* \* \*        | Run every second                                                                      |
| 30 5-59/15 \* \* \* \* \*   | Run 30 seconds past the minute, starting at minute 5 and every 15 minutes there after |
| 0 0 1,5,9,15,21 \* \* \* \* | Run at hour 1, 5, 9, 15, and 21 of each day                                           |
| 0 0 0 \* \* 6 \*            | Run at midnight every Saturday                                                        |

Predefined expressions

| Expression | Description                                 | Equivalent         |
|------------|---------------------------------------------|--------------------|
| @hourly    | Run at the top of the hour                  | 0 0 \* \* \* \* \* |
| @daily     | Run at midnight every day                   | 0 0 0 \* \* \* \*  |
| @weekly    | Run at midnight on Sundays                  | 0 0 0 \* \* 0 \*   |
| @monthly   | Run at midnight on the first of every month | 0 0 0 1 \* \* \*   |

#### Register Scheduled job

Register the job and get its status

```go
package main

import (
  "fmt"

  "github.com/Axway/agent-sdk/pkg/jobs"
)

func main() {
  myJob := MyJob{}
  runHalfPastHour := "0 30 * * * * *"
  jobID, err := jobs.RegisterScheduledJobWithName(myJob, runHalfPastHour, "My Job")
  if err != nil {
    panic(err) // error registering the job
  }
  fmt.Println(GetJobStatus(jobID))
}
```

### Channel jobs

Channel jobs are executed a single time via a go routine.  It is expected that the channel job runs forever.  

To allow this job to be stopped and started the implementer will provide a channel that the jobs pool can use to tell the job to stop.

#### Channel job implementation example

```go
package myjob

import "github.com/Axway/agent-sdk/pkg/jobs"

type MyJob struct {
  jobs.Job
  workChannel chan interface{}
  stopChannel chan interface{}
}

func NewMyJob(stopChannel chan interface{}) *MyJob {
  return &MyJob{
    stopChannel: stopChannel,
    workChannel: make(chan interface{}),
  }
}

func (m *MyJob) Ready() bool {
  // validate all dependencies are ready
  return true
}

func (m *MyJob) Status() error {
  // check that the job is still running without error
  return nil
}

func (m *MyJob) Execute() error {
  for {
    select {
    case <-m.stopChannel:
      //stop the job
      return nil
    case msg := <-workChannel:
      //do work on msg
  }
  }
}

```

#### Register Channel job

Register the job and get its status

```go
package main

import (
  "fmt"

  "github.com/Axway/agent-sdk/pkg/jobs"
)

func main() {
  stopChannel := make(chan interface{})
  myJob := NewMyJob(stopChannel)
  jobID, err := jobs.RegisterChannelJobWithName(myJob, stopChannel, "My Job")
  if err != nil {
    panic(err) // error registering the job
  }
  fmt.Println(GetJobStatus(jobID))
}
```

## Job locks

All continuous jobs (Interval and Scheduled) create locks that the agent can use to prevent the job from running at the same time as another process or job.
The job will lock itself prior to calling its Execute function and unlock itself after Execute has finished

Here is an example of how to create 2 jobs that can not execute at the same time.

```go
package main

import (
  github.com/Axway/agent-sdk/pkg/jobs
)

type FirstJob struct {
  jobs.Job // implements interface
}

func (j *FirstJob) Status() error {
  ...
}

func (j *FirstJob) Ready() bool {
  ...
}

func (j *FirstJob) Execute() error {
  ...
}

type SecondJob struct {
  jobs.Job // implements interface
  firstJobID string
}

func (j *SecondJob) Status() error {
  ...
}

func (j *SecondJob) Ready() bool {
  ...
}

func (j *SecondJob) Execute() error {
  jobs.JobLock(j.firstJobID)
  defer jobs.JobUnlock(j.firstJobID)
  ...
}

func main() {
  myFirstJob := FirstJob{}
  jobID, err := jobs.RegisterIntervalJob(myFirstJob, 30 * time.Second)
  if err != nil {
    panic(err) // error registering the job
  }

  mySecondJob := jobID{
    firstJobID: jobID,
  }
  _, err := jobs.RegisterIntervalJob(mySecondJob, 30 * time.Second)
  if err != nil {
    panic(err) // error registering the job
  }
}
```
