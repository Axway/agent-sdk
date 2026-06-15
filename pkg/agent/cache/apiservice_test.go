package cache

import (
	"testing"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

// createAPIServiceWithTime builds a ResourceInstance with a specific CreateTimestamp so that
// duplicate-detection logic (which compares timestamps) can be exercised deterministically.
func createAPIServiceWithTime(apiID, apiName, primaryKey string, ts time.Time) *v1.ResourceInstance {
	svc := createAPIService(apiID, apiName, primaryKey)
	svc.Metadata.Audit.CreateTimestamp = v1.Time(ts)
	return svc
}

// TestAddAPIServiceDuplicateDetection validates that AddAPIService keeps the older (original)
// ResourceInstance in the cache and discards the newer one when two entries share the same
// external API ID, covering both the no-primaryKey and the primaryKey storage paths.
func TestAddAPIServiceDuplicateDetection(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		primaryKey     string
		firstTS        time.Time // timestamp on the first-added entry
		secondTS       time.Time // timestamp on the second-added entry
		wantInCache    string    // Name of the ResourceInstance that should survive
		wantNotInCache string    // Name of the ResourceInstance that should be absent
	}{
		{
			name:           "no primaryKey - second is newer duplicate, original stays",
			primaryKey:     "",
			firstTS:        older,
			secondTS:       newer,
			wantInCache:    "name-api1-original",
			wantNotInCache: "name-api1-duplicate",
		},
		{
			name:           "no primaryKey - second is older original, duplicate evicted",
			primaryKey:     "",
			firstTS:        newer,
			secondTS:       older,
			wantInCache:    "name-api1-original",
			wantNotInCache: "name-api1-duplicate",
		},
		{
			name:           "with primaryKey - second is newer duplicate, original stays",
			primaryKey:     "pk1",
			firstTS:        older,
			secondTS:       newer,
			wantInCache:    "name-api1-original",
			wantNotInCache: "name-api1-duplicate",
		},
		{
			name:           "with primaryKey - second is older original, duplicate evicted",
			primaryKey:     "pk1",
			firstTS:        newer,
			secondTS:       older,
			wantInCache:    "name-api1-original",
			wantNotInCache: "name-api1-duplicate",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewAgentCacheManager(&config.CentralConfiguration{}, false)

			// The "first" entry is the one added earlier in the sequence.
			// Its Name encodes which role it plays so we can assert the right one survived.
			firstIsOriginal := tc.firstTS.Before(tc.secondTS)
			var firstEntry, secondEntry *v1.ResourceInstance
			if firstIsOriginal {
				firstEntry = createAPIServiceWithTime("id1", "api1", tc.primaryKey, tc.firstTS)
				firstEntry.Name = "name-api1-original"
				secondEntry = createAPIServiceWithTime("id1", "api1", tc.primaryKey, tc.secondTS)
				secondEntry.Name = "name-api1-duplicate"
			} else {
				firstEntry = createAPIServiceWithTime("id1", "api1", tc.primaryKey, tc.firstTS)
				firstEntry.Name = "name-api1-duplicate"
				secondEntry = createAPIServiceWithTime("id1", "api1", tc.primaryKey, tc.secondTS)
				secondEntry.Name = "name-api1-original"
			}

			err := m.AddAPIService(firstEntry)
			assert.NoError(t, err)
			err = m.AddAPIService(secondEntry)
			assert.NoError(t, err)

			// Exactly one entry should be in the cache.
			keys := m.GetAPIServiceKeys()
			assert.Len(t, keys, 1, "expected exactly one entry in cache after duplicate handling")

			// The surviving entry must be the original (older) one.
			survived := m.GetAPIServiceWithAPIID("id1")
			assert.NotNil(t, survived, "original APIService should be retrievable by apiID")
			assert.Equal(t, tc.wantInCache, survived.Name, "wrong entry survived")

			// The duplicate should be gone.
			allKeys := m.GetAPIServiceKeys()
			for _, k := range allKeys {
				item, _ := m.GetAPIServiceCache().Get(k)
				if ri, ok := item.(*v1.ResourceInstance); ok {
					assert.NotEqual(t, tc.wantNotInCache, ri.Name, "duplicate should not be in cache")
				}
			}
		})
	}
}
