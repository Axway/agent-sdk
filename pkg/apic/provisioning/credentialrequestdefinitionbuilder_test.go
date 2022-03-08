package provisioning

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
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
			// builtDef := struct{}{}
			registerFunc := func(credentialRequestDefinition *v1alpha1.CredentialRequestDefinition) (*v1alpha1.CredentialRequestDefinition, error) {
				// TODO - validate that the credentialRequestDefinition is built properly
				// builtDef = credentialRequestDefinition.(struct{})
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
				SetMaxApplicationCredentials(1).
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
				// assert.NotNil(t, builtDef)
			}
		})
	}
}
