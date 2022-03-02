package provisioning_test

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/stretchr/testify/assert"
)

func TestCredentialBuilder(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		id     string
		secret string
		other  map[string]interface{}
	}{
		{
			name: "Build API Key Credential",
			key:  "api-key-data",
		},
		{
			name:   "Build OAuth Credential",
			id:     "client-id",
			secret: "secret",
		},
		{
			name: "Build Other Credential",
			other: map[string]interface{}{
				"data1": "data1",
				"data2": "data2",
			},
		},
		{
			name:   "Build Multiple Credential - error 1",
			key:    "api-key-data",
			id:     "client-id",
			secret: "secret",
		},
		{
			name: "Build Multiple Credential - error 2",
			key:  "api-key-data",
			other: map[string]interface{}{
				"data1": "data1",
				"data2": "data2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := false
			builder := provisioning.NewCredentialBuilder()
			if tt.key != "" {
				builder.SetAPIKey(tt.key)
			}
			if tt.id != "" {
				builder.SetOAuth(tt.key, tt.secret)
			}
			if tt.other != nil {
				builder.SetCredential(tt.other)
			}

			if (tt.key != "" && tt.id != "") || (tt.key != "" && tt.other != nil) {
				wantErr = true
				builder.SetAPIKey(tt.key)
			}

			cred, err := builder.Process()
			if wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, cred)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, cred)
			}
		})
	}
}
