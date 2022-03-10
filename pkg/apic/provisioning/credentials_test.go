package provisioning_test

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/stretchr/testify/assert"
)

func TestCredentialBuilder(t *testing.T) {
	t.Skip()

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := provisioning.NewCredentialBuilder()

			var cred provisioning.Credential
			switch {
			case tt.key != "":
				cred = builder.SetAPIKey(tt.key)
			case tt.id != "":
				cred = builder.SetOAuth(tt.key, tt.secret)
			case tt.other != nil:
				cred = builder.SetCredential(tt.other)
			}

			assert.NotNil(t, cred)
		})
	}
}
