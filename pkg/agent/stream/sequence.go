package stream

import (
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
)

// SequenceIDKey - the cache key name for watch sequence IDs
const SequenceIDKey = "watchSequenceID"

// AgentSequenceManager - represents the sequence manager for an agent
type AgentSequenceManager struct {
	sequenceCache cache.Cache
}

// GetSequence - return the watch sequenceID
func (s *AgentSequenceManager) GetSequence() int64 {
	if s.sequenceCache != nil {
		cachedSeqID, err := s.sequenceCache.Get(SequenceIDKey)
		if err == nil {
			if seqID, ok := cachedSeqID.(float64); ok {
				return int64(seqID)
			}
		}
	}
	return 0
}

//GetAgentSequenceManager -
func GetAgentSequenceManager(watchTopicName string) *AgentSequenceManager {
	seqCache := cache.New()
	if watchTopicName != "" {
		err := seqCache.Load(watchTopicName + ".sequence")
		if err != nil {
			seqCache.Set(SequenceIDKey, int64(0))
			if util.IsNotTest() {
				seqCache.Save(watchTopicName + ".sequence")
			}
		}
	}
	return &AgentSequenceManager{sequenceCache: seqCache}
}

// GetCache - return sequence cache
func (s *AgentSequenceManager) GetCache() cache.Cache {
	return s.sequenceCache
}
