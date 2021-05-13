package healthcheck

import "github.com/Axway/agent-sdk/pkg/util/errors"

//Healthcheck errors
var (
	ErrStartingPeriodicHealthCheck = errors.New(1611, "error starting periodic healthcheck")
	ErrMaxconsecutiveErrors        = errors.Newf(1612, "healthchecks failed %v consecutive times, pausing execution")
	ErrAlreadyRunning              = errors.New(1613, "terminating agent, another instance of agent already running")
)
