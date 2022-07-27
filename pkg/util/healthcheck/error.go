package healthcheck

import "github.com/Axway/agent-sdk/pkg/util/errors"

//Healthcheck errors
var (
	ErrAlreadyRunning = errors.New(1613, "terminating agent, another instance of agent already running")
)
