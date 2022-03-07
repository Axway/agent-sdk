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
		message string
		propKey string
		propVal string
		props   map[string]interface{}
	}{
		{
			name:    "Build Success Status",
			message: "message",
			propKey: "key",
			propVal: "val",
			success: true,
		},
		{
			name:    "Build Failed Status",
			message: "message",
			propKey: "key",
			propVal: "val",
			props:   map[string]interface{}{"a": "b"},
			success: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := provisioning.NewRequestStatusBuilder().SetMessage("").AddProperty(tt.propKey, tt.propVal)
			if tt.props != nil {
				builder.SetProperties(tt.props)
			}

			var req provisioning.RequestStatus
			if tt.success {
				req = builder.Success()
			} else {
				req = builder.Failed()
			}

			assert.NotNil(t, req)
		})
	}
}
