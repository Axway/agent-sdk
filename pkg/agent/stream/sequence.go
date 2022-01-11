package stream

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
)

// agentSequenceManager - represents the sequence manager for an agent
type agentSequenceManager struct {
	cacheManager   agentcache.Manager
	watchTopicName string
}

// GetSequence - return the watch sequenceID
func (s *agentSequenceManager) GetSequence() int64 {
	return s.cacheManager.GetSequence(s.watchTopicName)
}

// SetSequence - updates the sequenceID in the cache
func (s *agentSequenceManager) SetSequence(sequenceID int64) {
	s.cacheManager.AddSequence(s.watchTopicName, sequenceID)
}

func newAgentSequenceManager(cacheManager agentcache.Manager, watchTopicName string) *agentSequenceManager {
	return &agentSequenceManager{cacheManager: cacheManager, watchTopicName: watchTopicName}
}
