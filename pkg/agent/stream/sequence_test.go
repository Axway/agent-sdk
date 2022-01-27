package stream

import (
	"testing"

	agentcache "github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/config"
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
			name:   "should have matching values, no error",
			hasErr: false,
			key:    "watchTopicName_0",
			want:   0,
		},
		{
			name:   "should have matching values, no error",
			hasErr: false,
			key:    "watchTopicName_200",
			want:   200,
		},
		{
			name:   "should have incorrect Key and return default value: 0, has NO error",
			hasErr: false,
			key:    "wrongKey1",
			setVal: 102,
			want:   0,
		},
	}
	cacheManager := agentcache.NewAgentCacheManager(&config.CentralConfiguration{}, false)
	cacheManager.AddSequence("watchTopicName_0", 0)
	cacheManager.AddSequence("watchTopicName_200", 200)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			s := &agentSequenceManager{
				cacheManager,
				tt.key,
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
