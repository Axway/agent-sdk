package resource

import "github.com/Axway/agent-sdk/pkg/util/errors"

// Errors related to agent resource
var (
	ErrUnsupportedAgentType = errors.New(1000, "unsupported agent type")
)
