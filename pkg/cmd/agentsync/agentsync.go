package agentsync

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

var agentSync AgentSync

func init() {
	agentSync = &defaultAgentSync{}
}

// AgentSync - the interface discovery agents implement to handle the sync command line flag
type AgentSync interface {
	ProcessSynchronization() error
}

type defaultAgentSync struct {
	AgentSync
}

func (d *defaultAgentSync) ProcessSynchronization() error {
	log.Warn("This is the default synchronization method")
	return nil
}

// SetAgentSync - allows the agent to set the agent sync implementation for its gateway
func SetAgentSync(agentSyncImpl AgentSync) {
	agentSync = agentSyncImpl
}
