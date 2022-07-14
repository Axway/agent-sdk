package provisioning

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNewCredentialRequestBuilder(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Success",
			wantErr: false,
		},
		{
			name:    "Fail",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerFuncCalled := false
			registerFunc := func(credentialRequestDefinition *management.CredentialRequestDefinition) (*management.CredentialRequestDefinition, error) {
				assert.NotNil(t, credentialRequestDefinition)
				assert.Len(t, credentialRequestDefinition.Spec.Provision.Schema["properties"], 1)
				assert.Len(t, credentialRequestDefinition.Spec.Schema["properties"], 1)
				assert.NotNil(t, credentialRequestDefinition.Spec.Provision.Schema["properties"].(map[string]interface{})["prop"])
				assert.NotNil(t, credentialRequestDefinition.Spec.Schema["properties"].(map[string]interface{})["prop"])
				registerFuncCalled = true
				return nil, nil
			}

			builder := NewCRDBuilder(registerFunc).
				SetName(tt.name).
				SetProvisionSchema(
					NewSchemaBuilder().
						SetName("schema").
						AddProperty(
							NewSchemaPropertyBuilder().
								SetName("prop").
								IsString())).
				SetRequestSchema(
					NewSchemaBuilder().
						SetName("schema").
						AddProperty(
							NewSchemaPropertyBuilder().
								SetName("prop").
								IsString())).
				SetWebhooks([]string{"webhook1", "webhook2"}).
				AddWebhook("webhook3")

			if tt.wantErr {
				builder = builder.SetProvisionSchema(nil)
			}
			_, err := builder.Register()

			if tt.wantErr {
				assert.NotNil(t, err)
				assert.False(t, registerFuncCalled)
			} else {
				assert.Nil(t, err)
				assert.True(t, registerFuncCalled)
			}
		})
	}
}
