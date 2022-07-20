package provisioning_test

import (
	"testing"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/stretchr/testify/assert"
)

func TestRequestStatusBuilder(t *testing.T) {
	tests := []struct {
		name        string
		success     bool
		message     string
		propKey     string
		propVal     string
		prevReasons []v1.ResourceStatusReason
		props       map[string]string
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
			props:   map[string]string{"a": "b"},
			prevReasons: []v1.ResourceStatusReason{
				{
					Type:   "Error",
					Detail: "detail",
					Meta: map[string]interface{}{
						"action": "CredentialExpired",
					},
				},
			},
			success: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := provisioning.NewRequestStatusBuilder().SetCurrentStatusReasons(tt.prevReasons).SetMessage("").AddProperty(tt.propKey, tt.propVal)
			if tt.props != nil {
				builder.SetProperties(tt.props)
			}

			var req provisioning.RequestStatus
			if tt.success {
				req = builder.Success()
			} else {
				req = builder.Failed()
			}

			assert.EqualValues(t, tt.prevReasons, req.GetReasons())
			if tt.props != nil {
				assert.EqualValues(t, tt.props, req.GetProperties())
			} else {
				assert.EqualValues(t, map[string]string{tt.propKey: tt.propVal}, req.GetProperties())
			}
			assert.NotNil(t, req)

			newReq := provisioning.NewStatusReason(req)
			assert.EqualValues(t, len(tt.prevReasons)+1, len(newReq.Reasons))
			assert.NotNil(t, req)
		})
	}
}
