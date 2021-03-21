package jobs

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrRegisteringJob    = errors.Newf(1600, "%v job registration failed")
	ErrExecutingJob      = errors.Newf(1601, "Error in %v job %v execution")
	ErrExecutingRetryJob = errors.Newf(1602, "Error in %v job %v execution, %v more retries")
)
