package provisioning_test

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/stretchr/testify/assert"
)

func TestRequestStatusBuilder(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		wantErr bool
	}{
		{
			name:    "Build Success Status",
			success: true,
		},
		{
			name:    "Build Failed Status",
			success: false,
		},
		{
			name:    "Build Status 1 - error",
			success: true,
			wantErr: true,
		},
		{
			name:    "Build Status 2 - error",
			success: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := provisioning.NewRequestStatusBuilder()
			if tt.success {
				builder.Success()
			} else {
				builder.Failed("error")
			}

			if tt.wantErr {
				if tt.success {
					builder.Failed("error")
					builder.Success()
				} else {
					builder.Success()
					builder.Failed("error")
				}
			}

			req, err := builder.Process()
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, req)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, req)
			}
		})
	}
}
