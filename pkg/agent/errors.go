package agent

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors hit when validating AMPLIFY Central connectivity
var (
	ErrUnsupportedAgentType         = errors.New(1000, "unsupported agent type")
	ErrStartingPeriodicStatusUpdate = errors.New(1001, "error starting periodic status update")
)
