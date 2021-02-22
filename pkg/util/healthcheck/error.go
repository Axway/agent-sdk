package healthcheck

import "github.com/Axway/agent-sdk/pkg/util/errors"

//Healthcheck errors
var (
	ErrStartingPeriodicHealthCheck = errors.New(1611, "error starting periodic healthcheck")
)
