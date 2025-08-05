package healthcheck

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/stretchr/testify/assert"
)

// TestStatusCheck tests the statusCheck struct
func TestStatusCheck(t *testing.T) {
	tests := []struct {
		name           string
		statusCheck    statusCheck
		checkerResult  *Status
		expectedStatus *Status
	}{
		{
			name: "successful health check",
			statusCheck: statusCheck{
				ID:       "test-id-1",
				Name:     "test-check-1",
				Endpoint: "test-endpoint-1",
				Status:   &Status{},
				logger:   log.NewFieldLogger(),
				checker: func(name string) *Status {
					return &Status{Result: OK}
				},
			},
			checkerResult:  &Status{Result: OK},
			expectedStatus: &Status{Result: OK},
		},
		{
			name: "failed health check",
			statusCheck: statusCheck{
				ID:       "test-id-2",
				Name:     "test-check-2",
				Endpoint: "test-endpoint-2",
				Status:   &Status{},
				logger:   log.NewFieldLogger(),
				checker: func(name string) *Status {
					return &Status{Result: FAIL, Details: "test error"}
				},
			},
			checkerResult:  &Status{Result: FAIL, Details: "test error"},
			expectedStatus: &Status{Result: FAIL, Details: "test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test setStatus
			tt.statusCheck.setStatus(tt.expectedStatus)
			assert.Equal(t, tt.expectedStatus, tt.statusCheck.Status)

			// Test executeCheck
			initialStatus := &Status{}
			tt.statusCheck.Status = initialStatus
			tt.statusCheck.executeCheck()

			assert.Equal(t, tt.expectedStatus.Result, tt.statusCheck.Status.Result)
			assert.Equal(t, tt.expectedStatus.Details, tt.statusCheck.Status.Details)
		})
	}
}
