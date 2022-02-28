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
		ref    string
	}{
		{
			name: "Build API Key Credential",
			key:  "api-key-data",
			ref:  "creds",
		},
		{
			name:   "Build OAuth Credential",
			id:     "client-id",
			secret: "secret",
			ref:    "creds",
		},
		{
			name:   "Build Multiple Credential - error",
			key:    "api-key-data",
			id:     "client-id",
			secret: "secret",
			ref:    "creds",
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
			if tt.key != "" && tt.id != "" {
				wantErr = true
				builder.SetAPIKey(tt.key)
			}
			builder.SetCredentialReference(tt.ref)

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
