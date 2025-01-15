package provisioning

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNewQuotaFromAccessRequest(t *testing.T) {
	tests := []struct {
		name           string
		limit          int32
		intervalString string
		interval       QuotaInterval
		wantNil        bool
	}{
		{
			name:    "no quota on access request",
			wantNil: true,
		},
		{
			name:           "bad interval string",
			intervalString: "bad",
			wantNil:        true,
		},
		{
			name:           "good quota",
			intervalString: "weekly",
			interval:       Weekly,
			limit:          100,
			wantNil:        false,
		},
		{
			name:           "good hourly quota",
			intervalString: "hourly",
			interval:       Hourly,
			limit:          10,
			wantNil:        false,
		},
		{
			name:           "good every minute quota",
			intervalString: "minutes",
			interval:       Minute,
			limit:          1,
			wantNil:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ar := management.NewAccessRequest("name", "environment")
			if tt.intervalString != "" {
				ar.Spec.Quota = &management.AccessRequestSpecQuota{
					Limit:    tt.limit,
					Interval: tt.intervalString,
				}
			}

			quota := NewQuotaFromAccessRequest(ar)
			if tt.wantNil {
				assert.Nil(t, quota)
				return
			}

			assert.Equal(t, quota.GetIntervalString(), tt.intervalString)
			assert.Equal(t, quota.GetInterval(), tt.interval)
			assert.Equal(t, quota.GetLimit(), int64(tt.limit))
		})
	}
}
