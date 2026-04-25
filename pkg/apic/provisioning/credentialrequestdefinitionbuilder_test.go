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
				IsRenewable().
				IsSuspendable().
				SetExpirationDays(90).
				SetDeprovisionExpired().
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

func TestSetIdentityProvider(t *testing.T) {
	tests := []struct {
		name     string
		idpName  string
		expected string
	}{
		{name: "set identity provider name", idpName: "my-idp-resource", expected: "my-idp-resource"},
		{name: "empty identity provider name", idpName: "", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedSpec management.CredentialRequestDefinitionSpec
			registerFunc := func(crd *management.CredentialRequestDefinition) (*management.CredentialRequestDefinition, error) {
				capturedSpec = crd.Spec
				return crd, nil
			}

			_, err := NewCRDBuilder(registerFunc).
				SetName("test-crd").
				SetProvisionSchema(
					NewSchemaBuilder().SetName("s").AddProperty(NewSchemaPropertyBuilder().SetName("p").IsString())).
				SetRequestSchema(
					NewSchemaBuilder().SetName("s").AddProperty(NewSchemaPropertyBuilder().SetName("p").IsString())).
				SetIdentityProvider(tc.idpName).
				Register()

			assert.Nil(t, err)
			assert.Equal(t, tc.expected, capturedSpec.IdentityProvider)
		})
	}
}
