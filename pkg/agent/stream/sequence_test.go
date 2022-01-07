package stream

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/stretchr/testify/assert"
)

func Test_GetSequence(t *testing.T) {
	tests := []struct {
		name          string
		hasErr        bool
		key           string
		setVal        float64
		sequenceCache cache.Cache
		want          int64
	}{
		{
			name:          "should have matching values, no error",
			hasErr:        false,
			key:           SequenceIDKey,
			sequenceCache: cache.New(),
			setVal:        0,
			want:          0,
		},
		{
			name:          "should NOT have matching values, has error",
			hasErr:        true,
			key:           SequenceIDKey,
			sequenceCache: cache.New(),
			setVal:        12,
			want:          10,
		},
		{
			name:          "should have matching values, no error",
			hasErr:        false,
			key:           SequenceIDKey,
			sequenceCache: cache.New(),
			setVal:        200,
			want:          200,
		},
		{
			name:          "should have incorrect Key and return default value: 0, has NO error",
			hasErr:        false,
			key:           "wrongKey",
			sequenceCache: cache.New(),
			setVal:        102,
			want:          0,
		},
		{
			name:          "should have incorrect Key and return default value: 0, has error",
			hasErr:        true,
			key:           "wrongKey2",
			sequenceCache: cache.New(),
			setVal:        40,
			want:          40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.sequenceCache.Set(tt.key, tt.setVal)
			s := &AgentSequenceManager{
				sequenceCache: tt.sequenceCache,
			}

			if got := s.GetSequence(); got != tt.want {
				if !tt.hasErr {
					t.Errorf("agentSequenceManager.GetSequence() = %v, want %v", got, tt.want)
				}
			} else {
				assert.Equal(t, got, tt.want)
			}
		})
	}
}

func TestGetAgentSequenceManager(t *testing.T) {
	mockWT := getMockWatchTopic()
	mockName := mockWT.GetName() //wt-name

	sm1 := GetAgentSequenceManager(mockName)
	sm2 := GetAgentSequenceManager(mockName)
	seqID1 := sm1.GetSequence()
	seqID2 := sm2.GetSequence()
	assert.Equal(t, seqID1, seqID2)

	// alter cache1 and verify sequence inequality until sync between seq managers
	sm1.GetCache().Set(SequenceIDKey, int64(4))
	seqID1 = sm1.GetSequence()
	seqID2 = sm2.GetSequence()
	assert.NotEqual(t, seqID1, seqID2)
	sm1 = GetAgentSequenceManager(mockName)
	sm2 = GetAgentSequenceManager(mockName)
	seqID1 = sm1.GetSequence()
	seqID2 = sm2.GetSequence()
	assert.Equal(t, seqID1, seqID2)

	// new watch topic name should return int64 0
	sm3 := GetAgentSequenceManager("mockName")
	seqID3 := sm3.GetSequence()
	assert.Equal(t, seqID3, int64(0))

	// different watchTopicNames return different seq mgrs
	assert.NotEqual(t, seqID1, seqID3)
}
