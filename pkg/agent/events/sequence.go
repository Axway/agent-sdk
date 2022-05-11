package events

import (
	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
)

// SequenceProvider - Interface to provide event sequence ID to harvester client to fetch events
type SequenceProvider interface {
	GetSequence() int64
	SetSequence(sequenceID int64)
}

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

// NewSequenceProvider creates a new SequenceProvider
func NewSequenceProvider(cacheManager agentcache.Manager, watchTopicName string) SequenceProvider {
	return &agentSequenceManager{
		cacheManager:   cacheManager,
		watchTopicName: watchTopicName,
	}
}
